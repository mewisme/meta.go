import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FacebookPostOwnership(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FACEBOOK_POST_OWNERSHIP_UNSPECIFIED: _ClassVar[FacebookPostOwnership]
    FACEBOOK_POST_OWNERSHIP_OWNED: _ClassVar[FacebookPostOwnership]
    FACEBOOK_POST_OWNERSHIP_SHARED: _ClassVar[FacebookPostOwnership]
FACEBOOK_POST_OWNERSHIP_UNSPECIFIED: FacebookPostOwnership
FACEBOOK_POST_OWNERSHIP_OWNED: FacebookPostOwnership
FACEBOOK_POST_OWNERSHIP_SHARED: FacebookPostOwnership

class FacebookUser(_message.Message):
    __slots__ = ("id", "name", "first_name", "username", "profile_url", "avatar_url", "gender", "alternate_name", "non_friend")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FIRST_NAME_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    PROFILE_URL_FIELD_NUMBER: _ClassVar[int]
    AVATAR_URL_FIELD_NUMBER: _ClassVar[int]
    GENDER_FIELD_NUMBER: _ClassVar[int]
    ALTERNATE_NAME_FIELD_NUMBER: _ClassVar[int]
    NON_FRIEND_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    first_name: str
    username: str
    profile_url: str
    avatar_url: str
    gender: str
    alternate_name: str
    non_friend: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., first_name: _Optional[str] = ..., username: _Optional[str] = ..., profile_url: _Optional[str] = ..., avatar_url: _Optional[str] = ..., gender: _Optional[str] = ..., alternate_name: _Optional[str] = ..., non_friend: _Optional[bool] = ...) -> None: ...

class FacebookSearchResult(_message.Message):
    __slots__ = ("id", "name", "url")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    url: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., url: _Optional[str] = ...) -> None: ...

class FacebookNotification(_message.Message):
    __slots__ = ("id", "text", "url", "timestamp")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    id: str
    text: str
    url: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., text: _Optional[str] = ..., url: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class FacebookPost(_message.Message):
    __slots__ = ("id", "url")
    ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    url: str
    def __init__(self, id: _Optional[str] = ..., url: _Optional[str] = ...) -> None: ...

class MarketplaceLocation(_message.Message):
    __slots__ = ("latitude", "longitude", "name")
    LATITUDE_FIELD_NUMBER: _ClassVar[int]
    LONGITUDE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    latitude: float
    longitude: float
    name: str
    def __init__(self, latitude: _Optional[float] = ..., longitude: _Optional[float] = ..., name: _Optional[str] = ...) -> None: ...

class MarketplaceListingInput(_message.Message):
    __slots__ = ("title", "brand", "price", "currency", "description", "hashtags", "category", "photo_ids", "location")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    HASHTAGS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    PHOTO_IDS_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    title: str
    brand: str
    price: str
    currency: str
    description: str
    hashtags: _containers.RepeatedScalarFieldContainer[str]
    category: str
    photo_ids: _containers.RepeatedScalarFieldContainer[str]
    location: MarketplaceLocation
    def __init__(self, title: _Optional[str] = ..., brand: _Optional[str] = ..., price: _Optional[str] = ..., currency: _Optional[str] = ..., description: _Optional[str] = ..., hashtags: _Optional[_Iterable[str]] = ..., category: _Optional[str] = ..., photo_ids: _Optional[_Iterable[str]] = ..., location: _Optional[_Union[MarketplaceLocation, _Mapping]] = ...) -> None: ...

