import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from meta.v1 import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EncryptionPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ENCRYPTION_POLICY_UNSPECIFIED: _ClassVar[EncryptionPolicy]
    ENCRYPTION_POLICY_AUTO: _ClassVar[EncryptionPolicy]
    ENCRYPTION_POLICY_REQUIRED: _ClassVar[EncryptionPolicy]
    ENCRYPTION_POLICY_DISABLED: _ClassVar[EncryptionPolicy]

class MessengerBlockStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MESSENGER_BLOCK_STATUS_UNSPECIFIED: _ClassVar[MessengerBlockStatus]
    MESSENGER_BLOCK_STATUS_UNKNOWN: _ClassVar[MessengerBlockStatus]
    MESSENGER_BLOCK_STATUS_UNBLOCKED: _ClassVar[MessengerBlockStatus]
    MESSENGER_BLOCK_STATUS_MESSAGE_BLOCKED: _ClassVar[MessengerBlockStatus]
    MESSENGER_BLOCK_STATUS_FULLY_BLOCKED: _ClassVar[MessengerBlockStatus]
ENCRYPTION_POLICY_UNSPECIFIED: EncryptionPolicy
ENCRYPTION_POLICY_AUTO: EncryptionPolicy
ENCRYPTION_POLICY_REQUIRED: EncryptionPolicy
ENCRYPTION_POLICY_DISABLED: EncryptionPolicy
MESSENGER_BLOCK_STATUS_UNSPECIFIED: MessengerBlockStatus
MESSENGER_BLOCK_STATUS_UNKNOWN: MessengerBlockStatus
MESSENGER_BLOCK_STATUS_UNBLOCKED: MessengerBlockStatus
MESSENGER_BLOCK_STATUS_MESSAGE_BLOCKED: MessengerBlockStatus
MESSENGER_BLOCK_STATUS_FULLY_BLOCKED: MessengerBlockStatus

class User(_message.Message):
    __slots__ = ("id", "name", "first_name", "username", "profile_url", "avatar_url", "gender", "is_messenger_user", "is_verified", "can_viewer_message", "messenger_restricted", "messenger_block_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FIRST_NAME_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    PROFILE_URL_FIELD_NUMBER: _ClassVar[int]
    AVATAR_URL_FIELD_NUMBER: _ClassVar[int]
    GENDER_FIELD_NUMBER: _ClassVar[int]
    IS_MESSENGER_USER_FIELD_NUMBER: _ClassVar[int]
    IS_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    CAN_VIEWER_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    MESSENGER_RESTRICTED_FIELD_NUMBER: _ClassVar[int]
    MESSENGER_BLOCK_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    first_name: str
    username: str
    profile_url: str
    avatar_url: str
    gender: str
    is_messenger_user: bool
    is_verified: bool
    can_viewer_message: bool
    messenger_restricted: bool
    messenger_block_status: MessengerBlockStatus
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., first_name: _Optional[str] = ..., username: _Optional[str] = ..., profile_url: _Optional[str] = ..., avatar_url: _Optional[str] = ..., gender: _Optional[str] = ..., is_messenger_user: _Optional[bool] = ..., is_verified: _Optional[bool] = ..., can_viewer_message: _Optional[bool] = ..., messenger_restricted: _Optional[bool] = ..., messenger_block_status: _Optional[_Union[MessengerBlockStatus, str]] = ...) -> None: ...

