from __future__ import annotations

from pathlib import Path
from types import ModuleType

import pytest

from mewisme_meta import RuntimeDistributionOptions, RuntimeLaunchError, resolve_runtime_path
from mewisme_meta.distribution import _bundled_runtime_path, _runtime_target


def test_resolve_runtime_path_prefers_explicit_and_environment_paths() -> None:
    assert (
        resolve_runtime_path(
            RuntimeDistributionOptions(runtime_path="/explicit/runtime"), {"META_RUNTIME_PATH": "/env/runtime"}
        )
        == "/explicit/runtime"
    )
    assert resolve_runtime_path(RuntimeDistributionOptions(), {"META_RUNTIME_PATH": "/env/runtime"}) == "/env/runtime"


def test_resolve_runtime_path_falls_back_to_path_without_downloading(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setattr("mewisme_meta.distribution._bundled_runtime_path", lambda target: None)
    target = _runtime_target()
    runtime = tmp_path / target.filename
    runtime.write_text("runtime")
    runtime.chmod(0o755)
    assert resolve_runtime_path(RuntimeDistributionOptions(), {"PATH": str(tmp_path), "META_RUNTIME_PATH": ""}) == str(
        runtime
    )


def test_resolve_runtime_path_fails_locally_without_runtime(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("mewisme_meta.distribution._bundled_runtime_path", lambda target: None)
    with pytest.raises(RuntimeLaunchError, match="mewisme-meta-runtime or on PATH"):
        resolve_runtime_path(RuntimeDistributionOptions(), {"PATH": "", "META_RUNTIME_PATH": ""})


def test_bundled_runtime_locator_validates_binary(tmp_path: Path) -> None:
    target = _runtime_target()
    runtime = tmp_path / target.filename
    runtime.write_text("runtime")
    if target.os != "windows":
        runtime.chmod(0o755)
    module = ModuleType("mewisme_meta_runtime")
    module.runtime_path = lambda: str(runtime)  # type: ignore[attr-defined]
    assert _bundled_runtime_path(target, module) == runtime
    if target.os != "windows":
        runtime.chmod(0o644)
        with pytest.raises(RuntimeLaunchError, match="not executable"):
            _bundled_runtime_path(target, module)
