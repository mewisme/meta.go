from __future__ import annotations

import argparse
import base64
import csv
import hashlib
import io
import re
import stat
import zipfile
from pathlib import Path

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
DIST = "mewisme_meta_runtime"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--target", required=True, choices=sorted(TARGETS))
    parser.add_argument("--version", required=True)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--out-dir", required=True, type=Path)
    args = parser.parse_args()
    validate_version(args.version)
    binary = args.binary.resolve()
    if not binary.is_file():
        raise SystemExit(f"runtime binary does not exist: {binary}")
    out = build_wheel(args.target, args.version, binary, args.out_dir.resolve())
    print(out)


def build_wheel(target: str, version: str, binary: Path, out_dir: Path) -> Path:
    platform_tag = TARGETS[target]
    filename = "meta-runtime.exe" if target.startswith("win32-") else "meta-runtime"
    wheel = out_dir / f"{DIST}-{version}-py3-none-{platform_tag}.whl"
    dist_info = f"{DIST}-{version}.dist-info"
    repo_root = Path(__file__).resolve().parents[3]
    locator = (Path(__file__).resolve().parents[1] / "runtime" / "src" / DIST / "__init__.py").read_bytes()
    files = {
        f"{DIST}/__init__.py": (locator, 0o644),
        f"{DIST}/{filename}": (binary.read_bytes(), 0o755),
        f"{dist_info}/METADATA": (metadata(version).encode(), 0o644),
        f"{dist_info}/WHEEL": (wheel_metadata(platform_tag).encode(), 0o644),
        f"{dist_info}/licenses/LICENSE": ((repo_root / "LICENSE").read_bytes(), 0o644),
    }
    out_dir.mkdir(parents=True, exist_ok=True)
    records = []
    with zipfile.ZipFile(wheel, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for name, (data, mode) in files.items():
            write_member(archive, name, data, mode)
            records.append((name, wheel_hash(data), str(len(data))))
        record_name = f"{dist_info}/RECORD"
        record = record_csv([*records, (record_name, "", "")]).encode()
        write_member(archive, record_name, record, 0o644)
    return wheel


def metadata(version: str) -> str:
    return (
        "Metadata-Version: 2.4\n"
        "Name: mewisme-meta-runtime\n"
        f"Version: {version}\n"
        "Summary: Platform runtime for mewisme-meta\n"
        "Requires-Python: >=3.10\n"
        "License-Expression: AGPL-3.0-only\n"
        "Project-URL: Repository, https://github.com/mewisme/meta.go\n"
        "\n"
    )


def wheel_metadata(platform_tag: str) -> str:
    return "Wheel-Version: 1.0\nGenerator: meta.go\nRoot-Is-Purelib: false\nTag: py3-none-" + platform_tag + "\n"


def write_member(archive: zipfile.ZipFile, name: str, data: bytes, mode: int) -> None:
    info = zipfile.ZipInfo(name, (2020, 1, 1, 0, 0, 0))
    info.create_system = 3
    info.external_attr = (stat.S_IFREG | mode) << 16
    info.compress_type = zipfile.ZIP_DEFLATED
    archive.writestr(info, data)


def wheel_hash(data: bytes) -> str:
    digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()
    return f"sha256={digest}"


def record_csv(rows: list[tuple[str, str, str]]) -> str:
    output = io.StringIO(newline="")
    csv.writer(output, lineterminator="\n").writerows(rows)
    return output.getvalue()


def validate_version(version: str) -> None:
    if re.fullmatch(r"\d+(?:\.\d+){2,}(?:(?:a|b|rc)\d+)?", version) is None:
        raise SystemExit(f"invalid package version: {version}")


if __name__ == "__main__":
    main()
