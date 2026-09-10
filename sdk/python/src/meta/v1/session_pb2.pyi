import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from meta.v1 import common_pb2 as _common_pb2
from meta.v1 import events_pb2 as _events_pb2
from meta.v1 import health_pb2 as _health_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateSessionRequest(_message.Message):
    __slots__ = ("cookies", "e2ee", "event_buffer", "timeout", "auth")
    class CookiesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    COOKIES_FIELD_NUMBER: _ClassVar[int]
    E2EE_FIELD_NUMBER: _ClassVar[int]
    EVENT_BUFFER_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    AUTH_FIELD_NUMBER: _ClassVar[int]
    cookies: _containers.ScalarMap[str, str]
    e2ee: bool
    event_buffer: int
    timeout: _duration_pb2.Duration
    auth: SessionAuth
    def __init__(self, cookies: _Optional[_Mapping[str, str]] = ..., e2ee: _Optional[bool] = ..., event_buffer: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., auth: _Optional[_Union[SessionAuth, _Mapping]] = ...) -> None: ...

class CreateSessionResponse(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ConnectRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ConnectResponse(_message.Message):
    __slots__ = ("account",)
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    account: _common_pb2.Account
    def __init__(self, account: _Optional[_Union[_common_pb2.Account, _Mapping]] = ...) -> None: ...

class Credentials(_message.Message):
    __slots__ = ("identifier", "password", "totp", "otp")
    IDENTIFIER_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    TOTP_FIELD_NUMBER: _ClassVar[int]
    OTP_FIELD_NUMBER: _ClassVar[int]
    identifier: str
    password: str
    totp: str
    otp: str
    def __init__(self, identifier: _Optional[str] = ..., password: _Optional[str] = ..., totp: _Optional[str] = ..., otp: _Optional[str] = ...) -> None: ...

class AppStateCookie(_message.Message):
    __slots__ = ("key", "value", "domain", "path", "host_only", "secure", "http_only")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    HOST_ONLY_FIELD_NUMBER: _ClassVar[int]
    SECURE_FIELD_NUMBER: _ClassVar[int]
    HTTP_ONLY_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    domain: str
    path: str
    host_only: bool
    secure: bool
    http_only: bool
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ..., domain: _Optional[str] = ..., path: _Optional[str] = ..., host_only: _Optional[bool] = ..., secure: _Optional[bool] = ..., http_only: _Optional[bool] = ...) -> None: ...

class AppState(_message.Message):
    __slots__ = ("cookies",)
    COOKIES_FIELD_NUMBER: _ClassVar[int]
    cookies: _containers.RepeatedCompositeFieldContainer[AppStateCookie]
    def __init__(self, cookies: _Optional[_Iterable[_Union[AppStateCookie, _Mapping]]] = ...) -> None: ...

class CookieMap(_message.Message):
    __slots__ = ("values",)
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.ScalarMap[str, str]
    def __init__(self, values: _Optional[_Mapping[str, str]] = ...) -> None: ...

class SessionAuth(_message.Message):
    __slots__ = ("cookies", "app_state", "credentials")
    COOKIES_FIELD_NUMBER: _ClassVar[int]
    APP_STATE_FIELD_NUMBER: _ClassVar[int]
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    cookies: CookieMap
    app_state: AppState
    credentials: Credentials
    def __init__(self, cookies: _Optional[_Union[CookieMap, _Mapping]] = ..., app_state: _Optional[_Union[AppState, _Mapping]] = ..., credentials: _Optional[_Union[Credentials, _Mapping]] = ...) -> None: ...

class FacebookSession(_message.Message):
    __slots__ = ("account_id", "name", "username", "dtsg", "jazoest", "lsd", "session_id", "client_revision", "refreshed_at")
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    DTSG_FIELD_NUMBER: _ClassVar[int]
    JAZOEST_FIELD_NUMBER: _ClassVar[int]
    LSD_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CLIENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    REFRESHED_AT_FIELD_NUMBER: _ClassVar[int]
    account_id: str
    name: str
    username: str
    dtsg: str
    jazoest: str
    lsd: str
    session_id: str
    client_revision: int
    refreshed_at: _timestamp_pb2.Timestamp
    def __init__(self, account_id: _Optional[str] = ..., name: _Optional[str] = ..., username: _Optional[str] = ..., dtsg: _Optional[str] = ..., jazoest: _Optional[str] = ..., lsd: _Optional[str] = ..., session_id: _Optional[str] = ..., client_revision: _Optional[int] = ..., refreshed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AuthSnapshot(_message.Message):
    __slots__ = ("cookies", "app_state", "session")
    COOKIES_FIELD_NUMBER: _ClassVar[int]
    APP_STATE_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    cookies: CookieMap
    app_state: AppState
    session: FacebookSession
    def __init__(self, cookies: _Optional[_Union[CookieMap, _Mapping]] = ..., app_state: _Optional[_Union[AppState, _Mapping]] = ..., session: _Optional[_Union[FacebookSession, _Mapping]] = ...) -> None: ...

class RefreshAuthRequest(_message.Message):
    __slots__ = ("session_id", "auth")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    AUTH_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    auth: SessionAuth
    def __init__(self, session_id: _Optional[str] = ..., auth: _Optional[_Union[SessionAuth, _Mapping]] = ...) -> None: ...

class RefreshAuthResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: AuthSnapshot
    def __init__(self, snapshot: _Optional[_Union[AuthSnapshot, _Mapping]] = ...) -> None: ...

class GetAuthSnapshotRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class GetAuthSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: AuthSnapshot
    def __init__(self, snapshot: _Optional[_Union[AuthSnapshot, _Mapping]] = ...) -> None: ...

class CloseSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class CloseSessionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetHealthRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class GetHealthResponse(_message.Message):
    __slots__ = ("health",)
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    health: _health_pb2.HealthSnapshot
    def __init__(self, health: _Optional[_Union[_health_pb2.HealthSnapshot, _Mapping]] = ...) -> None: ...

class SubscribeEventsRequest(_message.Message):
    __slots__ = ("session_id", "buffer")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    BUFFER_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    buffer: int
    def __init__(self, session_id: _Optional[str] = ..., buffer: _Optional[int] = ...) -> None: ...

class SubscribeEventsResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: _events_pb2.Event
    def __init__(self, event: _Optional[_Union[_events_pb2.Event, _Mapping]] = ...) -> None: ...
