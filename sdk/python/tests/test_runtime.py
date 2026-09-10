from __future__ import annotations

import os
import subprocess
from pathlib import Path

import grpc
import pytest

from meta.v1 import session_pb2
from mewisme_meta import (
    AppStateAuth,
    AppStateCookie,
    CookieAuth,
    CredentialAuth,
    Credentials,
    ManagedRuntime,
    ManagedRuntimeOptions,
    MetaClient,
    MetaRpcError,
    UnsupportedCapabilityError,
)


@pytest.fixture(scope="session")
def runtime_path(tmp_path_factory: pytest.TempPathFactory) -> Path:
    repo_root = Path(__file__).resolve().parents[3]
    path = tmp_path_factory.mktemp("runtime") / ("meta-runtime.exe" if os.name == "nt" else "meta-runtime")
    subprocess.run(
        ["go", "build", "-o", str(path), "./cmd/meta-runtime"],
        cwd=repo_root,
        env={**os.environ, "GOWORK": "off"},
        check=True,
    )
    return path


@pytest.mark.asyncio
async def test_managed_runtime_exposes_capabilities_and_session_lifecycle(runtime_path: Path) -> None:
    client = await MetaClient.managed(runtime_path)
    try:
        assert client.info is not None
        assert client.info.protocol.major == 1
        assert client.has_capability("session.lifecycle")
        assert client.has_capability("session.auth.app_state")
        assert client.has_capability("session.auth.credentials")
        assert client.has_capability("session.auth.refresh")
        with pytest.raises(UnsupportedCapabilityError) as error:
            client.require_capability("future.echo")
        assert error.value.capability == "future.echo"

        created = await client.create_session(
            CookieAuth({"c_user": "42", "xs": "test"}), event_buffer=8, timeout_ms=5000
        )
        assert created.session_id
        app_state = await client.create_session(
            AppStateAuth((AppStateCookie("c_user", "42"), AppStateCookie("xs", "test", http_only=True)))
        )
        assert app_state.session_id
        credentials = await client.create_session(CredentialAuth(Credentials("user", "secret", otp="123456")))
        assert credentials.session_id
        with pytest.raises(ValueError, match="exactly one of totp or otp"):
            Credentials("user", "secret")
        health = await client.get_session_health(created.session_id)
        assert health.HasField("health")
        assert health.health.reconnect_count == 0
        await client.close_session(created.session_id)
        await client.close_session(app_state.session_id)
        await client.close_session(credentials.session_id)
    finally:
        await client.close()


@pytest.mark.asyncio
async def test_external_endpoint_authenticates_against_existing_runtime(runtime_path: Path) -> None:
    runtime = await ManagedRuntime.start(ManagedRuntimeOptions(runtime_path=runtime_path))
    try:
        client = await MetaClient.external(runtime.endpoint, runtime.token)
        try:
            assert client.info is not None
            assert client.info.protocol.major == 1
        finally:
            await client.close()

        with pytest.raises(MetaRpcError) as caught:
            await MetaClient.external(runtime.endpoint, "wrong-token")
        assert caught.value.code is grpc.StatusCode.UNAUTHENTICATED
    finally:
        await runtime.stop()


@pytest.mark.asyncio
async def test_auth_wrappers_preserve_event_state_and_enforce_runtime_capabilities(runtime_path: Path) -> None:
    client = await MetaClient.managed(runtime_path)
    original_capabilities = client._capabilities

    class FakeSessions:
        def __init__(self) -> None:
            self.refresh_session_id = ""
            self.snapshot_session_id = ""

        async def RefreshAuth(self, request: session_pb2.RefreshAuthRequest) -> session_pb2.RefreshAuthResponse:
            self.refresh_session_id = request.session_id
            return session_pb2.RefreshAuthResponse(snapshot=self._snapshot())

        async def GetAuthSnapshot(
            self, request: session_pb2.GetAuthSnapshotRequest
        ) -> session_pb2.GetAuthSnapshotResponse:
            self.snapshot_session_id = request.session_id
            return session_pb2.GetAuthSnapshotResponse(snapshot=self._snapshot())

        @staticmethod
        def _snapshot() -> session_pb2.AuthSnapshot:
            return session_pb2.AuthSnapshot(
                cookies=session_pb2.CookieMap(values={"c_user": "42", "xs": "fresh"}),
                app_state=session_pb2.AppState(cookies=[session_pb2.AppStateCookie(key="c_user", value="42")]),
                session=session_pb2.FacebookSession(account_id="42", session_id="facebook-session", client_revision=7),
            )

    class Marker:
        closed = False

        async def aclose(self) -> None:
            self.closed = True

    try:
        client._capabilities = frozenset({"session.lifecycle"})
        with pytest.raises(UnsupportedCapabilityError):
            await client.create_session(AppStateAuth((AppStateCookie("c_user", "42"), AppStateCookie("xs", "x"))))
        with pytest.raises(UnsupportedCapabilityError):
            await client.create_session(CredentialAuth(Credentials("user", "x", otp="123456")))
        with pytest.raises(UnsupportedCapabilityError):
            await client.refresh_auth("session-1")
        with pytest.raises(UnsupportedCapabilityError):
            await client.auth_snapshot("session-1")
        client._capabilities = original_capabilities

        sessions = FakeSessions()
        client.sessions = sessions
        marker = Marker()
        client._event_streams.add(marker)
        refreshed = await client.refresh_auth("session-1")
        current = await client.auth_snapshot("session-1")
        assert sessions.refresh_session_id == "session-1"
        assert sessions.snapshot_session_id == "session-1"
        assert refreshed.cookies["xs"] == "fresh"
        assert current.session is not None and current.session.session_id == "facebook-session"
        assert not marker.closed
        assert marker in client._event_streams
        client._event_streams.remove(marker)
    finally:
        client._capabilities = original_capabilities
        await client.close()


@pytest.mark.asyncio
async def test_managed_runtime_stop_is_idempotent(runtime_path: Path) -> None:
    runtime = await ManagedRuntime.start(ManagedRuntimeOptions(runtime_path=runtime_path))
    assert runtime.pid
    await runtime.stop()
    await runtime.stop()
