from __future__ import annotations

import argparse
import json
import tarfile
import zipfile
from pathlib import Path

import tomllib

NODE_TARGETS = ("darwin-arm64", "darwin-x64", "linux-arm64", "linux-x64", "win32-arm64", "win32-x64")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", required=True)
    parser.add_argument("--node-dir", type=Path)
    parser.add_argument("--python-dir", type=Path)
    parser.add_argument("--node-runtime-dir", type=Path)
    parser.add_argument("--python-runtime-dir", type=Path)
    parser.add_argument("--node-sdk-archive", type=Path)
    parser.add_argument("--python-sdk-wheel", type=Path)
    args = parser.parse_args()
    if args.node_dir:
        validate_node(args.node_dir, args.version)
    if args.python_dir:
        validate_python(args.python_dir, args.version)
    if args.node_runtime_dir:
        validate_node_runtime_artifacts(args.node_runtime_dir, args.version)
    if args.python_runtime_dir:
        validate_python_runtime_artifacts(args.python_runtime_dir, args.version)
    if args.node_sdk_archive:
        validate_node_sdk_archive(args.node_sdk_archive, args.version)
    if args.python_sdk_wheel:
        validate_python_sdk_wheel(args.python_sdk_wheel, args.version)
    if not any(
        (
            args.node_dir,
            args.python_dir,
            args.node_runtime_dir,
            args.python_runtime_dir,
            args.node_sdk_archive,
            args.python_sdk_wheel,
        )
    ):
        raise SystemExit("no release artifact or manifest input was provided")


def validate_node(root: Path, version: str) -> None:
    package = json.loads((root / "package.json").read_text())
    if package.get("version") != version:
        raise SystemExit(f"Node SDK version mismatch: {package.get('version')} != {version}")
    expected = {f"@meewmeew/meta-runtime-{target}": version for target in NODE_TARGETS}
    if package.get("optionalDependencies") != expected:
        raise SystemExit("Node SDK optional runtime dependencies do not exactly match the release version")
    for target in NODE_TARGETS:
        runtime = json.loads((root / "runtime-packages" / target / "package.json").read_text())
        if runtime.get("version") != version:
            raise SystemExit(f"Node runtime {target} version mismatch")


def validate_python(root: Path, version: str) -> None:
    project = tomllib.loads((root / "pyproject.toml").read_text())["project"]
    if project.get("version") != version:
        raise SystemExit(f"Python SDK version mismatch: {project.get('version')} != {version}")
    dependency = f"mewisme-meta-runtime=={version}"
    if dependency not in project.get("dependencies", []):
        raise SystemExit(f"Python SDK does not pin {dependency}")
    runtime = tomllib.loads((root / "runtime" / "pyproject.toml").read_text())["project"]
    if runtime.get("version") != version:
        raise SystemExit(f"Python runtime version mismatch: {runtime.get('version')} != {version}")


def validate_node_runtime_artifacts(root: Path, version: str) -> None:
    archives = sorted(root.glob("meewmeew-meta-runtime-*.tgz"))
    if len(archives) != 6:
        raise SystemExit(f"expected 6 Node runtime packages, found {len(archives)}")
    names = set()
    for archive in archives:
        with tarfile.open(archive, "r:gz") as bundle:
            file_members = {member.name: member for member in bundle.getmembers() if member.isfile()}
            members = set(file_members)
            package = json.load(bundle.extractfile("package/package.json"))  # type: ignore[arg-type]
        name = package.get("name")
        if name not in {f"@meewmeew/meta-runtime-{target}" for target in NODE_TARGETS}:
            raise SystemExit(f"unexpected Node runtime package name in {archive}: {name}")
        if package.get("version") != version:
            raise SystemExit(f"Node runtime package version mismatch in {archive}")
        runtime = "package/meta-runtime.exe" if "win32" in name else "package/meta-runtime"
        expected = {"package/LICENSE", "package/README.md", "package/package.json", runtime}
        if members != expected:
            raise SystemExit(f"unexpected Node runtime package contents in {archive}: {sorted(members)}")
        if file_members[runtime].mode & 0o111 == 0:
            raise SystemExit(f"Node runtime binary is not executable in {archive}")
        names.add(name)
    if len(names) != 6:
        raise SystemExit("Node runtime package set is incomplete")


