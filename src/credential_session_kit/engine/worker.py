"""Isolated existing-account password/TOTP login. Never runs the registration driver."""
import contextlib
from pathlib import Path
import sys
_ENGINE = Path(__file__).resolve().parent
if str(_ENGINE) not in sys.path:
    sys.path.insert(0, str(_ENGINE))
from email.utils import parsedate_to_datetime
import json
import logging
import os
import re
import sys
import time
from urllib.parse import urljoin, urlsplit


class LoginFailure(Exception):
    pass


def check_response(response):
    # Return only fixed categories; never expose response bodies, cookies or URLs.
    status = response.status_code
    if response.headers.get('cf-mitigated', '').lower() == 'challenge':
        raise LoginFailure('web_challenge')
    code = {403: 'http_403', 429: 'http_429', 407: 'proxy_auth'}.get(status)
    if code:
        raise LoginFailure(code)
    response.raise_for_status()


def redirect_error_code(url, response):
    """Return a stable, non-sensitive diagnostic for a redirect response.

    The redirect URL is never returned to the UI.  Splitting the 403 cases by
    endpoint is useful because a callback rejection and a normal ChatGPT page
    rejection have different remediation paths, while both must stay pinned to
    the same session and proxy.
    """
    if response.headers.get('cf-mitigated', '').lower() == 'challenge':
        return 'web_challenge'
    status = response.status_code
    if status == 403:
        host = (urlsplit(url).hostname or '').lower()
        path = (urlsplit(url).path or '').lower()
        if host == 'chatgpt.com' and path == '/api/auth/callback/openai':
            return 'callback_http_403'
        if host == 'chatgpt.com':
            return 'chatgpt_http_403'
        return 'redirect_http_403'
    return {407: 'proxy_auth', 429: 'http_429'}.get(status)


def redirect_headers(flow, url, referer):
    """Build browser-navigation headers for one manual redirect hop."""
    headers = flow._navigation_headers()
    headers['Referer'] = referer
    headers.pop('sec-fetch-user', None)
    try:
        headers['sec-fetch-site'] = (
            'same-origin'
            if urlsplit(url).netloc == urlsplit(referer).netloc
            else 'cross-site'
        )
    except Exception:
        headers['sec-fetch-site'] = 'cross-site'
    return headers


def normalized_transport_proxy(value):
    return 'socks5h://' + value[len('socks5://'):] if value.startswith('socks5://') else value


def require_fixed_proxy(flow, expected):
    """Fail closed if the shared account flow ever leaves the selected proxy."""
    configured = getattr(getattr(flow, 'config', None), 'proxy', None)
    if configured != (expected or None):
        raise LoginFailure('proxy_changed')
    proxies = getattr(flow.session, 'proxies', {})
    wanted = normalized_transport_proxy(expected) if expected else ''
    if not isinstance(proxies, dict) or proxies.get('https') != wanted or proxies.get('http') != wanted:
        raise LoginFailure('proxy_changed')


def through_selected_proxy(flow, expected, fn, *args, **kwargs):
    """Run one login operation and verify the configured proxy on both sides."""
    require_fixed_proxy(flow, expected)
    result = fn(*args, **kwargs)
    require_fixed_proxy(flow, expected)
    return result


def is_http_403(exc):
    """Recognize an upstream 403 without retaining or exposing its body."""
    status = getattr(getattr(exc, 'response', None), 'status_code', None)
    if status == 403:
        return True
    return bool(re.search(r'\b403\b', str(exc)))


def is_invalid_state(exc):
    status = getattr(getattr(exc, 'response', None), 'status_code', None)
    text = str(exc).lower()
    return status == 409 or bool(re.search(r'\b409\b|invalid[_ -]?state', text))


def response_epoch(exc):
    """Read only the HTTP Date header for TOTP clock-skew correction."""
    response = getattr(exc, 'response', None)
    headers = getattr(response, 'headers', None) or {}
    value = headers.get('Date') or headers.get('date')
    if not value:
        return None
    try:
        return parsedate_to_datetime(value).timestamp()
    except (TypeError, ValueError, OverflowError):
        return None


