import base64
import json
import subprocess
import sys
from types import SimpleNamespace
import pytest
from credential_session_kit.client import _stream_worker
from credential_session_kit.errors import CredentialSessionError
from credential_session_kit.engine import worker
from auth_flow import AuthFlow


def jwt(email):
    body = base64.urlsafe_b64encode(json.dumps({"email": email}).encode()).decode().rstrip("=")
    return "fixture." + body + ".fixture"


@pytest.mark.parametrize("expected,session_email,token_email,ok", [
    ("a@example.com", "b@example.com", "b@example.com", False),
    ("a@example.com", "a@example.com", "b@example.com", False),
    ("a@example.com", "", "", False),
    ("a@example.com", "a@example.com", "", True),
    ("a@example.com", "", "a@example.com", True),
    ("a@example.com", "a@example.com", "a@example.com", True),
])
def test_real_session_identity_check(expected, session_email, token_email, ok):
    flow = object.__new__(AuthFlow)
    flow._expected_login_email = expected
    flow._common_headers = lambda *a: {}
    flow._trace_http = lambda *a: None
    flow._safe_cookie_names = lambda: []
    flow._extract_session_cookie = lambda: "s" * 24
    flow._build_chatgpt_cookie_header = lambda: ""
    data = {"sessionToken": "s" * 24, "accessToken": jwt(token_email), "user": {"email": session_email}}
    response = SimpleNamespace(status_code=200, headers={}, content=b"", json=lambda: data, raise_for_status=lambda: None)
    flow.session = SimpleNamespace(get=lambda *a, **kw: response)
    flow.result = SimpleNamespace()
    flow.get_auth_session(include_identity=True)
    assert flow._auth_identity_consistent is ok


def test_strict_login_propagates_409_without_signup():
    flow = object.__new__(AuthFlow)
    flow.prepare_existing_login = lambda **kw: None
    flow.result = SimpleNamespace()
    flow.get_csrf_token = lambda: "csrf"
    flow.get_auth_url = lambda *a, **kw: "url"
    flow.auth_oauth_init = lambda *a: "device"
    flow.get_sentinel_token = lambda *a: "sentinel"
    flow._get_env = lambda *a: "60"
    def reject(**kw):
        raise RuntimeError("HTTP 409 invalid_state")
    flow.authorize_continue = reject
    flow.signup = lambda *a: pytest.fail("strict login must not fall back to signup")
    with pytest.raises(RuntimeError, match="invalid_state"):
        flow.run_protocol_login(None, "a@example.com", "password", existing_only=True)


@pytest.mark.parametrize("always_fail", [False, True])
def test_retry_closes_old_session_and_is_bounded(always_fail):
    flows = []
    events = []
    class Flow:
        def __init__(self, config, **kw):
            self.config = config
            self.result = SimpleNamespace(access_token="a"*24, session_token="s"*24, refresh_token="r"*24)
            self.closed = False
            self.session = SimpleNamespace(proxies={"http": config.proxy, "https": config.proxy}, close=self.close)
            if flows:
                assert flows[-1].closed
            flows.append(self)
        def close(self):
            self.closed = True
        def run_protocol_login(self, *a, **kw):
            if always_fail or len(flows) == 1:
                raise RuntimeError("invalid_state")
            return self.result
    payload = {"email": "a@example.com", "password": "secret", "totp_secret": "fixture", "proxy": "http://proxy"}
    if always_fail:
        with pytest.raises(RuntimeError, match="invalid_state"):
            worker.login_via_account_pool(payload, events.append, factory=Flow)
    else:
        assert worker.login_via_account_pool(payload, events.append, factory=Flow)["credentials"]["access_token"] == "a"*24
    assert len(flows) == 2
    assert all(f.closed and f.config.proxy == "http://proxy" for f in flows)
    assert events.count("state_reset") == 1


def child(code):
    return subprocess.Popen([sys.executable, "-u", "-c", code], stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                            text=True, encoding="utf-8")


def test_progress_arrives_before_child_finishes():
    process = child('import sys,json,time; sys.stdin.readline(); print(json.dumps({"type":"stage","stage":"password"}),flush=True); time.sleep(.5); print(json.dumps({"type":"result","email":"a@example.com"}),flush=True)')
    stages = []
    def progress(stage):
        assert process.poll() is None
        stages.append(stage)
    result, error = _stream_worker(process, {}, 5, progress)
    assert stages == ["password"] and result["email"] == "a@example.com"
    assert error is None and process.poll() == 0


def test_timeout_keeps_emitted_progress_and_reaps_child():
    process = child('import sys,json,time; sys.stdin.readline(); print(json.dumps({"type":"stage","stage":"password"}),flush=True); time.sleep(30)')
    stages = []
    with pytest.raises(CredentialSessionError) as error:
        _stream_worker(process, {}, .6, stages.append)
    assert error.value.code == "auth_timeout"
    assert stages == ["password"] and process.poll() is not None
