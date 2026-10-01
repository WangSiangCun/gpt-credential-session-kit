from pathlib import Path
import pytest
from credential_session_kit.engine import sentinel_quickjs as runtime
from credential_session_kit.engine.worker import error_code


def test_runtime_asset_is_bundled():
    asset = runtime._quickjs_script_path()
    assert asset.is_file()
    assert asset.stat().st_size > 1000


def test_missing_asset_reports_dependency_before_network(monkeypatch, tmp_path):
    monkeypatch.setattr(runtime, "_quickjs_script_path", lambda: tmp_path / "missing.js")
    with pytest.raises(RuntimeError, match="sentinel_asset_missing"):
        runtime.get_sentinel_token_via_quickjs(None, device_id="fixture", flow="authorize_continue")


def test_missing_node_reports_dependency_before_network(monkeypatch):
    monkeypatch.setattr(runtime.shutil, "which", lambda _: None)
    with pytest.raises(RuntimeError, match="sentinel_node_missing"):
        runtime.get_sentinel_token_via_quickjs(None, device_id="fixture", flow="authorize_continue")


@pytest.mark.parametrize(
    ("message", "expected"),
    [
        ("authorize/continue 失败(screen_hint=login): HTTP 403", "account_lookup_http_403"),
        ("密码登录失败: 403", "password_http_403"),
        ("TOTP 验证失败: 403", "totp_http_403"),
    ],
)
def test_login_403_is_reported_at_the_rejected_stage(message, expected):
    assert error_code(RuntimeError(message)) == expected