def submit_mfa_compat(flow, code, challenge_id):
    """Use the current MFA endpoint and only fall back on version errors.

    A 403 is an authentication or edge decision, not an endpoint-version
    signal. Retrying it against several paths would submit the same challenge
    repeatedly and makes the actual failure harder to diagnose.
    """
    primary = "https://auth.openai.com/api/accounts/mfa/verify"
    variants = (
        "https://auth.openai.com/api/accounts/mfa/challenge/"
        f"{challenge_id}/verify",
        "https://auth.openai.com/api/accounts/mfa/totp/verify",
    )
    try:
        return flow.submit_mfa_totp(code, challenge_id, endpoint=primary)
    except Exception as exc:
        status = getattr(getattr(exc, "response", None), "status_code", None)
        if status not in (404, 405):
            raise
        last = exc
        for endpoint in variants:
            try:
                return flow.submit_mfa_totp(code, challenge_id, endpoint=endpoint)
            except Exception as variant_exc:
                last = variant_exc
                variant_status = getattr(getattr(variant_exc, "response", None), "status_code", None)
                if variant_status not in (404, 405):
                    raise
        raise last


def verified_auth_session(flow, expected, retries=1):
    """Read the web session again when callback propagation is still settling."""
    last = None
    for attempt in range(retries + 1):
        result = through_selected_proxy(
            flow, expected, flow.get_auth_session, include_identity=True)
        last = result
        # The password/TOTP challenge and callback already authenticated this
        # fresh session. Some valid session responses omit user.email entirely;
        # AT + session presence is the reliable hand-off check.
        identity_ok = bool(result[0]) and bool(result[1])
        if identity_ok:
            return result
        if attempt < retries:
            time.sleep(1)
    return last


def recover_web_auth_session(flow, expected, stage):
    """Recover a missing web AT using the account-pool session dump path."""
    result = verified_auth_session(flow, expected, retries=0)
    if result[1]:
        return result
    # Match the browser account-pool fallback: a document navigation can
    # finish NextAuth cookie propagation even when the first API read was 200.
    navigate_home = getattr(flow, 'refresh_chatgpt_home', None)
    if callable(navigate_home):
        try:
            through_selected_proxy(flow, expected, navigate_home)
            result = verified_auth_session(flow, expected, retries=0)
        except Exception:
            # Continue through the remaining non-destructive recovery paths.
            pass
    if result[1]:
        return result
    # Match the account-pool NextAuth refresh: ?update + target-route headers.
    try:
        result = through_selected_proxy(
            flow, expected, flow.get_auth_session,
            include_identity=True, refresh=True)
    except TypeError:
        # Test doubles and older external workers may not expose refresh yet.
        pass
    if result[1]:
        return result
    dump = getattr(flow, 'fetch_client_auth_session_dump', None)
    if callable(dump):
        through_selected_proxy(flow, expected, dump, stage)
    result = verified_auth_session(flow, expected, retries=1)
    if result[1]:
        return result
    candidate = str(getattr(getattr(flow, 'result', None), 'access_token', '') or '').strip()
    if candidate:
        return result[0], candidate, result[2]
    return result


def session_diagnostic(flow):
    """Return structural session diagnostics without secret material."""
    value = getattr(flow, '_auth_session_diagnostic', {})
    if not isinstance(value, dict):
        return {}
    allowed = ('refresh', 'status', 'content_type', 'body_size', 'top_keys',
               'cookie_names', 'has_session_cookie', 'has_session_token',
               'has_access_token')
    return {key: value.get(key) for key in allowed if key in value}


def login_step_with_403_retry(flow, expected, fn, *args, retries=2, **kwargs):
    """Retry transient auth API 403s on the same session and proxy only."""
    for retry in range(retries + 1):
        try:
            return through_selected_proxy(flow, expected, fn, *args, **kwargs)
        except Exception as exc:
            if not is_http_403(exc) or retry >= retries:
                raise
            time.sleep(retry + 1)
    raise LoginFailure('http_403')


def checked_url(value):
    try:
        url = urlsplit(value)
        if (url.scheme != 'https' or url.hostname not in ('chatgpt.com', 'auth.openai.com')
                or url.username or url.password or url.port not in (None, 443)):
            raise ValueError()
    except ValueError:
        raise LoginFailure('challenge') from None
    return value


def check_step(page, url):
    value = (str(page) + ' ' + urlsplit(str(url)).path).lower()
    if any(part in value for part in ('create-account', 'create_account', 'signup', 'sign-up', 'register')):
        raise LoginFailure('registration')
    if any(part in value for part in ('email_otp', 'email-verification', 'passwordless')):
        raise LoginFailure('email_otp')
    if any(part in value for part in ('add-phone', 'phone_verification', 'phone-verification')):
        raise LoginFailure('phone')


