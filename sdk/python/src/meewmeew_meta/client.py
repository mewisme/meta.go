from __future__ import annotations

from collections.abc import AsyncIterator, Mapping
from pathlib import Path
from typing import Any, cast

import grpc
from google.protobuf import duration_pb2

from meta.v1 import (
    e2ee_pb2_grpc,
    facebook_pb2_grpc,
    messenger_pb2_grpc,
    runtime_pb2,
    runtime_pb2_grpc,
    session_pb2,
    session_pb2_grpc,
)

from .errors import ProtocolMismatchError, UnsupportedCapabilityError, map_rpc_error
from .events import EventStream
from .runtime import PROTOCOL_MAJOR, ManagedRuntime, ManagedRuntimeOptions


class _MetaInterceptor(
    grpc.aio.UnaryUnaryClientInterceptor,
    grpc.aio.UnaryStreamClientInterceptor,
    grpc.aio.StreamUnaryClientInterceptor,
    grpc.aio.StreamStreamClientInterceptor,
):
    def __init__(self, token: str) -> None:
        self._authorization = ("authorization", f"Bearer {token}")

    def _details(self, details: grpc.aio.ClientCallDetails) -> grpc.aio.ClientCallDetails:
        metadata = grpc.aio.Metadata(*(details.metadata or ()))
        metadata.add(*self._authorization)
        return grpc.aio.ClientCallDetails(
            details.method, details.timeout, metadata, details.credentials, details.wait_for_ready
        )

    async def intercept_unary_unary(self, continuation: Any, details: grpc.aio.ClientCallDetails, request: Any) -> Any:
        try:
            call = await continuation(self._details(details), request)
            return await call
        except grpc.aio.AioRpcError as error:
            raise map_rpc_error(error) from error

    async def intercept_unary_stream(
        self, continuation: Any, details: grpc.aio.ClientCallDetails, request: Any
    ) -> AsyncIterator[Any]:
        call = await continuation(self._details(details), request)

        async def iterate() -> AsyncIterator[Any]:
            try:
                async for response in call:
                    yield response
            except grpc.aio.AioRpcError as error:
                raise map_rpc_error(error) from error

        return iterate()

    async def intercept_stream_unary(self, continuation: Any, details: grpc.aio.ClientCallDetails, request: Any) -> Any:
        try:
            call = await continuation(self._details(details), request)
            return await call
        except grpc.aio.AioRpcError as error:
            raise map_rpc_error(error) from error

    async def intercept_stream_stream(
        self, continuation: Any, details: grpc.aio.ClientCallDetails, request: Any
    ) -> AsyncIterator[Any]:
        call = await continuation(self._details(details), request)

        async def iterate() -> AsyncIterator[Any]:
            try:
                async for response in call:
                    yield response
            except grpc.aio.AioRpcError as error:
                raise map_rpc_error(error) from error

        return iterate()


