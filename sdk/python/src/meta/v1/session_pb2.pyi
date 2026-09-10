import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from meta.v1 import common_pb2 as _common_pb2
from meta.v1 import events_pb2 as _events_pb2
from meta.v1 import health_pb2 as _health_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateSessionRequest(_message.Message):
    __slots__ = ("cookies", "e2ee", "event_buffer", "timeout")
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
    cookies: _containers.ScalarMap[str, str]
    e2ee: bool
    event_buffer: int
    timeout: _duration_pb2.Duration
    def __init__(self, cookies: _Optional[_Mapping[str, str]] = ..., e2ee: _Optional[bool] = ..., event_buffer: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...

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
