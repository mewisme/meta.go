import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from meta.v1 import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ThreadSystemKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    THREAD_SYSTEM_KIND_UNSPECIFIED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_NICKNAME_UPDATED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_EMOJI_UPDATED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_APPROVAL_MODE_UPDATED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_THEME_UPDATED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_MEMBER_ADDED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_MEMBER_REMOVED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_PARTICIPANT_ADMIN_UPDATED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_MESSAGE_PIN_UPDATED: _ClassVar[ThreadSystemKind]
    THREAD_SYSTEM_KIND_POLL_UPDATED: _ClassVar[ThreadSystemKind]
THREAD_SYSTEM_KIND_UNSPECIFIED: ThreadSystemKind
THREAD_SYSTEM_KIND_NICKNAME_UPDATED: ThreadSystemKind
THREAD_SYSTEM_KIND_EMOJI_UPDATED: ThreadSystemKind
THREAD_SYSTEM_KIND_APPROVAL_MODE_UPDATED: ThreadSystemKind
THREAD_SYSTEM_KIND_THEME_UPDATED: ThreadSystemKind
THREAD_SYSTEM_KIND_MEMBER_ADDED: ThreadSystemKind
THREAD_SYSTEM_KIND_MEMBER_REMOVED: ThreadSystemKind
THREAD_SYSTEM_KIND_PARTICIPANT_ADMIN_UPDATED: ThreadSystemKind
THREAD_SYSTEM_KIND_MESSAGE_PIN_UPDATED: ThreadSystemKind
THREAD_SYSTEM_KIND_POLL_UPDATED: ThreadSystemKind

class Event(_message.Message):
    __slots__ = ("session_id", "sequence", "emitted_at", "ready", "reconnected", "disconnected", "error", "message", "reaction", "typing", "read_receipt", "delivery_receipt", "message_edit", "message_unsend", "thread_update", "e2ee_ready", "e2ee_receipt", "thread_system")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    RECONNECTED_FIELD_NUMBER: _ClassVar[int]
    DISCONNECTED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REACTION_FIELD_NUMBER: _ClassVar[int]
    TYPING_FIELD_NUMBER: _ClassVar[int]
    READ_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_EDIT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_UNSEND_FIELD_NUMBER: _ClassVar[int]
    THREAD_UPDATE_FIELD_NUMBER: _ClassVar[int]
    E2EE_READY_FIELD_NUMBER: _ClassVar[int]
    E2EE_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    THREAD_SYSTEM_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    sequence: int
    emitted_at: _timestamp_pb2.Timestamp
    ready: ReadyEvent
    reconnected: ReconnectedEvent
    disconnected: DisconnectedEvent
    error: ErrorEvent
    message: MessageEvent
    reaction: ReactionEvent
    typing: TypingEvent
    read_receipt: ReadReceiptEvent
    delivery_receipt: DeliveryReceiptEvent
    message_edit: MessageEditEvent
    message_unsend: MessageUnsendEvent
    thread_update: ThreadUpdateEvent
    e2ee_ready: E2EEReadyEvent
    e2ee_receipt: E2EEReceiptEvent
    thread_system: ThreadSystemEvent
    def __init__(self, session_id: _Optional[str] = ..., sequence: _Optional[int] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ready: _Optional[_Union[ReadyEvent, _Mapping]] = ..., reconnected: _Optional[_Union[ReconnectedEvent, _Mapping]] = ..., disconnected: _Optional[_Union[DisconnectedEvent, _Mapping]] = ..., error: _Optional[_Union[ErrorEvent, _Mapping]] = ..., message: _Optional[_Union[MessageEvent, _Mapping]] = ..., reaction: _Optional[_Union[ReactionEvent, _Mapping]] = ..., typing: _Optional[_Union[TypingEvent, _Mapping]] = ..., read_receipt: _Optional[_Union[ReadReceiptEvent, _Mapping]] = ..., delivery_receipt: _Optional[_Union[DeliveryReceiptEvent, _Mapping]] = ..., message_edit: _Optional[_Union[MessageEditEvent, _Mapping]] = ..., message_unsend: _Optional[_Union[MessageUnsendEvent, _Mapping]] = ..., thread_update: _Optional[_Union[ThreadUpdateEvent, _Mapping]] = ..., e2ee_ready: _Optional[_Union[E2EEReadyEvent, _Mapping]] = ..., e2ee_receipt: _Optional[_Union[E2EEReceiptEvent, _Mapping]] = ..., thread_system: _Optional[_Union[ThreadSystemEvent, _Mapping]] = ...) -> None: ...

class ReadyEvent(_message.Message):
    __slots__ = ("is_new_session",)
    IS_NEW_SESSION_FIELD_NUMBER: _ClassVar[int]
    is_new_session: bool
    def __init__(self, is_new_session: _Optional[bool] = ...) -> None: ...