class MetaClient:
    def __init__(self, endpoint: str, token: str, managed: ManagedRuntime | None = None) -> None:
        self.endpoint = endpoint
        self._managed = managed
        self._channel = grpc.aio.insecure_channel(endpoint, interceptors=[_MetaInterceptor(token)])
        self.runtime: Any = cast(Any, runtime_pb2_grpc.RuntimeServiceStub)(self._channel)
        self.sessions: Any = cast(Any, session_pb2_grpc.SessionServiceStub)(self._channel)
        self.messenger: Any = cast(Any, messenger_pb2_grpc.MessengerServiceStub)(self._channel)
        self.e2ee: Any = cast(Any, e2ee_pb2_grpc.E2EEServiceStub)(self._channel)
        self.facebook: Any = cast(Any, facebook_pb2_grpc.FacebookServiceStub)(self._channel)
        self._info: runtime_pb2.GetInfoResponse | None = None
        self._capabilities: frozenset[str] = frozenset()
        self._event_streams: set[EventStream] = set()
        self._closed = False

    @classmethod
    async def managed(
        cls,
        runtime_path: str | Path | None = None,
        *,
        runtime_version: str | None = None,
        runtime_cache_dir: str | Path | None = None,
        release_base_url: str | None = None,
        download_runtime: bool = True,
        env: Mapping[str, str] | None = None,
        listen: str = "127.0.0.1:0",
        token: str | None = None,
        bootstrap_timeout: float = 10.0,
        stop_timeout: float = 5.0,
    ) -> MetaClient:
        managed = await ManagedRuntime.start(
            ManagedRuntimeOptions(
                runtime_path=runtime_path,
                runtime_version=runtime_version,
                runtime_cache_dir=runtime_cache_dir,
                release_base_url=release_base_url,
                download_runtime=download_runtime,
                env=env,
                listen=listen,
                token=token,
                bootstrap_timeout=bootstrap_timeout,
                stop_timeout=stop_timeout,
            )
        )
        client = cls(managed.endpoint, managed.token, managed)
        return await client._handshake()

    @classmethod
    async def external(cls, endpoint: str, token: str) -> MetaClient:
        return await cls(endpoint, token)._handshake()

    async def _handshake(self) -> MetaClient:
        try:
            info = await self.runtime.GetInfo(runtime_pb2.GetInfoRequest())
            major = info.protocol.major if info.HasField("protocol") else 0
            if major != PROTOCOL_MAJOR:
                raise ProtocolMismatchError(PROTOCOL_MAJOR, major)
            self._info = info
            self._capabilities = frozenset(info.capabilities)
            return self
        except BaseException:
            await self.close()
            raise

    @property
    def info(self) -> runtime_pb2.GetInfoResponse | None:
        return self._info

    def has_capability(self, capability: str) -> bool:
        return capability in self._capabilities

    def require_capability(self, capability: str) -> None:
        if capability not in self._capabilities:
            raise UnsupportedCapabilityError(capability)

    async def create_session(
        self,
        cookies: dict[str, str],
        *,
        e2ee: bool = False,
        event_buffer: int = 0,
        timeout_ms: int | None = None,
    ) -> session_pb2.CreateSessionResponse:
        timeout = None
        if timeout_ms is not None:
            if timeout_ms <= 0:
                raise ValueError("timeout_ms must be positive")
            timeout = duration_pb2.Duration()
            timeout.FromMilliseconds(timeout_ms)
        return cast(
            session_pb2.CreateSessionResponse,
            await self.sessions.CreateSession(
                session_pb2.CreateSessionRequest(cookies=cookies, e2ee=e2ee, event_buffer=event_buffer, timeout=timeout)
            ),
        )

    async def connect_session(self, session_id: str) -> session_pb2.ConnectResponse:
        return cast(
            session_pb2.ConnectResponse, await self.sessions.Connect(session_pb2.ConnectRequest(session_id=session_id))
        )

    async def get_session_health(self, session_id: str) -> session_pb2.GetHealthResponse:
        return cast(
            session_pb2.GetHealthResponse,
            await self.sessions.GetHealth(session_pb2.GetHealthRequest(session_id=session_id)),
        )

    async def close_session(self, session_id: str) -> None:
        await self.sessions.CloseSession(session_pb2.CloseSessionRequest(session_id=session_id))

    def events(self, session_id: str, *, buffer: int = 0, iterator_buffer: int = 256) -> EventStream:
        call = self.sessions.SubscribeEvents(session_pb2.SubscribeEventsRequest(session_id=session_id, buffer=buffer))
        stream = EventStream(
            call, cancel=call.cancel, iterator_buffer=iterator_buffer, on_close=self._event_streams.discard
        )
        self._event_streams.add(stream)
        return stream

    async def close(self) -> None:
        if self._closed:
            return
        self._closed = True
        for stream in tuple(self._event_streams):
            await stream.aclose()
        self._event_streams.clear()
        await self._channel.close()
        if self._managed:
            await self._managed.stop()

    async def __aenter__(self) -> MetaClient:
        return self

    async def __aexit__(self, exc_type: object, exc: object, traceback: object) -> None:
        await self.close()
