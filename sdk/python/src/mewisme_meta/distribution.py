from __future__ import annotations

import hashlib
import os
import platform
import re
import shutil
import sys
import urllib.error
import urllib.request
from collections.abc import Mapping
from dataclasses import dataclass
from importlib.metadata import PackageNotFoundError
from importlib.metadata import version as package_version
from pathlib import Path

from .errors import RuntimeLaunchError

_DEFAULT_RELEASE_BASE_URL = "https://github.com/mewisme/meta.go/releases/download"


@dataclass(frozen=True, slots=True)
class RuntimeDistributionOptions:
    runtime_path: str | Path | None = None
    runtime_version: str | None = None
    runtime_cache_dir: str | Path | None = None
    release_base_url: str | None = None
    download_runtime: bool = True
    env: Mapping[str, str] | None = None


@dataclass(frozen=True, slots=True)
class RuntimeTarget:
    os: str
    arch: str
    filename: str


def resolve_runtime_path(
    options: RuntimeDistributionOptions | None = None, env: Mapping[str, str] | None = None
) -> str:
    options = options or RuntimeDistributionOptions()
    resolved_env = dict(os.environ if env is None else env)
    if options.env:
        resolved_env.update(options.env)
    if options.runtime_path is not None:
        return str(options.runtime_path)
    if resolved_env.get("META_RUNTIME_PATH"):
        return resolved_env["META_RUNTIME_PATH"]
    target = _runtime_target()
    installed = shutil.which(target.filename, path=resolved_env.get("PATH"))
    if installed:
        return installed
    if not options.download_runtime:
        raise RuntimeLaunchError(f"{target.filename} was not found on PATH and runtime downloading is disabled")
    runtime_version = _normalize_version(
        options.runtime_version or resolved_env.get("META_RUNTIME_VERSION") or _package_version()
    )
    cache_root = Path(
        options.runtime_cache_dir or resolved_env.get("META_RUNTIME_CACHE_DIR") or _default_cache_dir(resolved_env)
    )
    cache_dir = cache_root / runtime_version / f"{target.os}-{target.arch}"
    runtime_path = cache_dir / target.filename
    checksum_path = runtime_path.with_name(f"{runtime_path.name}.sha256")
    if _verify_cached(runtime_path, checksum_path):
        return str(runtime_path)
    cache_dir.mkdir(parents=True, exist_ok=True)
    return str(
        _download_runtime(
            runtime_version,
            target,
            runtime_path,
            checksum_path,
            options.release_base_url or resolved_env.get("META_RUNTIME_RELEASE_BASE_URL") or _DEFAULT_RELEASE_BASE_URL,
        )
    )


def _runtime_target() -> RuntimeTarget:
    os_name = (
        "windows"
        if sys.platform == "win32"
        else "darwin"
        if sys.platform == "darwin"
        else "linux"
        if sys.platform.startswith("linux")
        else None
    )
    machine = platform.machine().lower()
    arch = "amd64" if machine in {"amd64", "x86_64"} else "arm64" if machine in {"arm64", "aarch64"} else None
    if os_name is None or arch is None:
        raise RuntimeLaunchError(f"unsupported runtime target {sys.platform}/{platform.machine()}")
    return RuntimeTarget(os_name, arch, "meta-runtime.exe" if os_name == "windows" else "meta-runtime")


def _package_version() -> str:
    try:
        return package_version("mewisme-meta")
    except PackageNotFoundError:
        return "0.0.0"


def _normalize_version(value: str) -> str:
    version = value[1:] if value.startswith("v") else value
    if not version or re.fullmatch(r"[0-9A-Za-z][0-9A-Za-z.+-]*", version) is None:
        raise RuntimeLaunchError(f"invalid runtime version {value}")
    return version


def _default_cache_dir(env: Mapping[str, str]) -> Path:
    if sys.platform == "win32" and env.get("LOCALAPPDATA"):
        return Path(env["LOCALAPPDATA"]) / "meewmeew" / "meta-runtime"
    if env.get("XDG_CACHE_HOME"):
        return Path(env["XDG_CACHE_HOME"]) / "meewmeew" / "meta-runtime"
    return Path.home() / ".cache" / "meewmeew" / "meta-runtime"


def _verify_cached(runtime_path: Path, checksum_path: Path) -> bool:
    try:
        expected = checksum_path.read_text().strip().lower()
        return _sha256(runtime_path.read_bytes()) == expected
    except OSError:
        return False


def _download_runtime(
    version: str, target: RuntimeTarget, runtime_path: Path, checksum_path: Path, base_url: str
) -> Path:
    asset = f"meta-runtime_{version}_{target.os}_{target.arch}{'.exe' if target.os == 'windows' else ''}"
    root = f"{base_url.rstrip('/')}/v{version}"
    manifest = _fetch_bytes(f"{root}/meta_{version}_checksums.txt").decode()
    expected = _checksum_for(manifest, asset)
    binary = _fetch_bytes(f"{root}/{asset}")
    if _sha256(binary) != expected:
        raise RuntimeLaunchError(f"checksum mismatch for {asset}")
    temp = runtime_path.with_name(f"{runtime_path.name}.{os.getpid()}.tmp")
    try:
        temp.write_bytes(binary)
        if target.os != "windows":
            temp.chmod(0o755)
        os.replace(temp, runtime_path)
        checksum_path.write_text(f"{expected}\n")
    finally:
        temp.unlink(missing_ok=True)
    return runtime_path


def _fetch_bytes(url: str) -> bytes:
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            return bytes(response.read())
    except (OSError, urllib.error.URLError) as error:
        raise RuntimeLaunchError(f"failed to download runtime asset {url}") from error


def _checksum_for(manifest: str, asset: str) -> str:
    for line in manifest.splitlines():
        match = re.fullmatch(r"([a-fA-F0-9]{64})\s+\*?(.+)", line.strip())
        if match and match.group(2) == asset:
            return match.group(1).lower()
    raise RuntimeLaunchError(f"checksum manifest does not contain {asset}")


def _sha256(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()