class ReconnectedEvent(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DisconnectedEvent(_message.Message):
    __slots__ = ("transport",)
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    transport: _common_pb2.TransportKind
    def __init__(self, transport: _Optional[_Union[_common_pb2.TransportKind, str]] = ...) -> None: ...

class ErrorEvent(_message.Message):
    __slots__ = ("error",)
    ERROR_FIELD_NUMBER: _ClassVar[int]
    error: _common_pb2.ErrorDetail
    def __init__(self, error: _Optional[_Union[_common_pb2.ErrorDetail, _Mapping]] = ...) -> None: ...

class MessageEvent(_message.Message):
    __slots__ = ("message",)
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    message: _common_pb2.Message
    def __init__(self, message: _Optional[_Union[_common_pb2.Message, _Mapping]] = ...) -> None: ...

class ReactionEvent(_message.Message):
    __slots__ = ("message_id", "thread_id", "actor_id", "reaction")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    REACTION_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    thread_id: str
    actor_id: str
    reaction: str
    def __init__(self, message_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., actor_id: _Optional[str] = ..., reaction: _Optional[str] = ...) -> None: ...

class TypingEvent(_message.Message):
    __slots__ = ("thread_id", "sender_id", "typing")
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    SENDER_ID_FIELD_NUMBER: _ClassVar[int]
    TYPING_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    sender_id: str
    typing: bool
    def __init__(self, thread_id: _Optional[str] = ..., sender_id: _Optional[str] = ..., typing: _Optional[bool] = ...) -> None: ...

class ReadReceiptEvent(_message.Message):
    __slots__ = ("thread_id", "reader_id", "watermark")
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    READER_ID_FIELD_NUMBER: _ClassVar[int]
    WATERMARK_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    reader_id: str
    watermark: _timestamp_pb2.Timestamp
    def __init__(self, thread_id: _Optional[str] = ..., reader_id: _Optional[str] = ..., watermark: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DeliveryReceiptEvent(_message.Message):
    __slots__ = ("thread_id", "recipient_id", "watermark")
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPIENT_ID_FIELD_NUMBER: _ClassVar[int]
    WATERMARK_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    recipient_id: str
    watermark: _timestamp_pb2.Timestamp
    def __init__(self, thread_id: _Optional[str] = ..., recipient_id: _Optional[str] = ..., watermark: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MessageEditEvent(_message.Message):
    __slots__ = ("message_id", "text", "edit_count")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    EDIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    text: str
    edit_count: int
    def __init__(self, message_id: _Optional[str] = ..., text: _Optional[str] = ..., edit_count: _Optional[int] = ...) -> None: ...

class MessageUnsendEvent(_message.Message):
    __slots__ = ("message_id", "thread_id")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    thread_id: str
    def __init__(self, message_id: _Optional[str] = ..., thread_id: _Optional[str] = ...) -> None: ...

class ThreadUpdateEvent(_message.Message):
    __slots__ = ("thread_id", "field", "value")
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    field: str
    value: str
    def __init__(self, thread_id: _Optional[str] = ..., field: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class ThreadSystemEvent(_message.Message):
    __slots__ = ("kind", "thread_id", "participant_id", "message_id", "poll_id", "nickname", "emoji", "enabled", "pinned", "is_admin")
    KIND_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    PARTICIPANT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    POLL_ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    EMOJI_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    IS_ADMIN_FIELD_NUMBER: _ClassVar[int]
    kind: ThreadSystemKind
    thread_id: str
    participant_id: str
    message_id: str
    poll_id: str
    nickname: str
    emoji: str
    enabled: bool
    pinned: bool
    is_admin: bool
    def __init__(self, kind: _Optional[_Union[ThreadSystemKind, str]] = ..., thread_id: _Optional[str] = ..., participant_id: _Optional[str] = ..., message_id: _Optional[str] = ..., poll_id: _Optional[str] = ..., nickname: _Optional[str] = ..., emoji: _Optional[str] = ..., enabled: _Optional[bool] = ..., pinned: _Optional[bool] = ..., is_admin: _Optional[bool] = ...) -> None: ...

class E2EEReadyEvent(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class E2EEReceiptEvent(_message.Message):
    __slots__ = ("type", "chat_jid", "sender_jid", "message_ids")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    SENDER_JID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_IDS_FIELD_NUMBER: _ClassVar[int]
    type: str
    chat_jid: str
    sender_jid: str
    message_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, type: _Optional[str] = ..., chat_jid: _Optional[str] = ..., sender_jid: _Optional[str] = ..., message_ids: _Optional[_Iterable[str]] = ...) -> None: ...