class Thread(_message.Message):
    __slots__ = ("id", "name", "type", "participants", "admins", "nicknames", "emoji", "message_count", "approval_mode", "joinable", "joinable_url", "last_activity")
    class NicknamesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PARTICIPANTS_FIELD_NUMBER: _ClassVar[int]
    ADMINS_FIELD_NUMBER: _ClassVar[int]
    NICKNAMES_FIELD_NUMBER: _ClassVar[int]
    EMOJI_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_MODE_FIELD_NUMBER: _ClassVar[int]
    JOINABLE_FIELD_NUMBER: _ClassVar[int]
    JOINABLE_URL_FIELD_NUMBER: _ClassVar[int]
    LAST_ACTIVITY_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    type: str
    participants: _containers.RepeatedCompositeFieldContainer[User]
    admins: _containers.RepeatedScalarFieldContainer[str]
    nicknames: _containers.ScalarMap[str, str]
    emoji: str
    message_count: int
    approval_mode: bool
    joinable: bool
    joinable_url: str
    last_activity: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., type: _Optional[str] = ..., participants: _Optional[_Iterable[_Union[User, _Mapping]]] = ..., admins: _Optional[_Iterable[str]] = ..., nicknames: _Optional[_Mapping[str, str]] = ..., emoji: _Optional[str] = ..., message_count: _Optional[int] = ..., approval_mode: _Optional[bool] = ..., joinable: _Optional[bool] = ..., joinable_url: _Optional[str] = ..., last_activity: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PinnedMessage(_message.Message):
    __slots__ = ("thread_id", "message_id", "pinned_at", "authority_level")
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    PINNED_AT_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_LEVEL_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    message_id: str
    pinned_at: _timestamp_pb2.Timestamp
    authority_level: int
    def __init__(self, thread_id: _Optional[str] = ..., message_id: _Optional[str] = ..., pinned_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., authority_level: _Optional[int] = ...) -> None: ...

class PollOption(_message.Message):
    __slots__ = ("id", "text", "sort_key_voting_timestamp", "sort_key_creation_timestamp")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SORT_KEY_VOTING_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SORT_KEY_CREATION_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    id: str
    text: str
    sort_key_voting_timestamp: _timestamp_pb2.Timestamp
    sort_key_creation_timestamp: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., text: _Optional[str] = ..., sort_key_voting_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sort_key_creation_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PollVote(_message.Message):
    __slots__ = ("option_id", "contact_id", "timestamp", "vote_count", "thread_id", "message_id")
    OPTION_ID_FIELD_NUMBER: _ClassVar[int]
    CONTACT_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    VOTE_COUNT_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    option_id: str
    contact_id: str
    timestamp: _timestamp_pb2.Timestamp
    vote_count: int
    thread_id: str
    message_id: str
    def __init__(self, option_id: _Optional[str] = ..., contact_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., vote_count: _Optional[int] = ..., thread_id: _Optional[str] = ..., message_id: _Optional[str] = ...) -> None: ...

class PollDetails(_message.Message):
    __slots__ = ("id", "thread_id", "title", "last_update_message_id", "last_update_message_timestamp", "last_update_message_event_type", "options", "votes")
    ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_MESSAGE_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_MESSAGE_EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    VOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    thread_id: str
    title: str
    last_update_message_id: str
    last_update_message_timestamp: _timestamp_pb2.Timestamp
    last_update_message_event_type: int
    options: _containers.RepeatedCompositeFieldContainer[PollOption]
    votes: _containers.RepeatedCompositeFieldContainer[PollVote]
    def __init__(self, id: _Optional[str] = ..., thread_id: _Optional[str] = ..., title: _Optional[str] = ..., last_update_message_id: _Optional[str] = ..., last_update_message_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_update_message_event_type: _Optional[int] = ..., options: _Optional[_Iterable[_Union[PollOption, _Mapping]]] = ..., votes: _Optional[_Iterable[_Union[PollVote, _Mapping]]] = ...) -> None: ...

class MessageSearchHighlight(_message.Message):
    __slots__ = ("offset", "length")
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    LENGTH_FIELD_NUMBER: _ClassVar[int]
    offset: int
    length: int
    def __init__(self, offset: _Optional[int] = ..., length: _Optional[int] = ...) -> None: ...

