from meta.v1 import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetInfoRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetInfoResponse(_message.Message):
    __slots__ = ("protocol", "build", "capabilities")
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    BUILD_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    protocol: _common_pb2.ProtocolVersion
    build: _common_pb2.BuildInfo
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, protocol: _Optional[_Union[_common_pb2.ProtocolVersion, _Mapping]] = ..., build: _Optional[_Union[_common_pb2.BuildInfo, _Mapping]] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...
