from __future__ import annotations

import csv
import subprocess
import sys
import zipfile
from pathlib import Path

import pytest

TARGETS = {
    "linux-x64-manylinux": "manylinux_2_17_x86_64",
    "linux-x64-musllinux": "musllinux_1_2_x86_64",
    "linux-arm64-manylinux": "manylinux_2_17_aarch64",
    "linux-arm64-musllinux": "musllinux_1_2_aarch64",
    "darwin-x64": "macosx_11_0_x86_64",
    "darwin-arm64": "macosx_11_0_arm64",
    "win32-x64": "win_amd64",
    "win32-arm64": "win_arm64",
}


@pytest.mark.parametrize(("target", "platform_tag"), TARGETS.items())
def test_runtime_wheel_builder_emits_platform_specific_wheel(tmp_path: Path, target: str, platform_tag: str) -> None:
    sdk_dir = Path(__file__).resolve().parents[1]
    binary = tmp_path / ("meta-runtime.exe" if target.startswith("win32-") else "meta-runtime")
    binary.write_bytes(b"runtime")
    script = sdk_dir / "scripts" / "package-runtime.py"
    subprocess.run(
        [
            sys.executable,
            str(script),
            "--target",
            target,
            "--version",
            "0.260911.0",
            "--binary",
            str(binary),
            "--out-dir",
            str(tmp_path),
        ],
        check=True,
        capture_output=True,
        text=True,
    )
    wheel = tmp_path / f"mewisme_meta_runtime-0.260911.0-py3-none-{platform_tag}.whl"
    assert wheel.is_file()
    with zipfile.ZipFile(wheel) as archive:
        names = set(archive.namelist())
        filename = "meta-runtime.exe" if target.startswith("win32-") else "meta-runtime"
        runtime_name = f"mewisme_meta_runtime/{filename}"
        assert runtime_name in names
        assert "mewisme_meta_runtime/__init__.py" in names
        assert archive.getinfo(runtime_name).external_attr >> 16 & 0o777 == 0o755
        wheel_metadata = archive.read("mewisme_meta_runtime-0.260911.0.dist-info/WHEEL").decode()
        assert f"Tag: py3-none-{platform_tag}" in wheel_metadata
        metadata = archive.read("mewisme_meta_runtime-0.260911.0.dist-info/METADATA").decode()
        assert "Name: mewisme-meta-runtime" in metadata
        assert "Version: 0.260911.0" in metadata
        record = list(
            csv.reader(archive.read("mewisme_meta_runtime-0.260911.0.dist-info/RECORD").decode().splitlines())
        )
        assert record[-1] == ["mewisme_meta_runtime-0.260911.0.dist-info/RECORD", "", ""]


def test_release_preparation_pins_runtime_dependency(tmp_path: Path) -> None:
    sdk_dir = Path(__file__).resolve().parents[1]
    project = tmp_path / "python"
    (project / "runtime").mkdir(parents=True)
    (project / "pyproject.toml").write_text((sdk_dir / "pyproject.toml").read_text())
    (project / "runtime" / "pyproject.toml").write_text((sdk_dir / "runtime" / "pyproject.toml").read_text())
    subprocess.run(
        [sys.executable, str(sdk_dir / "scripts" / "prepare-release.py"), "0.260911.0", "--project-dir", str(project)],
        check=True,
    )
    root = (project / "pyproject.toml").read_text()
    runtime = (project / "runtime" / "pyproject.toml").read_text()
    assert 'version = "0.260911.0"' in root
    assert '"mewisme-meta-runtime==0.260911.0"' in root
    assert 'version = "0.260911.0"' in runtime
