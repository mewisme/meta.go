from __future__ import annotations

import argparse
import re
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("version")
    parser.add_argument("--project-dir", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    if re.fullmatch(r"\d+(?:\.\d+){2,}(?:(?:a|b|rc)\d+)?", args.version) is None:
        raise SystemExit(f"invalid package version: {args.version}")
    project = args.project_dir.resolve()
    update_root(project / "pyproject.toml", args.version)
    update_runtime(project / "runtime" / "pyproject.toml", args.version)


def update_root(path: Path, version: str) -> None:
    text = path.read_text()
    text, count = re.subn(r'(?m)^version = "[^"]+"$', f'version = "{version}"', text, count=1)
    if count != 1:
        raise SystemExit(f"could not update project version in {path}")
    text = re.sub(r'(?m)^  "mewisme-meta-runtime==[^\"]+",\n', "", text)
    marker = "dependencies = [\n"
    if text.count(marker) != 1:
        raise SystemExit(f"could not locate project dependencies in {path}")
    text = text.replace(marker, marker + f'  "mewisme-meta-runtime=={version}",\n', 1)
    path.write_text(text)


def update_runtime(path: Path, version: str) -> None:
    text = path.read_text()
    text, count = re.subn(r'(?m)^version = "[^"]+"$', f'version = "{version}"', text, count=1)
    if count != 1:
        raise SystemExit(f"could not update runtime version in {path}")
    path.write_text(text)


if __name__ == "__main__":
    main()
