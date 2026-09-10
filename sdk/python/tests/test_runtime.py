from __future__ import annotations

import os
import subprocess
from pathlib import Path

import grpc
import pytest

from meewmeew_meta import ManagedRuntime, ManagedRuntimeOptions, MetaClient, MetaRpcError, UnsupportedCapabilityError


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
        with pytest.raises(UnsupportedCapabilityError) as error:
            client.require_capability("future.echo")
        assert error.value.capability == "future.echo"

        created = await client.create_session({"c_user": "42", "xs": "test"}, event_buffer=8, timeout_ms=5000)
        assert created.session_id
        health = await client.get_session_health(created.session_id)
        assert health.HasField("health")
        assert health.health.reconnect_count == 0
        await client.close_session(created.session_id)
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
async def test_managed_runtime_stop_is_idempotent(runtime_path: Path) -> None:
    runtime = await ManagedRuntime.start(ManagedRuntimeOptions(runtime_path=runtime_path))
    assert runtime.pid
    await runtime.stop()
    await runtime.stop()