class MessageSearchResult(_message.Message):
    __slots__ = ("message_id", "thread_id", "thread_type", "global_index", "sender_name", "sender_avatar_url", "timestamp", "text", "highlights")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_TYPE_FIELD_NUMBER: _ClassVar[int]
    GLOBAL_INDEX_FIELD_NUMBER: _ClassVar[int]
    SENDER_NAME_FIELD_NUMBER: _ClassVar[int]
    SENDER_AVATAR_URL_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHTS_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    thread_id: str
    thread_type: str
    global_index: int
    sender_name: str
    sender_avatar_url: str
    timestamp: _timestamp_pb2.Timestamp
    text: str
    highlights: _containers.RepeatedCompositeFieldContainer[MessageSearchHighlight]
    def __init__(self, message_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., thread_type: _Optional[str] = ..., global_index: _Optional[int] = ..., sender_name: _Optional[str] = ..., sender_avatar_url: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., text: _Optional[str] = ..., highlights: _Optional[_Iterable[_Union[MessageSearchHighlight, _Mapping]]] = ...) -> None: ...

class MessageRequest(_message.Message):
    __slots__ = ("sender_id", "snippet", "timestamp")
    SENDER_ID_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    sender_id: str
    snippet: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, sender_id: _Optional[str] = ..., snippet: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Theme(_message.Message):
    __slots__ = ("id", "name", "description", "app_color_mode", "composer_background_color", "background_gradient_colors", "title_bar_button_tint_color", "inbound_message_gradient_colors", "title_bar_text_color", "composer_tint_color", "title_bar_attribution_color", "composer_input_background_color", "hot_like_color", "background_image", "message_text_color", "inbound_message_text_color", "primary_button_background_color", "title_bar_background_color", "tertiary_text_color", "reaction_pill_background_color", "secondary_text_color", "fallback_color", "gradient_colors", "normal_theme_id", "icon_asset")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    APP_COLOR_MODE_FIELD_NUMBER: _ClassVar[int]
    COMPOSER_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_GRADIENT_COLORS_FIELD_NUMBER: _ClassVar[int]
    TITLE_BAR_BUTTON_TINT_COLOR_FIELD_NUMBER: _ClassVar[int]
    INBOUND_MESSAGE_GRADIENT_COLORS_FIELD_NUMBER: _ClassVar[int]
    TITLE_BAR_TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    COMPOSER_TINT_COLOR_FIELD_NUMBER: _ClassVar[int]
    TITLE_BAR_ATTRIBUTION_COLOR_FIELD_NUMBER: _ClassVar[int]
    COMPOSER_INPUT_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    HOT_LIKE_COLOR_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_IMAGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    INBOUND_MESSAGE_TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_BUTTON_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    TITLE_BAR_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    TERTIARY_TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    REACTION_PILL_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    SECONDARY_TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_COLOR_FIELD_NUMBER: _ClassVar[int]
    GRADIENT_COLORS_FIELD_NUMBER: _ClassVar[int]
    NORMAL_THEME_ID_FIELD_NUMBER: _ClassVar[int]
    ICON_ASSET_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    app_color_mode: str
    composer_background_color: str
    background_gradient_colors: _containers.RepeatedScalarFieldContainer[str]
    title_bar_button_tint_color: str
    inbound_message_gradient_colors: _containers.RepeatedScalarFieldContainer[str]
    title_bar_text_color: str
    composer_tint_color: str
    title_bar_attribution_color: str
    composer_input_background_color: str
    hot_like_color: str
    background_image: str
    message_text_color: str
    inbound_message_text_color: str
    primary_button_background_color: str
    title_bar_background_color: str
    tertiary_text_color: str
    reaction_pill_background_color: str
    secondary_text_color: str
    fallback_color: str
    gradient_colors: _containers.RepeatedScalarFieldContainer[str]
    normal_theme_id: str
    icon_asset: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., app_color_mode: _Optional[str] = ..., composer_background_color: _Optional[str] = ..., background_gradient_colors: _Optional[_Iterable[str]] = ..., title_bar_button_tint_color: _Optional[str] = ..., inbound_message_gradient_colors: _Optional[_Iterable[str]] = ..., title_bar_text_color: _Optional[str] = ..., composer_tint_color: _Optional[str] = ..., title_bar_attribution_color: _Optional[str] = ..., composer_input_background_color: _Optional[str] = ..., hot_like_color: _Optional[str] = ..., background_image: _Optional[str] = ..., message_text_color: _Optional[str] = ..., inbound_message_text_color: _Optional[str] = ..., primary_button_background_color: _Optional[str] = ..., title_bar_background_color: _Optional[str] = ..., tertiary_text_color: _Optional[str] = ..., reaction_pill_background_color: _Optional[str] = ..., secondary_text_color: _Optional[str] = ..., fallback_color: _Optional[str] = ..., gradient_colors: _Optional[_Iterable[str]] = ..., normal_theme_id: _Optional[str] = ..., icon_asset: _Optional[str] = ...) -> None: ...