def validate_python_runtime_artifacts(root: Path, version: str) -> None:
    expected_tags = {
        "macosx_11_0_arm64",
        "macosx_11_0_x86_64",
        "manylinux_2_17_aarch64",
        "manylinux_2_17_x86_64",
        "musllinux_1_2_aarch64",
        "musllinux_1_2_x86_64",
        "win_amd64",
        "win_arm64",
    }
    wheels = sorted(root.glob("*.whl"))
    if len(wheels) != len(expected_tags):
        raise SystemExit(f"expected {len(expected_tags)} Python runtime wheels, found {len(wheels)}")
    found = set()
    for wheel in wheels:
        prefix = f"mewisme_meta_runtime-{version}-py3-none-"
        if not wheel.name.startswith(prefix) or not wheel.name.endswith(".whl"):
            raise SystemExit(f"unexpected Python runtime wheel name: {wheel.name}")
        tag = wheel.name[len(prefix) : -4]
        found.add(tag)
        with zipfile.ZipFile(wheel) as archive:
            metadata_name = f"mewisme_meta_runtime-{version}.dist-info/METADATA"
            metadata = archive.read(metadata_name).decode()
            runtime = (
                "mewisme_meta_runtime/meta-runtime.exe"
                if tag.startswith("win_")
                else "mewisme_meta_runtime/meta-runtime"
            )
            expected = {
                "mewisme_meta_runtime/__init__.py",
                runtime,
                metadata_name,
                f"mewisme_meta_runtime-{version}.dist-info/WHEEL",
                f"mewisme_meta_runtime-{version}.dist-info/RECORD",
                f"mewisme_meta_runtime-{version}.dist-info/licenses/LICENSE",
            }
            if set(archive.namelist()) != expected:
                raise SystemExit(f"unexpected Python runtime wheel contents in {wheel}")
            if archive.getinfo(runtime).external_attr >> 16 & 0o111 == 0:
                raise SystemExit(f"Python runtime binary is not executable in {wheel}")
        if f"Version: {version}\n" not in metadata:
            raise SystemExit(f"Python runtime wheel version mismatch in {wheel}")
    if found != expected_tags:
        raise SystemExit(f"Python runtime wheel platform set mismatch: {sorted(found)}")


def validate_node_sdk_archive(archive: Path, version: str) -> None:
    with tarfile.open(archive, "r:gz") as bundle:
        package = json.load(bundle.extractfile("package/package.json"))  # type: ignore[arg-type]
        names = {member.name for member in bundle.getmembers() if member.isfile()}
    expected = {f"@meewmeew/meta-runtime-{target}": version for target in NODE_TARGETS}
    if package.get("name") != "@meewmeew/meta" or package.get("version") != version:
        raise SystemExit("Node SDK package identity/version mismatch")
    if package.get("optionalDependencies") != expected:
        raise SystemExit("Node SDK runtime dependencies are not exact release-version pins")
    if any(name.endswith("/meta-runtime") or name.endswith("/meta-runtime.exe") for name in names):
        raise SystemExit("Node SDK tarball must not embed a runtime binary")


def validate_python_sdk_wheel(wheel: Path, version: str) -> None:
    with zipfile.ZipFile(wheel) as archive:
        metadata = archive.read(f"mewisme_meta-{version}.dist-info/METADATA").decode()
        names = set(archive.namelist())
    if "Name: mewisme-meta\n" not in metadata or f"Version: {version}\n" not in metadata:
        raise SystemExit("Python SDK package identity/version mismatch")
    if f"Requires-Dist: mewisme-meta-runtime=={version}\n" not in metadata:
        raise SystemExit("Python SDK runtime dependency is not an exact release-version pin")
    if any(name.endswith("/meta-runtime") or name.endswith("/meta-runtime.exe") for name in names):
        raise SystemExit("Python SDK wheel must not embed a runtime binary")


if __name__ == "__main__":
    main()