class _NoEmailOtp:
    """Keep the mother flow strictly password/TOTP based.

    The shared account-pool driver can handle email OTP when a mail provider
    is supplied. Team mother login deliberately has no mail-provider input,
    so reaching that branch becomes a precise diagnostic instead of an
    AttributeError or an accidental registration fallback.
    """

    def wait_for_otp(self, *args, **kwargs):
        raise LoginFailure('email_otp')


def login_via_account_pool(payload, emit, factory=None):
    """Run the exact existing-account protocol used by the account pool."""
    from auth_flow import AuthFlow
    from config import Config

    flow_factory = factory or AuthFlow

    def new_flow():
        try:
            current = flow_factory(
                Config(proxy=payload['proxy'] or None),
                env_overrides={
                    'WEBUI_ALLOW_LOGIN': '1',
                    'OAUTH_CODEX_RT_ALLOW_RETRY': '1',
                },
            )
        except TypeError as exc:
            # Keep the worker test seam and older external factories compatible.
            if factory is None or 'env_overrides' not in str(exc):
                raise
            current = flow_factory(Config(proxy=payload['proxy'] or None))
        current._http_trace_enabled = False
        current._trace_dump_enabled = False
        current._trace_include_cookie = False
        current.result.email = payload['email']
        current.result.password = payload['password']
        current.result.totp_secret = payload['totp_secret']
        current._expected_login_email = payload['email']
        current._is_existing_account = True
        return current

    flow = new_flow()
    try:
        result = None
        for attempt in range(2):
            emit('protocol' if attempt == 0 else 'state_reset')
            try:
                result = through_selected_proxy(
                    flow,
                    payload['proxy'],
                    flow.run_protocol_login,
                    _NoEmailOtp(),
                    payload['email'],
                    payload['password'],
                    require_session=True,
                    existing_only=True,
                )
                break
            except Exception as exc:
                if attempt == 0 and is_invalid_state(exc):
                    # A 409/invalid_state belongs to the old OAuth cookie
                    # state. Rebuild the complete session while pinning the
                    # same selected proxy; never rotate behind the account.
                    try:
                        flow.session.close()
                    finally:
                        flow = new_flow()
                    continue
                raise
        if result is None:
            raise LoginFailure('invalid_state')
        values = {
            'access_token': str(getattr(result, 'access_token', '') or '').strip(),
            'session_token': str(getattr(result, 'session_token', '') or '').strip(),
            'refresh_token': str(getattr(result, 'refresh_token', '') or '').strip(),
        }
        if not all(20 <= len(value) <= 65536 for value in values.values()):
            raise LoginFailure('credentials_incomplete')
        emit('credentials')
        return {'type': 'result', 'email': payload['email'], 'credentials': values}
    finally:
        flow.session.close()