class Note(_message.Message):
    __slots__ = ("id", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    description: str
    def __init__(self, id: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class SendRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "text", "reply_to", "mentions", "sticker_id", "url", "encryption", "attachment_ids")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    REPLY_TO_FIELD_NUMBER: _ClassVar[int]
    MENTIONS_FIELD_NUMBER: _ClassVar[int]
    STICKER_ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    ENCRYPTION_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    text: str
    reply_to: _common_pb2.ReplyReference
    mentions: _containers.RepeatedCompositeFieldContainer[_common_pb2.Mention]
    sticker_id: str
    url: str
    encryption: EncryptionPolicy
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., text: _Optional[str] = ..., reply_to: _Optional[_Union[_common_pb2.ReplyReference, _Mapping]] = ..., mentions: _Optional[_Iterable[_Union[_common_pb2.Mention, _Mapping]]] = ..., sticker_id: _Optional[str] = ..., url: _Optional[str] = ..., encryption: _Optional[_Union[EncryptionPolicy, str]] = ..., attachment_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class SendResponse(_message.Message):
    __slots__ = ("message_id", "timestamp")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, message_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UploadMetadata(_message.Message):
    __slots__ = ("session_id", "thread_id", "name", "content_type", "size", "voice", "sha256")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    name: str
    content_type: str
    size: int
    voice: bool
    sha256: bytes
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., name: _Optional[str] = ..., content_type: _Optional[str] = ..., size: _Optional[int] = ..., voice: _Optional[bool] = ..., sha256: _Optional[bytes] = ...) -> None: ...

class UploadRequest(_message.Message):
    __slots__ = ("metadata", "chunk")
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    metadata: UploadMetadata
    chunk: _common_pb2.MediaChunk
    def __init__(self, metadata: _Optional[_Union[UploadMetadata, _Mapping]] = ..., chunk: _Optional[_Union[_common_pb2.MediaChunk, _Mapping]] = ...) -> None: ...

class UploadResponse(_message.Message):
    __slots__ = ("id", "name", "content_type", "type", "sha256")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    content_type: str
    type: str
    sha256: bytes
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., content_type: _Optional[str] = ..., type: _Optional[str] = ..., sha256: _Optional[bytes] = ...) -> None: ...

class ForwardRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "message_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    message_id: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., message_id: _Optional[str] = ...) -> None: ...

class ForwardResponse(_message.Message):
    __slots__ = ("message_id", "timestamp")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, message_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ShareContactRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "contact_id", "text")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    CONTACT_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    contact_id: str
    text: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., contact_id: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class ShareContactResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReactRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "message_id", "reaction")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    REACTION_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    message_id: str
    reaction: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., message_id: _Optional[str] = ..., reaction: _Optional[str] = ...) -> None: ...

class ReactResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class EditRequest(_message.Message):
    __slots__ = ("session_id", "message_id", "text")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    message_id: str
    text: str
    def __init__(self, session_id: _Optional[str] = ..., message_id: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class EditResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UnsendRequest(_message.Message):
    __slots__ = ("session_id", "message_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    message_id: str
    def __init__(self, session_id: _Optional[str] = ..., message_id: _Optional[str] = ...) -> None: ...

class UnsendResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetTypingRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "typing", "group", "thread_type")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    TYPING_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    THREAD_TYPE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    typing: bool
    group: bool
    thread_type: int
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., typing: _Optional[bool] = ..., group: _Optional[bool] = ..., thread_type: _Optional[int] = ...) -> None: ...

class SetTypingResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MarkReadRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "watermark")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    WATERMARK_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    watermark: _timestamp_pb2.Timestamp
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., watermark: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MarkReadResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListMessageRequestsRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ListMessageRequestsResponse(_message.Message):
    __slots__ = ("requests",)
    REQUESTS_FIELD_NUMBER: _ClassVar[int]
    requests: _containers.RepeatedCompositeFieldContainer[MessageRequest]
    def __init__(self, requests: _Optional[_Iterable[_Union[MessageRequest, _Mapping]]] = ...) -> None: ...

class ListThemesRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ListThemesResponse(_message.Message):
    __slots__ = ("themes",)
    THEMES_FIELD_NUMBER: _ClassVar[int]
    themes: _containers.RepeatedCompositeFieldContainer[Theme]
    def __init__(self, themes: _Optional[_Iterable[_Union[Theme, _Mapping]]] = ...) -> None: ...

class FindThemeRequest(_message.Message):
    __slots__ = ("session_id", "query")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    query: str
    def __init__(self, session_id: _Optional[str] = ..., query: _Optional[str] = ...) -> None: ...

class FindThemeResponse(_message.Message):
    __slots__ = ("theme",)
    THEME_FIELD_NUMBER: _ClassVar[int]
    theme: Theme
    def __init__(self, theme: _Optional[_Union[Theme, _Mapping]] = ...) -> None: ...

class SetThemeRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "theme_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    THEME_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    theme_id: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., theme_id: _Optional[str] = ...) -> None: ...

class SetThemeResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCurrentNoteRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class GetCurrentNoteResponse(_message.Message):
    __slots__ = ("note",)
    NOTE_FIELD_NUMBER: _ClassVar[int]
    note: Note
    def __init__(self, note: _Optional[_Union[Note, _Mapping]] = ...) -> None: ...

class CreateNoteRequest(_message.Message):
    __slots__ = ("session_id", "text", "privacy")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    text: str
    privacy: str
    def __init__(self, session_id: _Optional[str] = ..., text: _Optional[str] = ..., privacy: _Optional[str] = ...) -> None: ...

class CreateNoteResponse(_message.Message):
    __slots__ = ("note",)
    NOTE_FIELD_NUMBER: _ClassVar[int]
    note: Note
    def __init__(self, note: _Optional[_Union[Note, _Mapping]] = ...) -> None: ...

class DeleteNoteRequest(_message.Message):
    __slots__ = ("session_id", "note_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    note_id: str
    def __init__(self, session_id: _Optional[str] = ..., note_id: _Optional[str] = ...) -> None: ...

class DeleteNoteResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RecreateNoteRequest(_message.Message):
    __slots__ = ("session_id", "old_note_id", "text", "privacy")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    OLD_NOTE_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    old_note_id: str
    text: str
    privacy: str
    def __init__(self, session_id: _Optional[str] = ..., old_note_id: _Optional[str] = ..., text: _Optional[str] = ..., privacy: _Optional[str] = ...) -> None: ...

class RecreateNoteResponse(_message.Message):
    __slots__ = ("note",)
    NOTE_FIELD_NUMBER: _ClassVar[int]
    note: Note
    def __init__(self, note: _Optional[_Union[Note, _Mapping]] = ...) -> None: ...

class SetRestrictedRequest(_message.Message):
    __slots__ = ("session_id", "user_id", "restricted")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    RESTRICTED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    restricted: bool
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ..., restricted: _Optional[bool] = ...) -> None: ...

class SetRestrictedResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetMessageBlockedRequest(_message.Message):
    __slots__ = ("session_id", "user_id", "blocked")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    blocked: bool
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ..., blocked: _Optional[bool] = ...) -> None: ...

class SetMessageBlockedResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListThreadsRequest(_message.Message):
    __slots__ = ("session_id", "limit")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    limit: int
    def __init__(self, session_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListThreadsResponse(_message.Message):
    __slots__ = ("threads", "sync_sequence_id")
    THREADS_FIELD_NUMBER: _ClassVar[int]
    SYNC_SEQUENCE_ID_FIELD_NUMBER: _ClassVar[int]
    threads: _containers.RepeatedCompositeFieldContainer[Thread]
    sync_sequence_id: int
    def __init__(self, threads: _Optional[_Iterable[_Union[Thread, _Mapping]]] = ..., sync_sequence_id: _Optional[int] = ...) -> None: ...

class GetThreadRequest(_message.Message):
    __slots__ = ("session_id", "thread_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ...) -> None: ...

class GetThreadResponse(_message.Message):
    __slots__ = ("thread",)
    THREAD_FIELD_NUMBER: _ClassVar[int]
    thread: Thread
    def __init__(self, thread: _Optional[_Union[Thread, _Mapping]] = ...) -> None: ...

class CreatePollRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "question", "options")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    question: str
    options: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., question: _Optional[str] = ..., options: _Optional[_Iterable[str]] = ...) -> None: ...

class CreatePollResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class VotePollRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "poll_id", "option_ids")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    POLL_ID_FIELD_NUMBER: _ClassVar[int]
    OPTION_IDS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    poll_id: str
    option_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., poll_id: _Optional[str] = ..., option_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class VotePollResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPinnedMessagesRequest(_message.Message):
    __slots__ = ("session_id", "thread_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ...) -> None: ...

class ListPinnedMessagesResponse(_message.Message):
    __slots__ = ("messages",)
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    messages: _containers.RepeatedCompositeFieldContainer[PinnedMessage]
    def __init__(self, messages: _Optional[_Iterable[_Union[PinnedMessage, _Mapping]]] = ...) -> None: ...

class GetPollDetailsRequest(_message.Message):
    __slots__ = ("session_id", "poll_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    POLL_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    poll_id: str
    def __init__(self, session_id: _Optional[str] = ..., poll_id: _Optional[str] = ...) -> None: ...

class GetPollDetailsResponse(_message.Message):
    __slots__ = ("poll",)
    POLL_FIELD_NUMBER: _ClassVar[int]
    poll: PollDetails
    def __init__(self, poll: _Optional[_Union[PollDetails, _Mapping]] = ...) -> None: ...

class SearchMessagesRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "query", "cursor")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    query: str
    cursor: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., query: _Optional[str] = ..., cursor: _Optional[str] = ...) -> None: ...

class SearchMessagesResponse(_message.Message):
    __slots__ = ("results", "result_count", "has_next_page", "next_cursor")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    RESULT_COUNT_FIELD_NUMBER: _ClassVar[int]
    HAS_NEXT_PAGE_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[MessageSearchResult]
    result_count: int
    has_next_page: bool
    next_cursor: str
    def __init__(self, results: _Optional[_Iterable[_Union[MessageSearchResult, _Mapping]]] = ..., result_count: _Optional[int] = ..., has_next_page: _Optional[bool] = ..., next_cursor: _Optional[str] = ...) -> None: ...

class MuteThreadRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "duration", "indefinitely")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    INDEFINITELY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    duration: _duration_pb2.Duration
    indefinitely: bool
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., duration: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., indefinitely: _Optional[bool] = ...) -> None: ...

class MuteThreadResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MuteThreadCallsRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "duration", "indefinitely")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    INDEFINITELY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    duration: _duration_pb2.Duration
    indefinitely: bool
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., duration: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., indefinitely: _Optional[bool] = ...) -> None: ...

class MuteThreadCallsResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetApprovalModeRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "enabled")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    enabled: bool
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class SetApprovalModeResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetArchivedRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "archived")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    archived: bool
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., archived: _Optional[bool] = ...) -> None: ...

class SetArchivedResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetMessagePinnedRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "message_id", "pinned")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    PINNED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    message_id: str
    pinned: bool
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., message_id: _Optional[str] = ..., pinned: _Optional[bool] = ...) -> None: ...

class SetMessagePinnedResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetThreadPhotoMetadata(_message.Message):
    __slots__ = ("session_id", "thread_id", "name", "content_type", "size", "sha256")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    name: str
    content_type: str
    size: int
    sha256: bytes
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., name: _Optional[str] = ..., content_type: _Optional[str] = ..., size: _Optional[int] = ..., sha256: _Optional[bytes] = ...) -> None: ...

class SetThreadPhotoRequest(_message.Message):
    __slots__ = ("metadata", "chunk")
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CHUNK_FIELD_NUMBER: _ClassVar[int]
    metadata: SetThreadPhotoMetadata
    chunk: _common_pb2.MediaChunk
    def __init__(self, metadata: _Optional[_Union[SetThreadPhotoMetadata, _Mapping]] = ..., chunk: _Optional[_Union[_common_pb2.MediaChunk, _Mapping]] = ...) -> None: ...

class SetThreadPhotoResponse(_message.Message):
    __slots__ = ("sha256",)
    SHA256_FIELD_NUMBER: _ClassVar[int]
    sha256: bytes
    def __init__(self, sha256: _Optional[bytes] = ...) -> None: ...

class DeleteThreadRequest(_message.Message):
    __slots__ = ("session_id", "thread_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ...) -> None: ...

class DeleteThreadResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CreateDMRequest(_message.Message):
    __slots__ = ("session_id", "user_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class CreateDMResponse(_message.Message):
    __slots__ = ("thread_id",)
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    def __init__(self, thread_id: _Optional[str] = ...) -> None: ...

class SearchUsersRequest(_message.Message):
    __slots__ = ("session_id", "query")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    query: str
    def __init__(self, session_id: _Optional[str] = ..., query: _Optional[str] = ...) -> None: ...

class SearchUsersResponse(_message.Message):
    __slots__ = ("users",)
    USERS_FIELD_NUMBER: _ClassVar[int]
    users: _containers.RepeatedCompositeFieldContainer[User]
    def __init__(self, users: _Optional[_Iterable[_Union[User, _Mapping]]] = ...) -> None: ...

class GetContactRequest(_message.Message):
    __slots__ = ("session_id", "user_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class GetContactResponse(_message.Message):
    __slots__ = ("user",)
    USER_FIELD_NUMBER: _ClassVar[int]
    user: User
    def __init__(self, user: _Optional[_Union[User, _Mapping]] = ...) -> None: ...

class SetAdminRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "user_id", "admin")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    ADMIN_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    user_id: str
    admin: bool
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., user_id: _Optional[str] = ..., admin: _Optional[bool] = ...) -> None: ...

class SetAdminResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetThreadNameRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "name")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    name: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class SetThreadNameResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetThreadEmojiRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "emoji")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    EMOJI_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    emoji: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., emoji: _Optional[str] = ...) -> None: ...

class SetThreadEmojiResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetNicknameRequest(_message.Message):
    __slots__ = ("session_id", "thread_id", "user_id", "nickname")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    thread_id: str
    user_id: str
    nickname: str
    def __init__(self, session_id: _Optional[str] = ..., thread_id: _Optional[str] = ..., user_id: _Optional[str] = ..., nickname: _Optional[str] = ...) -> None: ...

class SetNicknameResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
