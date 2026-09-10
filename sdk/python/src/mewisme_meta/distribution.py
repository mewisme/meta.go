from __future__ import annotations

import importlib
import os
import platform
import shutil
import sys
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from types import ModuleType

from .errors import RuntimeLaunchError


@dataclass(frozen=True, slots=True)
class RuntimeDistributionOptions:
    runtime_path: str | Path | None = None
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
    bundled = _bundled_runtime_path(target)
    if bundled is not None:
        return str(bundled)
    installed = shutil.which(target.filename, path=resolved_env.get("PATH"))
    if installed:
        return installed
    raise RuntimeLaunchError(f"{target.filename} was not found in mewisme-meta-runtime or on PATH")


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


def _bundled_runtime_path(target: RuntimeTarget, module: ModuleType | None = None) -> Path | None:
    if module is None:
        try:
            module = importlib.import_module("mewisme_meta_runtime")
        except ModuleNotFoundError as error:
            if error.name == "mewisme_meta_runtime":
                return None
            raise RuntimeLaunchError("failed to import mewisme-meta-runtime") from error
    locator = getattr(module, "runtime_path", None)
    if not callable(locator):
        raise RuntimeLaunchError("mewisme-meta-runtime does not expose runtime_path()")
    path = Path(str(locator()))
    if not path.is_file():
        raise RuntimeLaunchError("mewisme-meta-runtime does not contain the runtime binary")
    if target.os != "windows" and not os.access(path, os.X_OK):
        raise RuntimeLaunchError("mewisme-meta-runtime binary is not executable")
    return path
