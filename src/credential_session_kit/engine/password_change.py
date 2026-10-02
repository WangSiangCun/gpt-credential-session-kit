"""Password change through the authenticated reset flow, never local-only edits."""
from urllib.parse import urlsplit

from credential_session_kit.errors import CredentialSessionError


class _Finished(Exception):
    pass


def validate_new_password(value):
    if not isinstance(value, str) or not 12 <= len(value) <= 128 or any(c in value for c in "\r\n"):
        raise CredentialSessionError("security_password_policy")


def submit_password(flow, step, new_password, emit):
    """Only submit after the server explicitly issues the authenticated new-password page."""
    page = flow._extract_page_type(step)
    if page != "reset_password_new_password":
        return False
    path = urlsplit(flow._normalize_continue_url(flow._extract_continue_url_from_step(step)))
    if path.scheme != "https" or path.netloc != "auth.openai.com" or path.path != "/reset-password/new-password":
        raise CredentialSessionError("security_password_state")
    validate_new_password(new_password)
    emit("security_password_ready")
    # Existing flow supplies the required browser security headers on its own
    # authenticated session. Do not reuse an authorize_continue token here.
    flow.get_sentinel_token(flow.result.device_id, flow_name="password_reset")
    headers = flow._common_headers("https://auth.openai.com/reset-password/new-password")
    headers["Content-Type"] = "application/json"
    if flow._last_sentinel_token:
        headers["openai-sentinel-token"] = flow._last_sentinel_token
    if getattr(flow, "_last_sentinel_so_token", ""):
        headers["openai-sentinel-so-token"] = flow._last_sentinel_so_token
    # Disable urllib3 automatic retries for a requests-based fallback session.
    if hasattr(flow.session, "mount"):
        from requests.adapters import HTTPAdapter
        flow.session.mount("https://auth.openai.com/", HTTPAdapter(max_retries=0))
    emit("security_password_submitting")
    try:
        response = flow.session.request("POST", "https://auth.openai.com/api/accounts/password/reset",
            headers=headers, json={"password": new_password}, timeout=30, allow_redirects=False)
        if response.status_code in (400, 401, 403, 409, 422, 429):
            raise CredentialSessionError("security_password_rejected")
        if response.status_code != 200:
            raise CredentialSessionError("security_password_uncertain")
        data = response.json()
        if not isinstance(data, dict) or flow._extract_page_type(data) != "reset_password_success":
            raise CredentialSessionError("security_password_uncertain")
    except CredentialSessionError:
        raise
    except Exception:
        raise CredentialSessionError("security_password_uncertain") from None
    emit("security_password_applied")
    return True


def run_password_change(payload, emit, *, run_login=None, flow_class=None):
    from auth_flow import AuthFlow
    from worker import login_via_account_pool
    validate_new_password(payload.get("new_password"))
    if not payload.get("mail_code_url"):
        raise CredentialSessionError("security_mail_required")
    base = flow_class or AuthFlow
    runner = run_login or login_via_account_pool

    class PasswordFlow(base):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, **kwargs)
            self._password_change_intent = True

        def _inspect_security_step(self, result):
            if submit_password(self, result, payload["new_password"], emit):
                raise _Finished()
            return result

        def authorize_continue(self, *args, **kwargs):
            return self._inspect_security_step(super().authorize_continue(*args, **kwargs))

        def login_password_verify(self, *args, **kwargs):
            return self._inspect_security_step(super().login_password_verify(*args, **kwargs))

        def submit_mfa_totp(self, *args, **kwargs):
            return self._inspect_security_step(super().submit_mfa_totp(*args, **kwargs))

        def verify_otp(self, *args, **kwargs):
            return self._inspect_security_step(super().verify_otp(*args, **kwargs))

    try:
        runner(payload, emit, factory=PasswordFlow)
    except _Finished:
        return {"type": "result", "password_changed": True, "email": payload["email"]}
    raise CredentialSessionError("security_password_state")