class MarketplaceListing(_message.Message):
    __slots__ = ("id", "title", "description", "price", "currency", "seller", "location", "url", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRICE_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    SELLER_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    price: str
    currency: str
    seller: FacebookUser
    location: MarketplaceLocation
    url: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., price: _Optional[str] = ..., currency: _Optional[str] = ..., seller: _Optional[_Union[FacebookUser, _Mapping]] = ..., location: _Optional[_Union[MarketplaceLocation, _Mapping]] = ..., url: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class FacebookServiceGetUserRequest(_message.Message):
    __slots__ = ("session_id", "user_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class FacebookServiceGetUserResponse(_message.Message):
    __slots__ = ("user",)
    USER_FIELD_NUMBER: _ClassVar[int]
    user: FacebookUser
    def __init__(self, user: _Optional[_Union[FacebookUser, _Mapping]] = ...) -> None: ...

class FacebookServiceSearchRequest(_message.Message):
    __slots__ = ("session_id", "query", "limit")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    query: str
    limit: int
    def __init__(self, session_id: _Optional[str] = ..., query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class FacebookServiceSearchResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[FacebookSearchResult]
    def __init__(self, results: _Optional[_Iterable[_Union[FacebookSearchResult, _Mapping]]] = ...) -> None: ...

class FacebookServiceListNotificationsRequest(_message.Message):
    __slots__ = ("session_id", "limit")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    limit: int
    def __init__(self, session_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class FacebookServiceListNotificationsResponse(_message.Message):
    __slots__ = ("notifications",)
    NOTIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    notifications: _containers.RepeatedCompositeFieldContainer[FacebookNotification]
    def __init__(self, notifications: _Optional[_Iterable[_Union[FacebookNotification, _Mapping]]] = ...) -> None: ...

class FacebookServiceSetBioRequest(_message.Message):
    __slots__ = ("session_id", "bio", "publish_feed_story")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    BIO_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_FEED_STORY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    bio: str
    publish_feed_story: bool
    def __init__(self, session_id: _Optional[str] = ..., bio: _Optional[str] = ..., publish_feed_story: _Optional[bool] = ...) -> None: ...

class FacebookServiceSetBioResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FacebookServiceCreateAdditionalProfileRequest(_message.Message):
    __slots__ = ("session_id", "name", "username")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    name: str
    username: str
    def __init__(self, session_id: _Optional[str] = ..., name: _Optional[str] = ..., username: _Optional[str] = ...) -> None: ...

class FacebookServiceCreateAdditionalProfileResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FacebookServiceUnfriendRequest(_message.Message):
    __slots__ = ("session_id", "user_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class FacebookServiceUnfriendResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FacebookServiceSetBlockedRequest(_message.Message):
    __slots__ = ("session_id", "user_id", "blocked")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    user_id: str
    blocked: bool
    def __init__(self, session_id: _Optional[str] = ..., user_id: _Optional[str] = ..., blocked: _Optional[bool] = ...) -> None: ...

class FacebookServiceSetBlockedResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FacebookServiceCreatePostRequest(_message.Message):
    __slots__ = ("session_id", "text")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    text: str
    def __init__(self, session_id: _Optional[str] = ..., text: _Optional[str] = ...) -> None: ...

class FacebookServiceCreatePostResponse(_message.Message):
    __slots__ = ("post",)
    POST_FIELD_NUMBER: _ClassVar[int]
    post: FacebookPost
    def __init__(self, post: _Optional[_Union[FacebookPost, _Mapping]] = ...) -> None: ...

class FacebookServiceArchivePostRequest(_message.Message):
    __slots__ = ("session_id", "post_id", "ownership")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    POST_ID_FIELD_NUMBER: _ClassVar[int]
    OWNERSHIP_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    post_id: str
    ownership: FacebookPostOwnership
    def __init__(self, session_id: _Optional[str] = ..., post_id: _Optional[str] = ..., ownership: _Optional[_Union[FacebookPostOwnership, str]] = ...) -> None: ...

class FacebookServiceArchivePostResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FacebookServiceDeletePostRequest(_message.Message):
    __slots__ = ("session_id", "post_id", "ownership")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    POST_ID_FIELD_NUMBER: _ClassVar[int]
    OWNERSHIP_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    post_id: str
    ownership: FacebookPostOwnership
    def __init__(self, session_id: _Optional[str] = ..., post_id: _Optional[str] = ..., ownership: _Optional[_Union[FacebookPostOwnership, str]] = ...) -> None: ...

class FacebookServiceDeletePostResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FacebookServiceCreateMarketplaceListingRequest(_message.Message):
    __slots__ = ("session_id", "listing")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LISTING_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    listing: MarketplaceListingInput
    def __init__(self, session_id: _Optional[str] = ..., listing: _Optional[_Union[MarketplaceListingInput, _Mapping]] = ...) -> None: ...

class FacebookServiceCreateMarketplaceListingResponse(_message.Message):
    __slots__ = ("listing",)
    LISTING_FIELD_NUMBER: _ClassVar[int]
    listing: MarketplaceListing
    def __init__(self, listing: _Optional[_Union[MarketplaceListing, _Mapping]] = ...) -> None: ...

class FacebookServiceGetMarketplaceListingRequest(_message.Message):
    __slots__ = ("session_id", "listing_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LISTING_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    listing_id: str
    def __init__(self, session_id: _Optional[str] = ..., listing_id: _Optional[str] = ...) -> None: ...

class FacebookServiceGetMarketplaceListingResponse(_message.Message):
    __slots__ = ("listing",)
    LISTING_FIELD_NUMBER: _ClassVar[int]
    listing: MarketplaceListing
    def __init__(self, listing: _Optional[_Union[MarketplaceListing, _Mapping]] = ...) -> None: ...

class FacebookServiceSetProfessionalModeRequest(_message.Message):
    __slots__ = ("session_id", "enabled")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    enabled: bool
    def __init__(self, session_id: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class FacebookServiceSetProfessionalModeResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
