from pathlib import Path
import pytest
from credential_session_kit.engine import sentinel_quickjs as runtime


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
