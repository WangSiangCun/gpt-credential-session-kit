"""Opt-in mail-code reader. No login cookies or tokens reach the mail provider."""
from __future__ import annotations

import re
import time
from datetime import datetime, timezone
from urllib.parse import parse_qs, urlsplit

from .errors import CredentialSessionError


def validate_mail_code_url(value: str) -> str:
    raw = str(value or "").strip()
    try:
        p = urlsplit(raw)
        query = parse_qs(p.query, strict_parsing=True)
        valid = (len(raw) <= 2048 and not any(ch.isspace() for ch in raw)
                 and p.scheme == "https" and p.netloc == "gapi.mailsapi.com"
                 and p.path == "/api/code/fetch" and not p.fragment
                 and set(query) == {"token", "uid"}
                 and all(len(v) == 1 and re.fullmatch(r"[A-Za-z0-9_-]{1,200}", v[0])
                         for v in query.values()))
    except (ValueError, TypeError):
        valid = False
    if not valid:
        raise CredentialSessionError("mail_code_url")
    return raw


def extract_message(payload):
    """Read explicit code fields, never six arbitrary digits in an error/URL.

    A provider's integer top-level `code` is an API status, not an OTP.
    Unknown structures fail closed rather than guessing a code.
    """
    if not isinstance(payload, dict):
        raise CredentialSessionError("mail_code_response")
    data = payload.get("data")
    if data is None:
        if payload.get("code") not in (0, 200):
            raise CredentialSessionError("mail_code_provider")
        return None
    if payload.get("code") not in (0, 200):
        raise CredentialSessionError("mail_code_provider")
    if isinstance(data, str):
        code, issued = data, None
    elif isinstance(data, dict):
        code, issued = data.get("code"), data.get("received_at", data.get("timestamp"))
    else:
        raise CredentialSessionError("mail_code_response")
    if not isinstance(code, str) or not re.fullmatch(r"[0-9]{6}", code):
        raise CredentialSessionError("mail_code_response")
    if issued is not None:
        try:
            if isinstance(issued, bool):
                raise ValueError()
            if isinstance(issued, (int, float)):
                issued = float(issued)
                if issued > 1e12:
                    issued /= 1000
            else:
                parsed = datetime.fromisoformat(str(issued).replace("Z", "+00:00"))
                if parsed.tzinfo is None:
                    raise ValueError()
                issued = parsed.astimezone(timezone.utc).timestamp()
            if not (0 < issued < float("inf")):
                raise ValueError()
        except (ValueError, TypeError, OverflowError):
            raise CredentialSessionError("mail_code_response") from None
    return code, issued


class MailCodeProvider:
    """Snapshot before triggering authentication; accept only a fresh code.

    `session` must be an isolated, proxy-locked session dedicated to mail.
    No redirects, raw response text, URLs or OTPs are exposed in errors.
    """
    def __init__(self, email, url, session, *, emit=None, clock=time.monotonic,
                 wall_clock=time.time, sleep=time.sleep):
        self.email = email.strip().lower()
        self.url = validate_mail_code_url(url)
        self.session = session
        self.emit = emit or (lambda stage: None)
        self.clock, self.wall_clock, self.sleep = clock, wall_clock, sleep
        self.seen = set()
        self.prepared = False

    def _read(self):
        try:
            r = self.session.get(self.url, headers={"Accept": "application/json"},
                                 allow_redirects=False, timeout=10)
            if r.status_code != 200:
                raise CredentialSessionError("mail_code_http")
            if len(r.content) > 65536:
                raise CredentialSessionError("mail_code_response")
            return extract_message(r.json())
        except CredentialSessionError:
            raise
        except Exception:
            raise CredentialSessionError("mail_code_network") from None

    def prepare(self):
        message = self._read()
        if message:
            self.seen.add(message[0])
        self.prepared = True

    def wait_for_otp(self, email, *, timeout=60, issued_after=None, **kwargs):
        if email.strip().lower() != self.email or not self.prepared:
            raise CredentialSessionError("mail_code_identity")
        self.emit("email_otp")
        deadline = self.clock() + min(max(float(timeout), 1), 120)
        after = float(issued_after) if issued_after is not None else self.wall_clock()
        while self.clock() < deadline:
            message = self._read()
            if message:
                code, received = message
                fresh = received is None or after <= received <= self.wall_clock() + 60
                if code not in self.seen and fresh:
                    self.seen.add(code)
                    return code
            self.sleep(min(3, max(0, deadline - self.clock())))
        raise CredentialSessionError("mail_code_timeout")
