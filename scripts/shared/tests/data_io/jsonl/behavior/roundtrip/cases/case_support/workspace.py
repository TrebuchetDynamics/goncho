"""Filesystem fixtures shared by JSONL round-trip case tests."""
from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

from data_io.shared.workspace import temp_workspace_path


@contextmanager
def temp_jsonl_path(*parts: str) -> Iterator[Path]:
    """Yield a JSONL path inside an isolated temporary workspace."""
    with temp_workspace_path(*parts) as path:
        yield path


__all__ = ["temp_jsonl_path"]
