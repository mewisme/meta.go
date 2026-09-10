import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConnectionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONNECTION_STATE_UNSPECIFIED: _ClassVar[ConnectionState]
    CONNECTION_STATE_DISCONNECTED: _ClassVar[ConnectionState]
    CONNECTION_STATE_CONNECTING: _ClassVar[ConnectionState]
    CONNECTION_STATE_CONNECTED: _ClassVar[ConnectionState]
    CONNECTION_STATE_FAILED: _ClassVar[ConnectionState]
CONNECTION_STATE_UNSPECIFIED: ConnectionState
CONNECTION_STATE_DISCONNECTED: ConnectionState
CONNECTION_STATE_CONNECTING: ConnectionState
CONNECTION_STATE_CONNECTED: ConnectionState
CONNECTION_STATE_FAILED: ConnectionState

class HealthSnapshot(_message.Message):
    __slots__ = ("regular", "e2ee", "reconnect_count", "engine_dropped_event_count", "subscriber_dropped_event_count", "last_successful_send", "last_receive", "last_error_category")
    REGULAR_FIELD_NUMBER: _ClassVar[int]
    E2EE_FIELD_NUMBER: _ClassVar[int]
    RECONNECT_COUNT_FIELD_NUMBER: _ClassVar[int]
    ENGINE_DROPPED_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUBSCRIBER_DROPPED_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESSFUL_SEND_FIELD_NUMBER: _ClassVar[int]
    LAST_RECEIVE_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_CATEGORY_FIELD_NUMBER: _ClassVar[int]
    regular: ConnectionState
    e2ee: ConnectionState
    reconnect_count: int
    engine_dropped_event_count: int
    subscriber_dropped_event_count: int
    last_successful_send: _timestamp_pb2.Timestamp
    last_receive: _timestamp_pb2.Timestamp
    last_error_category: str
    def __init__(self, regular: _Optional[_Union[ConnectionState, str]] = ..., e2ee: _Optional[_Union[ConnectionState, str]] = ..., reconnect_count: _Optional[int] = ..., engine_dropped_event_count: _Optional[int] = ..., subscriber_dropped_event_count: _Optional[int] = ..., last_successful_send: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_receive: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_error_category: _Optional[str] = ...) -> None: ...
