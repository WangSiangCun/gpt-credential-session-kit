from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path

from .errors import CredentialSessionError
from .models import CredentialResult


class CredentialSessionClient:
    """Small process-isolated facade over the account-pool protocol flow."""

    def __init__(self, *_ignored, python: str | None = None):
        # The protocol worker ships inside this package.
        self.python = python or sys.executable
        self.worker_module = "credential_session_kit.engine.worker"


    def preflight(self, proxy_url: str, timeout: int = 180) -> dict:
        """Run the account-pool proxy preflight without starting a login."""
        proxy = str(proxy_url or "").strip()
        if not proxy:
            raise CredentialSessionError("credentials")
        values = {"mode": "preflight", "proxy": proxy}
        env = {key: value for key, value in os.environ.items()
               if key.upper() not in {"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
                                      "http_proxy", "https_proxy", "all_proxy"}}
        env.update(PYTHONIOENCODING="utf-8", PYTHONUNBUFFERED="1",
                   AUTH_HTTP_TRACE="0", AUTH_TRACE_DUMP="0")
        env["PYTHONPATH"] = os.pathsep.join(
            part for part in (str(Path(__file__).resolve().parent / "engine"), env.get("PYTHONPATH", "")) if part)
        process = None
        try:
            process = subprocess.Popen(
                [self.python, "-m", self.worker_module],
                cwd=str(Path(__file__).resolve().parent / "engine"), env=env, stdin=subprocess.PIPE,
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
                text=True, encoding="utf-8",
                creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
            stdout, _ = process.communicate(
                json.dumps(values, ensure_ascii=False) + "\n", timeout=timeout)
        except subprocess.TimeoutExpired:
            process.kill()
            process.communicate()
            raise CredentialSessionError("auth_timeout") from None
        except OSError:
            raise CredentialSessionError("auth_runner") from None
        finally:
            values.clear()
        result = None
        error = None
        for line in (stdout or "").splitlines():
            try:
                event = json.loads(line)
            except (TypeError, ValueError):
                continue
            if event.get("type") == "preflight_result":
                result = event
            elif event.get("type") == "error":
                error = event
        if error or process.returncode != 0 or not result or not result.get("ready"):
            raise CredentialSessionError(str((error or {}).get("code") or "warmup_failed"))
        return {"ok": True}

    def login(
        self,
        *,
        email: str,
        password: str,
        totp_secret: str,
        proxy_url: str,
        timeout: int = 300,
        on_stage=None,
    ) -> CredentialResult:
        values = {
            "email": str(email or "").strip(),
            "password": str(password or ""),
            "totp_secret": str(totp_secret or ""),
            "proxy": str(proxy_url or "").strip(),
        }
        if not all(values.values()):
            raise CredentialSessionError("credentials")
        env = {
            key: value for key, value in os.environ.items()
            if key.upper() not in {
                "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
                "http_proxy", "https_proxy", "all_proxy",
            }
        }
        env.update(PYTHONIOENCODING="utf-8", PYTHONUNBUFFERED="1",
                   AUTH_HTTP_TRACE="0", AUTH_TRACE_DUMP="0")
        env["PYTHONPATH"] = os.pathsep.join(
            part for part in (str(Path(__file__).resolve().parent / "engine"), env.get("PYTHONPATH", "")) if part
        )
        process = None
        try:
            process = subprocess.Popen(
                [self.python, "-m", self.worker_module],
                cwd=str(Path(__file__).resolve().parent / "engine"), env=env,
                stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL, text=True, encoding="utf-8",
                creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
            )
            stdout, _ = process.communicate(
                json.dumps(values, ensure_ascii=False) + "\n", timeout=timeout
            )
        except subprocess.TimeoutExpired:
            process.kill()
            process.communicate()
            raise CredentialSessionError("auth_timeout") from None
        except OSError:
            raise CredentialSessionError("auth_runner") from None
        finally:
            values.clear()

        result = None
        error = None
        for line in (stdout or "").splitlines():
            try:
                event = json.loads(line)
            except (TypeError, ValueError):
                continue
            if event.get("type") == "stage":
                if on_stage:
                    on_stage(str(event.get("stage") or "unknown"))
            elif event.get("type") == "result":
                result = event
            elif event.get("type") == "error":
                error = event
        if error or process.returncode != 0 or not result:
            raise CredentialSessionError(str((error or {}).get("code") or "auth_failed"))
        credentials = result.get("credentials")
        if not isinstance(credentials, dict):
            raise CredentialSessionError("credentials_incomplete")
        fields = {key: str(credentials.get(key) or "").strip()
                  for key in ("access_token", "session_token", "refresh_token")}
        if not all(20 <= len(value) <= 65536 for value in fields.values()):
            raise CredentialSessionError("credentials_incomplete")
        return CredentialResult(
            email=str(result.get("email") or "").strip(),
            **fields,
        )