def login(payload, emit, factory=None):
    # Import only in this isolated process, never attach original flow loggers to WebUI.
    from auth_flow import AuthFlow, _hotp, _totp_now
    from config import Config

    # Keep Team mother login on the account-pool implementation. The legacy
    # hand-written driver below remains in the file for rollback/reference,
    # but production calls use one protocol implementation and one callback /
    # session / OAuth ordering.
    return login_via_account_pool(payload, emit, factory)

    # Legacy driver (unreachable; retained temporarily for a small rollback).
    flow = None

    def new_flow():
        """Create a fresh auth state while retaining the selected proxy."""
        current = (factory or AuthFlow)(Config(proxy=payload['proxy'] or None))
        current._http_trace_enabled = False
        current._trace_dump_enabled = False
        current._trace_include_cookie = False
        current.result.email = payload['email']
        current.result.password = payload['password']
        current.result.totp_secret = payload['totp_secret']
        current._expected_login_email = payload['email']
        current._is_existing_account = True
        emit('warmup')
        try:
            through_selected_proxy(current, payload['proxy'], current.prepare_existing_login,
                                   fixed_proxy=True)
        except RuntimeError as exc:
            if 'warmup' in str(exc).lower():
                detail = str(exc).lower()
                if 'http 403' in detail:
                    raise LoginFailure('warmup_http_403') from None
                if '未取得响应' in str(exc) or 'runtimeerror' in detail:
                    raise LoginFailure('warmup_network') from None
                if '未取得 chatgpt.com 登录 cookie' in str(exc):
                    raise LoginFailure('warmup_no_device_cookie') from None
                raise LoginFailure('warmup_failed') from None
            raise
        return current

    try:
        address = payload['email']
        flow = new_flow()
        def account_lookup():
            emit('csrf')
            csrf = through_selected_proxy(flow, payload['proxy'], flow.get_csrf_token)
            emit('auth_url')
            auth_url = checked_url(through_selected_proxy(
                flow, payload['proxy'], flow.get_auth_url, csrf, email=address))
            emit('oauth_init')
            device = through_selected_proxy(flow, payload['proxy'], flow.auth_oauth_init, auth_url)
            emit('sentinel')
            sentinel = through_selected_proxy(flow, payload['proxy'], flow.get_sentinel_token, device)
            emit('account_lookup')
            return login_step_with_403_retry(
                flow, payload['proxy'], flow.authorize_continue, email=address,
                sentinel_token=sentinel, screen_hint='login',
                referer='https://auth.openai.com/log-in', trace_step='mother_login')

        try:
            step = account_lookup()
        except Exception as exc:
            # A stale OAuth state is recoverable before password/TOTP have been
            # submitted. Rebuild the complete flow once. Reusing the old
            # cookies can preserve the stale state, so a fresh AuthFlow is
            # created while the selected proxy URL remains exactly unchanged.
            if not is_invalid_state(exc):
                raise
            emit('state_reset')
            try:
                flow.session.close()
            except Exception:
                pass
            flow = new_flow()
            step = account_lookup()
        def unpack(step):
            if not isinstance(step, dict):
                raise LoginFailure('challenge')
            page = flow._extract_page_type(step) or ''
            url = flow._extract_continue_url_from_step(step) or ''
            if url:
                url = checked_url(urljoin('https://auth.openai.com/', url))
            check_step(page, url)
            return page, url
        page, url = unpack(step)
        if page == 'login_password' or '/log-in/password' in url:
            emit('password')
            page, url = unpack(login_step_with_403_retry(
                flow, payload['proxy'], flow.login_password_verify, payload['password']))
        if flow._is_mfa_challenge_state(page, url):
            emit('totp')
            match = re.search(r'/mfa-challenge/([A-Za-z0-9_-]+)(?:/)?$', urlsplit(url).path)
            if not match:
                raise LoginFailure('challenge')
            challenge_id = match[1]
            # A small clock skew is common on servers running behind a VM or
            # a proxy gateway. Try the current TOTP window first, then the
            # adjacent windows only after the auth endpoint returns 403.
            last_totp_error = None
            totp_offsets = (0, -1, 1)
            clock_epoch = time.time()
            for index, offset in enumerate(totp_offsets):
                # Prefer the remote HTTP clock when available. A login retry
                # can cross a 30-second TOTP boundary or run on a skewed VM.
                counter = int(clock_epoch) // 30
                code = _hotp(payload['totp_secret'], counter + offset)
                try:
                    # A failed TOTP submission can invalidate the challenge.
                    # Do not submit another code to that same challenge, and
                    # do not repeat the same code three times.
                    page, url = unpack(login_step_with_403_retry(
                        flow, payload['proxy'], submit_mfa_compat,
                        flow, code, challenge_id, retries=0))
                    last_totp_error = None
                    break
                except Exception as exc:
                    last_totp_error = exc
                    if not is_http_403(exc) or index >= len(totp_offsets) - 1:
                        raise
                    remote_epoch = response_epoch(exc)
                    if remote_epoch is not None:
                        clock_epoch = remote_epoch
                    emit('state_reset')
                    try:
                        try:
                            flow.session.close()
                        except Exception:
                            pass
                        flow = new_flow()
                        step = account_lookup()
                        page, url = unpack(step)
                        if page == 'login_password' or '/log-in/password' in url:
                            emit('password')
                            page, url = unpack(login_step_with_403_retry(
                                flow, payload['proxy'], flow.login_password_verify,
                                payload['password']))
                        if not flow._is_mfa_challenge_state(page, url):
                            raise LoginFailure('challenge')
                        match = re.search(r'/mfa-challenge/([A-Za-z0-9_-]+)(?:/)?$',
                                          urlsplit(url).path)
                        if not match:
                            raise LoginFailure('challenge')
                        challenge_id = match[1]
                    except Exception:
                        # A secondary account-lookup rejection must not hide
                        # the original 2FA failure from the operator.
                        raise last_totp_error
            if last_totp_error is not None:
                raise last_totp_error
        if not url or flow._is_mfa_challenge_state(page, url) or page == 'login_password':
            raise LoginFailure('challenge')
        emit('redirect')
        # Consume callback once. Every redirect is checked; no guessed POST/consent/registration paths.
        # A transient 403 can be returned by the callback edge while the auth
        # session is being propagated. Retry that exact request a few times;
        # keep the same cookies, fingerprint and selected proxy. Explicit CF
        # challenges are not retried because repeating them cannot complete a
        # browser interaction that this worker does not perform.
        referer = 'https://auth.openai.com/'
        for _ in range(15):
            checked_url(url)
            check_step('', url)
            response = None
            for retry in range(4):
                response = through_selected_proxy(
                    flow, payload['proxy'], flow.session.get, url,
                    headers=redirect_headers(flow, url, referer),
                    timeout=30, allow_redirects=False)
                code = redirect_error_code(url, response)
                if code is None:
                    break
                if code == 'web_challenge':
                    raise LoginFailure(code)
                if response.status_code == 403 and retry < 3:
                    # Keep this deliberately short: this is a propagation
                    # retry, not a proxy rotation or a login retry.
                    time.sleep(1 + retry)
                    continue
                raise LoginFailure(code)
            if response is None:
                raise LoginFailure('redirect_http_403')
            if response.status_code in (301, 302, 303, 307, 308):
                location = response.headers.get('Location')
                if not location:
                    raise LoginFailure('challenge')
                referer = url
                url = checked_url(urljoin(url, location))
                continue
            break
        else:
            raise LoginFailure('challenge')
        emit('session')
        session_token, token, identity = recover_web_auth_session(
            flow, payload['proxy'], 'mother_post_callback')
        if not isinstance(token, str) or not 20 <= len(token) <= 32768:
            diagnostic = session_diagnostic(flow)
            if diagnostic.get('top_keys') == ['WARNING_BANNER']:
                failure = LoginFailure('session_warning_banner')
            else:
                failure = LoginFailure('no_token')
            failure.session_diagnostic = diagnostic
            raise failure
        if not isinstance(session_token, str) or not 20 <= len(session_token) <= 65536:
            raise LoginFailure('no_session')
        web_access_token = token

        # Reuse the account-pool Codex OAuth path to complete RT. This uses an
        # independent authorization code and does not consume the web session.
        emit('refresh')
        if not through_selected_proxy(
                flow, payload['proxy'], flow.oauth_codex_rt_exchange, mail_provider=None):
            raise LoginFailure('no_refresh')
        refresh_token = flow.result.refresh_token
        if not isinstance(refresh_token, str) or not 20 <= len(refresh_token) <= 65536:
            raise LoginFailure('no_refresh')

        # Codex exchange temporarily writes its access token into AuthResult;
        # read the web session again so Team stores the ChatGPT web AT.
        emit('credentials')
        final_session, final_token, identity = recover_web_auth_session(
            flow, payload['proxy'], 'mother_post_refresh')
        session_token = final_session or session_token
        token = final_token or web_access_token
        if not all(isinstance(value, str) and 20 <= len(value) <= 65536
                   for value in (session_token, token, refresh_token)):
            raise LoginFailure('credentials_incomplete')
        return {'type': 'result', 'email': address, 'credentials': {
            'access_token': token, 'session_token': session_token,
            'refresh_token': refresh_token,
        }}
    finally:
        if flow is not None:
            flow.session.close()


