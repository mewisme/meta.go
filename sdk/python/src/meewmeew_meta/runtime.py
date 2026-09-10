from __future__ import annotations

import asyncio
import contextlib
import json
import os
from dataclasses import dataclass
from pathlib import Path

from .distribution import RuntimeDistributionOptions, resolve_runtime_path
from .errors import ProtocolMismatchError, RuntimeLaunchError

PROTOCOL_MAJOR = 1
_MAX_BOOTSTRAP_BYTES = 64 << 10
_MAX_STDERR_BYTES = 16 << 10


@dataclass(frozen=True, slots=True)
class RuntimeBootstrap:
    endpoint: str
    token: str
    protocol: int


@dataclass(frozen=True, slots=True)
class ManagedRuntimeOptions(RuntimeDistributionOptions):
    listen: str = "127.0.0.1:0"
    token: str | None = None
    bootstrap_timeout: float = 10.0
    stop_timeout: float = 5.0


class ManagedRuntime:
    def __init__(
        self,
        process: asyncio.subprocess.Process,
        bootstrap: RuntimeBootstrap,
        stop_timeout: float,
        stderr_task: asyncio.Task[bytes],
    ) -> None:
        self._process = process
        self._stderr_task = stderr_task
        self.endpoint = bootstrap.endpoint
        self.token = bootstrap.token
        self.protocol = bootstrap.protocol
        self.stop_timeout = stop_timeout
        self._stopped = False

    @property
    def pid(self) -> int | None:
        return self._process.pid

    @classmethod
    async def start(cls, options: ManagedRuntimeOptions | None = None) -> ManagedRuntime:
        options = options or ManagedRuntimeOptions()
        env = dict(os.environ)
        if options.env:
            env.update(options.env)
        runtime_path = await asyncio.to_thread(resolve_runtime_path, options, env)
        if options.token:
            env["META_RUNTIME_TOKEN"] = options.token
        try:
            process = await asyncio.create_subprocess_exec(
                runtime_path,
                "--listen",
                options.listen,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                env=env,
            )
        except OSError as error:
            raise RuntimeLaunchError(f"failed to start {Path(runtime_path).name}") from error
        stderr_task = asyncio.create_task(_capture_stderr(process.stderr))
        try:
            bootstrap = await _read_bootstrap(process, options.bootstrap_timeout, stderr_task)
            if bootstrap.protocol != PROTOCOL_MAJOR:
                process.terminate()
                raise ProtocolMismatchError(PROTOCOL_MAJOR, bootstrap.protocol)
            return cls(process, bootstrap, options.stop_timeout, stderr_task)
        except BaseException:
            if process.returncode is None:
                process.terminate()
            with contextlib.suppress(Exception):
                await asyncio.wait_for(process.wait(), 1.0)
            stderr_task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await stderr_task
            raise

    async def stop(self) -> None:
        if self._stopped:
            return
        self._stopped = True
        if self._process.returncode is None:
            self._process.terminate()
            try:
                await asyncio.wait_for(self._process.wait(), self.stop_timeout)
            except TimeoutError:
                self._process.kill()
                await self._process.wait()
        self._stderr_task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await self._stderr_task


async def _capture_stderr(stream: asyncio.StreamReader | None) -> bytes:
    if stream is None:
        return b""
    data = bytearray()
    while True:
        chunk = await stream.read(4096)
        if not chunk:
            break
        data.extend(chunk)
        if len(data) > _MAX_STDERR_BYTES:
            del data[:-_MAX_STDERR_BYTES]
    return bytes(data)


async def _read_bootstrap(
    process: asyncio.subprocess.Process, timeout: float, stderr_task: asyncio.Task[bytes]
) -> RuntimeBootstrap:
    if process.stdout is None:
        raise RuntimeLaunchError("runtime stdout is unavailable")
    line_task = asyncio.create_task(process.stdout.readline())
    exit_task = asyncio.create_task(process.wait())
    try:
        done, _ = await asyncio.wait({line_task, exit_task}, timeout=timeout, return_when=asyncio.FIRST_COMPLETED)
        if line_task in done:
            try:
                line = line_task.result()
            except ValueError as error:
                raise RuntimeLaunchError("runtime bootstrap line is too large") from error
            if not line:
                details = await _stderr_text(stderr_task)
                raise RuntimeLaunchError(f"runtime exited before bootstrap{details}")
            if len(line) > _MAX_BOOTSTRAP_BYTES:
                raise RuntimeLaunchError("runtime bootstrap line is too large")
            try:
                value = json.loads(line)
            except (UnicodeDecodeError, json.JSONDecodeError) as error:
                raise RuntimeLaunchError("runtime emitted invalid bootstrap JSON") from error
            endpoint = value.get("endpoint")
            token = value.get("token")
            protocol = value.get("protocol")
            if (
                not isinstance(endpoint, str)
                or not endpoint
                or not isinstance(token, str)
                or not token
                or not isinstance(protocol, int)
            ):
                raise RuntimeLaunchError("runtime emitted invalid bootstrap fields")
            return RuntimeBootstrap(endpoint=endpoint, token=token, protocol=protocol)
        if exit_task in done:
            details = await _stderr_text(stderr_task)
            raise RuntimeLaunchError(f"runtime exited before bootstrap ({process.returncode}){details}")
        details = await _stderr_text(stderr_task) if stderr_task.done() else ""
        raise RuntimeLaunchError(f"runtime bootstrap timed out after {timeout:g}s{details}")
    finally:
        for task in (line_task, exit_task):
            if not task.done():
                task.cancel()


async def _stderr_text(task: asyncio.Task[bytes]) -> str:
    if not task.done():
        return ""
    data = await task
    text = data.decode("utf-8", errors="replace").strip()
    return f": {text}" if text else ""
