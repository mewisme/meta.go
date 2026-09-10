from __future__ import annotations

import os
from pathlib import Path


def runtime_path() -> str:
    return str(Path(__file__).with_name("meta-runtime.exe" if os.name == "nt" else "meta-runtime"))


__all__ = ["runtime_path"]
