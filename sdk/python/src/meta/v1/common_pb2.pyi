import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ErrorCategory(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ERROR_CATEGORY_UNSPECIFIED: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_UNKNOWN: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_AUTH: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_CHECKPOINT: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_RATE_LIMIT: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_INVALID_INPUT: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_CONNECTION: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_E2EE: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_PROTOCOL: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_PERMISSION: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_NETWORK: _ClassVar[ErrorCategory]
    ERROR_CATEGORY_CANCELED: _ClassVar[ErrorCategory]

class TransportKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRANSPORT_KIND_UNSPECIFIED: _ClassVar[TransportKind]
    TRANSPORT_KIND_MESSENGER: _ClassVar[TransportKind]
    TRANSPORT_KIND_E2EE: _ClassVar[TransportKind]

class EncryptionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ENCRYPTION_KIND_UNSPECIFIED: _ClassVar[EncryptionKind]
    ENCRYPTION_KIND_NONE: _ClassVar[EncryptionKind]
    ENCRYPTION_KIND_E2EE: _ClassVar[EncryptionKind]
ERROR_CATEGORY_UNSPECIFIED: ErrorCategory
ERROR_CATEGORY_UNKNOWN: ErrorCategory
ERROR_CATEGORY_AUTH: ErrorCategory
ERROR_CATEGORY_CHECKPOINT: ErrorCategory
ERROR_CATEGORY_RATE_LIMIT: ErrorCategory
ERROR_CATEGORY_INVALID_INPUT: ErrorCategory
ERROR_CATEGORY_CONNECTION: ErrorCategory
ERROR_CATEGORY_E2EE: ErrorCategory
ERROR_CATEGORY_PROTOCOL: ErrorCategory
ERROR_CATEGORY_PERMISSION: ErrorCategory
ERROR_CATEGORY_NETWORK: ErrorCategory
ERROR_CATEGORY_CANCELED: ErrorCategory
TRANSPORT_KIND_UNSPECIFIED: TransportKind
TRANSPORT_KIND_MESSENGER: TransportKind
TRANSPORT_KIND_E2EE: TransportKind
ENCRYPTION_KIND_UNSPECIFIED: EncryptionKind
ENCRYPTION_KIND_NONE: EncryptionKind
ENCRYPTION_KIND_E2EE: EncryptionKind

class ProtocolVersion(_message.Message):
    __slots__ = ("major", "minor")
    MAJOR_FIELD_NUMBER: _ClassVar[int]
    MINOR_FIELD_NUMBER: _ClassVar[int]
    major: int
    minor: int
    def __init__(self, major: _Optional[int] = ..., minor: _Optional[int] = ...) -> None: ...

class BuildInfo(_message.Message):
    __slots__ = ("version", "commit", "go_version", "os", "arch")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    GO_VERSION_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    version: str
    commit: str
    go_version: str
    os: str
    arch: str
    def __init__(self, version: _Optional[str] = ..., commit: _Optional[str] = ..., go_version: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ...) -> None: ...

class ErrorDetail(_message.Message):
    __slots__ = ("category", "code", "message", "retryable", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    category: ErrorCategory
    code: str
    message: str
    retryable: bool
    details: _containers.ScalarMap[str, str]
    def __init__(self, category: _Optional[_Union[ErrorCategory, str]] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Account(_message.Message):
    __slots__ = ("id", "name", "username")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    username: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., username: _Optional[str] = ...) -> None: ...

class ReplyReference(_message.Message):
    __slots__ = ("message_id", "sender_id")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    SENDER_ID_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    sender_id: str
    def __init__(self, message_id: _Optional[str] = ..., sender_id: _Optional[str] = ...) -> None: ...

class Mention(_message.Message):
    __slots__ = ("user_id", "offset", "length")
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    LENGTH_FIELD_NUMBER: _ClassVar[int]
    user_id: str
    offset: int
    length: int
    def __init__(self, user_id: _Optional[str] = ..., offset: _Optional[int] = ..., length: _Optional[int] = ...) -> None: ...

class E2EEMediaReference(_message.Message):
    __slots__ = ("kind", "direct_path", "media_key", "file_sha256", "file_enc_sha256", "content_type", "size")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DIRECT_PATH_FIELD_NUMBER: _ClassVar[int]
    MEDIA_KEY_FIELD_NUMBER: _ClassVar[int]
    FILE_SHA256_FIELD_NUMBER: _ClassVar[int]
    FILE_ENC_SHA256_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    direct_path: str
    media_key: bytes
    file_sha256: bytes
    file_enc_sha256: bytes
    content_type: str
    size: int
    def __init__(self, kind: _Optional[str] = ..., direct_path: _Optional[str] = ..., media_key: _Optional[bytes] = ..., file_sha256: _Optional[bytes] = ..., file_enc_sha256: _Optional[bytes] = ..., content_type: _Optional[str] = ..., size: _Optional[int] = ...) -> None: ...

class Attachment(_message.Message):
    __slots__ = ("id", "type", "url", "preview_url", "file_name", "content_type", "size", "width", "height", "duration", "e2ee")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    E2EE_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    url: str
    preview_url: str
    file_name: str
    content_type: str
    size: int
    width: int
    height: int
    duration: int
    e2ee: E2EEMediaReference
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., url: _Optional[str] = ..., preview_url: _Optional[str] = ..., file_name: _Optional[str] = ..., content_type: _Optional[str] = ..., size: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., duration: _Optional[int] = ..., e2ee: _Optional[_Union[E2EEMediaReference, _Mapping]] = ...) -> None: ...

class Message(_message.Message):
    __slots__ = ("id", "thread_id", "sender_id", "text", "timestamp", "reply_to", "mentions", "attachments", "encryption", "transport")
    ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    SENDER_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    REPLY_TO_FIELD_NUMBER: _ClassVar[int]
    MENTIONS_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    ENCRYPTION_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    id: str
    thread_id: str
    sender_id: str
    text: str
    timestamp: _timestamp_pb2.Timestamp
    reply_to: ReplyReference
    mentions: _containers.RepeatedCompositeFieldContainer[Mention]
    attachments: _containers.RepeatedCompositeFieldContainer[Attachment]
    encryption: EncryptionKind
    transport: TransportKind
    def __init__(self, id: _Optional[str] = ..., thread_id: _Optional[str] = ..., sender_id: _Optional[str] = ..., text: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reply_to: _Optional[_Union[ReplyReference, _Mapping]] = ..., mentions: _Optional[_Iterable[_Union[Mention, _Mapping]]] = ..., attachments: _Optional[_Iterable[_Union[Attachment, _Mapping]]] = ..., encryption: _Optional[_Union[EncryptionKind, str]] = ..., transport: _Optional[_Union[TransportKind, str]] = ...) -> None: ...

class MediaChunk(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    def __init__(self, data: _Optional[bytes] = ...) -> None: ...
