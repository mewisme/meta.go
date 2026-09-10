from __future__ import annotations

from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import TypeAlias

from meta.v1 import session_pb2


@dataclass(frozen=True, slots=True)
class Credentials:
    identifier: str
    password: str
    totp: str | None = None
    otp: str | None = None

    def __post_init__(self) -> None:
        if not self.identifier.strip():
            raise ValueError("identifier is required")
        if not self.password:
            raise ValueError("password is required")
        has_totp = bool(self.totp and self.totp.strip())
        has_otp = bool(self.otp and self.otp.strip())
        if has_totp == has_otp:
            raise ValueError("credentials require exactly one of totp or otp")


@dataclass(frozen=True, slots=True)
class AppStateCookie:
    key: str
    value: str
    domain: str = ""
    path: str = ""
    host_only: bool = False
    secure: bool = False
    http_only: bool = False

    def __post_init__(self) -> None:
        if not self.key.strip() or not self.value:
            raise ValueError("app state cookies require key and value")


@dataclass(frozen=True, slots=True)
class CookieAuth:
    cookies: Mapping[str, str]

    def __post_init__(self) -> None:
        if not self.cookies.get("c_user") or not self.cookies.get("xs"):
            raise ValueError("cookies require c_user and xs")


@dataclass(frozen=True, slots=True)
class AppStateAuth:
    app_state: Sequence[AppStateCookie]

    def __post_init__(self) -> None:
        if not self.app_state:
            raise ValueError("app_state must contain at least one cookie")


@dataclass(frozen=True, slots=True)
class CredentialAuth:
    credentials: Credentials


SessionAuth: TypeAlias = CookieAuth | AppStateAuth | CredentialAuth


@dataclass(frozen=True, slots=True)
class FacebookSession:
    account_id: str
    name: str
    username: str
    dtsg: str
    jazoest: str
    lsd: str
    session_id: str
    client_revision: int
    refreshed_at: datetime | None


@dataclass(frozen=True, slots=True)
class AuthSnapshot:
    cookies: dict[str, str]
    app_state: tuple[AppStateCookie, ...]
    session: FacebookSession | None


def _auth_capability(auth: SessionAuth) -> str | None:
    if isinstance(auth, AppStateAuth):
        return "session.auth.app_state"
    if isinstance(auth, CredentialAuth):
        return "session.auth.credentials"
    return None


def _session_auth_to_proto(auth: SessionAuth) -> session_pb2.SessionAuth:
    if isinstance(auth, CookieAuth):
        return session_pb2.SessionAuth(cookies=session_pb2.CookieMap(values=dict(auth.cookies)))
    if isinstance(auth, AppStateAuth):
        return session_pb2.SessionAuth(
            app_state=session_pb2.AppState(
                cookies=[
                    session_pb2.AppStateCookie(
                        key=cookie.key,
                        value=cookie.value,
                        domain=cookie.domain,
                        path=cookie.path,
                        host_only=cookie.host_only,
                        secure=cookie.secure,
                        http_only=cookie.http_only,
                    )
                    for cookie in auth.app_state
                ]
            )
        )
    credentials = auth.credentials
    factor = {"totp": credentials.totp} if credentials.totp else {"otp": credentials.otp}
    return session_pb2.SessionAuth(
        credentials=session_pb2.Credentials(identifier=credentials.identifier, password=credentials.password, **factor)
    )


def _auth_snapshot_from_proto(snapshot: session_pb2.AuthSnapshot) -> AuthSnapshot:
    cookies = dict(snapshot.cookies.values) if snapshot.HasField("cookies") else {}
    app_state = (
        tuple(
            AppStateCookie(
                key=cookie.key,
                value=cookie.value,
                domain=cookie.domain,
                path=cookie.path,
                host_only=cookie.host_only,
                secure=cookie.secure,
                http_only=cookie.http_only,
            )
            for cookie in snapshot.app_state.cookies
        )
        if snapshot.HasField("app_state")
        else ()
    )
    if not snapshot.HasField("session"):
        return AuthSnapshot(cookies=cookies, app_state=app_state, session=None)
    session = snapshot.session
    refreshed_at = session.refreshed_at.ToDatetime(tzinfo=timezone.utc) if session.HasField("refreshed_at") else None
    return AuthSnapshot(
        cookies=cookies,
        app_state=app_state,
        session=FacebookSession(
            account_id=session.account_id,
            name=session.name,
            username=session.username,
            dtsg=session.dtsg,
            jazoest=session.jazoest,
            lsd=session.lsd,
            session_id=session.session_id,
            client_revision=session.client_revision,
            refreshed_at=refreshed_at,
        ),
    )
