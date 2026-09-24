class CredentialSessionError(RuntimeError):
    """Stable, redacted error returned by the credential client."""

    def __init__(self, code: str):
        self.code = str(code or "unknown")
        super().__init__(self.code)
