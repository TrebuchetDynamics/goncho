"""Filesystem fixtures shared by JSONL round-trip case tests."""
from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path
import tempfile


@contextmanager
def temp_jsonl_path(*parts: str) -> Iterator[Path]:
    """Yield a JSONL path inside an isolated temporary workspace."""
    with tempfile.TemporaryDirectory() as td:
        yield Path(td).joinpath(*parts)


__all__ = ["temp_jsonl_path"]
