from dataclasses import dataclass


@dataclass(frozen=True)
class CredentialResult:
    email: str
    access_token: str
    session_token: str
    refresh_token: str

    def to_dict(self) -> dict[str, str]:
        return {
            "email": self.email,
            "access_token": self.access_token,
            "session_token": self.session_token,
            "refresh_token": self.refresh_token,
        }
