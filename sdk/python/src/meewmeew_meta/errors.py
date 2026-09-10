from __future__ import annotations

import grpc


class MetaError(Exception):
    pass


class MetaRpcError(MetaError):
    def __init__(self, code: grpc.StatusCode, details: str, *, cause: BaseException | None = None) -> None:
        super().__init__(f"{code.name}: {details}")
        self.code = code
        self.details = details
        self.__cause__ = cause


class UnsupportedCapabilityError(MetaError):
    def __init__(self, capability: str) -> None:
        super().__init__(f"runtime does not support capability {capability}")
        self.capability = capability


class RuntimeLaunchError(MetaError):
    pass


class ProtocolMismatchError(MetaError):
    def __init__(self, expected: int, actual: int) -> None:
        super().__init__(f"runtime protocol major {actual} is incompatible with SDK protocol major {expected}")
        self.expected = expected
        self.actual = actual


class EventBufferOverflowError(MetaError):
    def __init__(self, limit: int) -> None:
        super().__init__(f"event iterator exceeded its local buffer limit of {limit}")
        self.limit = limit


def map_rpc_error(error: BaseException) -> BaseException:
    if isinstance(error, grpc.aio.AioRpcError):
        return MetaRpcError(error.code(), error.details() or "", cause=error)
    return error
