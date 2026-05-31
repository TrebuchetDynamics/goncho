"""Shared test helpers for scripts.shared helper modules."""
from __future__ import annotations

import importlib
import sys
from collections.abc import Iterable
from pathlib import Path
from types import ModuleType
from typing import Any


def write_package_module(root: Path, package_name: str, module_name: str, content: str) -> Path:
    """Create a small importable package module for compatibility-wrapper tests."""
    package = root / package_name
    package.mkdir()
    (package / "__init__.py").write_text("", encoding="utf-8")
    module_path = package / f"{module_name}.py"
    module_path.write_text(content, encoding="utf-8")
    return module_path


def ensure_scripts_parent() -> Path:
    """Put both scripts/ and its parent on sys.path for compatibility imports."""
    scripts_dir = Path(__file__).resolve().parents[2]
    repo_root = scripts_dir.parent
    for import_root in (repo_root, scripts_dir):
        import_root_text = str(import_root)
        if import_root_text not in sys.path:
            sys.path.insert(0, import_root_text)
    return repo_root


def import_first(*module_names: str) -> ModuleType:
    """Import the first available module name from compatibility alternatives."""
    ensure_scripts_parent()
    last_error: ModuleNotFoundError | None = None
    for module_name in module_names:
        try:
            return importlib.import_module(module_name)
        except ModuleNotFoundError as exc:
            last_error = exc
    if last_error is None:
        raise ValueError("at least one module name is required")
    raise last_error


def import_attrs(module_names: Iterable[str], *attrs: str) -> tuple[Any, ...]:
    """Import named attributes from the first available module path."""
    module = import_first(*module_names)
    return tuple(getattr(module, attr) for attr in attrs)
