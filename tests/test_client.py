import json
import sys
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1] / "src"))

from credential_session_kit import CredentialSessionClient, CredentialSessionError


def test_missing_input_is_rejected(tmp_path):
    worker = tmp_path / "webui" / "team"
    worker.mkdir(parents=True)
    (worker / "mother_login_worker.py").write_text("", encoding="utf-8")
    client = CredentialSessionClient(tmp_path)
    try:
        client.login(email="", password="p", totp_secret="s", proxy_url="http://x")
    except CredentialSessionError as exc:
        assert exc.code == "credentials"
    else:
        raise AssertionError("missing input accepted")


def test_result_is_redacted_error_and_tokens_are_returned(tmp_path):
    worker = tmp_path / "webui" / "team"
    worker.mkdir(parents=True)
    (worker / "mother_login_worker.py").write_text("", encoding="utf-8")
    client = CredentialSessionClient(tmp_path, python=sys.executable)
    payload = {
        "type": "stage", "stage": "password",
    }
    result = {
        "type": "result", "email": "owner@example.test",
        "credentials": {
            "access_token": "a" * 24,
            "session_token": "s" * 24,
            "refresh_token": "r" * 24,
        },
    }
    class Proc:
        returncode = 0
        def communicate(self, *args, **kwargs):
            return json.dumps(payload) + "\n" + json.dumps(result) + "\n", ""
    with patch("credential_session_kit.client.subprocess.Popen", return_value=Proc()):
        actual = client.login(email="owner@example.test", password="p", totp_secret="s", proxy_url="http://x")
    assert actual.access_token == "a" * 24
    assert actual.session_token == "s" * 24
    assert actual.refresh_token == "r" * 24
