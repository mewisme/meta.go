from __future__ import annotations

import hashlib
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pytest

from meewmeew_meta import RuntimeDistributionOptions, RuntimeLaunchError, resolve_runtime_path
from meewmeew_meta.distribution import _runtime_target


def test_resolve_runtime_path_downloads_verifies_and_reuses_cache(tmp_path: Path) -> None:
    version = "test"
    target = _runtime_target()
    asset = f"meta-runtime_{version}_{target.os}_{target.arch}{'.exe' if target.os == 'windows' else ''}"
    binary = b"runtime-test"
    checksum = hashlib.sha256(binary).hexdigest()
    requests = 0

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            nonlocal requests
            requests += 1
            if self.path.endswith("_checksums.txt"):
                body = f"{checksum}  {asset}\n".encode()
            elif self.path.endswith(f"/{asset}"):
                body = binary
            else:
                self.send_error(404)
                return
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, format: str, *args: object) -> None:
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    options = RuntimeDistributionOptions(
        runtime_version=version,
        runtime_cache_dir=tmp_path,
        release_base_url=f"http://127.0.0.1:{server.server_port}",
        env={"PATH": ""},
    )
    try:
        first = Path(resolve_runtime_path(options))
        assert first.read_bytes() == binary
        assert requests == 2
        second = Path(resolve_runtime_path(options))
        assert second == first
        assert requests == 2
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


def test_resolve_runtime_path_rejects_mismatched_checksum(tmp_path: Path) -> None:
    version = "bad"
    target = _runtime_target()
    asset = f"meta-runtime_{version}_{target.os}_{target.arch}{'.exe' if target.os == 'windows' else ''}"

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            body = f"{'0' * 64}  {asset}\n".encode() if self.path.endswith("_checksums.txt") else b"tampered"
            self.send_response(200)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, format: str, *args: object) -> None:
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        with pytest.raises(RuntimeLaunchError, match="checksum mismatch"):
            resolve_runtime_path(
                RuntimeDistributionOptions(
                    runtime_version=version,
                    runtime_cache_dir=tmp_path,
                    release_base_url=f"http://127.0.0.1:{server.server_port}",
                    env={"PATH": ""},
                )
            )
    finally:
        server.shutdown()
        server.server_close()
        thread.join()
