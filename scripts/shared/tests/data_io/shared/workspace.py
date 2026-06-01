"""Temporary workspace contracts shared by data I/O behavior tests."""
from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path
import tempfile


@contextmanager
def temp_workspace_path(*parts: str) -> Iterator[Path]:
    """Yield a path inside an isolated temporary workspace."""
    with tempfile.TemporaryDirectory() as td:
        yield Path(td).joinpath(*parts)


__all__ = ["temp_workspace_path"]
