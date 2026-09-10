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

class E2EEMediaKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    E2EE_MEDIA_KIND_UNSPECIFIED: _ClassVar[E2EEMediaKind]
    E2EE_MEDIA_KIND_IMAGE: _ClassVar[E2EEMediaKind]
    E2EE_MEDIA_KIND_VIDEO: _ClassVar[E2EEMediaKind]
    E2EE_MEDIA_KIND_AUDIO: _ClassVar[E2EEMediaKind]
    E2EE_MEDIA_KIND_DOCUMENT: _ClassVar[E2EEMediaKind]
    E2EE_MEDIA_KIND_STICKER: _ClassVar[E2EEMediaKind]
E2EE_MEDIA_KIND_UNSPECIFIED: E2EEMediaKind
E2EE_MEDIA_KIND_IMAGE: E2EEMediaKind
E2EE_MEDIA_KIND_VIDEO: E2EEMediaKind
E2EE_MEDIA_KIND_AUDIO: E2EEMediaKind
E2EE_MEDIA_KIND_DOCUMENT: E2EEMediaKind
E2EE_MEDIA_KIND_STICKER: E2EEMediaKind

class E2EEServiceSendTextRequest(_message.Message):
    __slots__ = ("session_id", "chat_jid", "facebook_user_id", "text", "reply_to", "reply_sender_jid")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    FACEBOOK_USER_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    REPLY_TO_FIELD_NUMBER: _ClassVar[int]
    REPLY_SENDER_JID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    facebook_user_id: str
    text: str
    reply_to: str
    reply_sender_jid: str
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., facebook_user_id: _Optional[str] = ..., text: _Optional[str] = ..., reply_to: _Optional[str] = ..., reply_sender_jid: _Optional[str] = ...) -> None: ...

class E2EEServiceSendTextResponse(_message.Message):
    __slots__ = ("message_id", "timestamp")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, message_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class E2EEServiceSendMediaMetadata(_message.Message):
    __slots__ = ("session_id", "chat_jid", "facebook_user_id", "kind", "name", "content_type", "size", "caption", "width", "height", "duration", "voice", "reply_to", "reply_sender_jid", "sha256")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    FACEBOOK_USER_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    REPLY_TO_FIELD_NUMBER: _ClassVar[int]
    REPLY_SENDER_JID_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    facebook_user_id: str
    kind: E2EEMediaKind
    name: str
    content_type: str
    size: int
    caption: str
    width: int
    height: int
    duration: int
    voice: bool
    reply_to: str
    reply_sender_jid: str
    sha256: bytes
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., facebook_user_id: _Optional[str] = ..., kind: _Optional[_Union[E2EEMediaKind, str]] = ..., name: _Optional[str] = ..., content_type: _Optional[str] = ..., size: _Optional[int] = ..., caption: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., duration: _Optional[int] = ..., voice: _Optional[bool] = ..., reply_to: _Optional[str] = ..., reply_sender_jid: _Optional[str] = ..., sha256: _Optional[bytes] = ...) -> None: ...

class E2EEServiceSendMediaRequest(_message.Message):
    __slots__ = ("metadata", "chunk")
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    metadata: E2EEServiceSendMediaMetadata
    chunk: _common_pb2.MediaChunk
    def __init__(self, metadata: _Optional[_Union[E2EEServiceSendMediaMetadata, _Mapping]] = ..., chunk: _Optional[_Union[_common_pb2.MediaChunk, _Mapping]] = ...) -> None: ...

class E2EEServiceSendMediaResponse(_message.Message):
    __slots__ = ("message_id", "timestamp", "sha256")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    timestamp: _timestamp_pb2.Timestamp
    sha256: bytes
    def __init__(self, message_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sha256: _Optional[bytes] = ...) -> None: ...

class E2EEServiceDownloadMediaRequest(_message.Message):
    __slots__ = ("session_id", "reference")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    reference: _common_pb2.E2EEMediaReference
    def __init__(self, session_id: _Optional[str] = ..., reference: _Optional[_Union[_common_pb2.E2EEMediaReference, _Mapping]] = ...) -> None: ...

class E2EEServiceDownloadMediaMetadata(_message.Message):
    __slots__ = ("size", "content_type", "sha256")
    SIZE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    size: int
    content_type: str
    sha256: bytes
    def __init__(self, size: _Optional[int] = ..., content_type: _Optional[str] = ..., sha256: _Optional[bytes] = ...) -> None: ...

class E2EEServiceDownloadMediaResponse(_message.Message):
    __slots__ = ("metadata", "chunk")
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    metadata: E2EEServiceDownloadMediaMetadata
    chunk: _common_pb2.MediaChunk
    def __init__(self, metadata: _Optional[_Union[E2EEServiceDownloadMediaMetadata, _Mapping]] = ..., chunk: _Optional[_Union[_common_pb2.MediaChunk, _Mapping]] = ...) -> None: ...

class E2EEServiceReactRequest(_message.Message):
    __slots__ = ("session_id", "chat_jid", "message_id", "sender_jid", "reaction")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    SENDER_JID_FIELD_NUMBER: _ClassVar[int]
    REACTION_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    message_id: str
    sender_jid: str
    reaction: str
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., message_id: _Optional[str] = ..., sender_jid: _Optional[str] = ..., reaction: _Optional[str] = ...) -> None: ...

class E2EEServiceReactResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class E2EEServiceEditRequest(_message.Message):
    __slots__ = ("session_id", "chat_jid", "message_id", "text")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    message_id: str
    text: str
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., message_id: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class E2EEServiceEditResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class E2EEServiceUnsendRequest(_message.Message):
    __slots__ = ("session_id", "chat_jid", "message_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    message_id: str
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., message_id: _Optional[str] = ...) -> None: ...

class E2EEServiceUnsendResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class E2EEServiceSetTypingRequest(_message.Message):
    __slots__ = ("session_id", "chat_jid", "typing")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    TYPING_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    typing: bool
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., typing: _Optional[bool] = ...) -> None: ...

class E2EEServiceSetTypingResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class E2EEServiceMarkReadRequest(_message.Message):
    __slots__ = ("session_id", "chat_jid", "sender_jid", "message_ids", "timestamp")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_JID_FIELD_NUMBER: _ClassVar[int]
    SENDER_JID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_IDS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    chat_jid: str
    sender_jid: str
    message_ids: _containers.RepeatedScalarFieldContainer[str]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, session_id: _Optional[str] = ..., chat_jid: _Optional[str] = ..., sender_jid: _Optional[str] = ..., message_ids: _Optional[_Iterable[str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class E2EEServiceMarkReadResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