def preflight(payload, emit, factory=None):
    """Run the exact mother-login warmup without submitting credentials."""
    from auth_flow import AuthFlow
    from config import Config
    flow = (factory or AuthFlow)(Config(proxy=payload['proxy'] or None))
    flow._http_trace_enabled = False
    flow._trace_dump_enabled = False
    flow._trace_include_cookie = False
    try:
        emit('warmup')
        try:
            flow.prepare_existing_login(fixed_proxy=True, fail_fast_proxy=True)
        except RuntimeError as exc:
            if 'proxy_preflight_failed' in str(exc).lower():
                raise LoginFailure('proxy_preflight_failed') from None
            raise
        return {'type': 'preflight_result', 'ready': True}
    finally:
        flow.session.close()


def error_code(exc):
    if isinstance(exc, LoginFailure):
        return str(exc)
    diagnostic = getattr(exc, 'session_diagnostic', None)
    if isinstance(diagnostic, dict) and diagnostic.get('top_keys') == ['WARNING_BANNER']:
        return 'session_warning_banner'
    status = getattr(getattr(exc, 'response', None), 'status_code', None)
    # Only extract HTTP numbers; never forward raw exceptions, response bodies or URLs.
    text = str(exc)
    lowered = text.lower()
    auth_error = str(getattr(exc, "auth_error_code", "") or "").strip().lower()
    if auth_error in {"invalid_code", "challenge_expired", "rate_limited", "web_challenge"}:
        return "totp_" + auth_error
    if re.search(r'\b403\b', text):
        if 'totp' in lowered or 'mfa' in lowered:
            return 'totp_http_403'
        if '密码登录' in text or 'password' in lowered:
            return 'password_http_403'
        if 'authorize/continue' in lowered:
            return 'account_lookup_http_403'
    if 'user was rejected by the socks5 server' in lowered or 'proxy authentication' in lowered:
        return 'proxy_auth'
    if 'csrf token' in lowered:
        return 'csrf_failed'
    if 'auth url' in lowered:
        return 'auth_url_failed'
    for dependency_code in ('sentinel_asset_missing', 'sentinel_node_missing'):
        if dependency_code in lowered:
            return dependency_code
    if 'sentinel' in lowered or 'proof of work' in lowered:
        return 'sentinel_failed'
    if 'authorize/continue' in lowered:
        if re.search(r'\b409\b', text):
            return 'invalid_state'
        if re.search(r'\b(?:500|502|503|504)\b', text):
            return 'upstream'
        return 'account_lookup_failed'
    if '已有账号登录未命中账号分支' in text:
        return 'account_lookup_failed'
    if '登录身份校验不一致' in text:
        return 'identity'
    if '未拿到有效 session/access token' in text:
        return 'no_token'
    if '协议登录完成，但 cpa' in lowered or 'access/refresh 未齐全' in text:
        return 'credentials_incomplete'
    if 'email_otp' in lowered or '邮箱 otp' in lowered:
        return 'email_otp'
    if 'proxy_preflight_failed' in lowered:
        return 'proxy_preflight_failed'
    if 'warmup' in lowered or 'oai-did' in lowered:
        if 'http 403' in lowered:
            return 'warmup_http_403'
        if '未取得响应' in text or 'runtimeerror' in lowered:
            return 'warmup_network'
        if '未取得 chatgpt.com 登录 cookie' in text:
            return 'warmup_no_device_cookie'
        return 'warmup_failed'
    # Prefer structured response status to digits embedded in exception text.
    if status is None:
        match = re.search(r'\b(400|401|403|407|409|429|500|502|503|504)\b', text)
        status = int(match[1]) if match else None
    if status in (401, 400):
        return 'unauthorized'
    if status in (403, 407, 429):
        return {403: 'http_403', 407: 'proxy_auth', 429: 'http_429'}[status]
    if status == 409:
        return 'invalid_state'
    if status in (500, 502, 503, 504):
        return 'upstream'
    if any(x in type(exc).__name__.lower() for x in ('timeout', 'connection', 'proxy', 'ssl')):
        return 'network'
    return 'unknown'


def main():
    logging.disable(logging.CRITICAL)
    if os.name == 'posix':
        import resource
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
    output = sys.stdout
    def send(data):
        output.write(json.dumps(data, ensure_ascii=False) + '\n')
        output.flush()
    payload = {}
    try:
        line = sys.stdin.readline(16385)
        if len(line) > 16384:
            raise LoginFailure('unknown')
        payload = json.loads(line)
        with open(os.devnull, 'w') as sink, contextlib.redirect_stdout(sink), contextlib.redirect_stderr(sink):
            runner = preflight if payload.get('mode') == 'preflight' else login
            result = runner(payload, lambda stage: send({'type': 'stage', 'stage': stage}))
        send(result)
    except Exception as exc:
        diagnostic = getattr(exc, 'session_diagnostic', None)
        if diagnostic:
            send({'type': 'diagnostic', 'stage': 'session', 'details': diagnostic})
        send({'type': 'error', 'code': error_code(exc)})
    finally:
        if isinstance(payload, dict):
            payload.clear()


if __name__ == '__main__':
    main()
