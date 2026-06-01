"""Filesystem fixtures shared by checksum file-hashing case tests."""
from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

from data_io.shared.workspace import temp_workspace_path


@contextmanager
def temp_binary_file(filename: str, payload: bytes) -> Iterator[Path]:
    """Yield a binary file with payload inside an isolated temporary workspace."""
    with temp_workspace_path(filename) as path:
        path.write_bytes(payload)
        yield path


__all__ = ["temp_binary_file"]
