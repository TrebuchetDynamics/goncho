"""Helpers for thin compatibility wrappers in scripts/."""
from __future__ import annotations

import importlib
from collections.abc import MutableMapping
from typing import Any, Callable

try:
    from shared.compatibility.path import add_scripts_root
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compatibility.path import add_scripts_root


def export_public(module_name: str, anchor: str, namespace: MutableMapping[str, Any]) -> Any:
    """Export public names from a moved module into a legacy wrapper."""
    add_scripts_root(anchor)
    module = importlib.import_module(module_name)
    public_names = getattr(module, "__all__", None)
    if public_names is None:
        public_names = [name for name in vars(module) if not name.startswith("_")]
    namespace.update({name: getattr(module, name) for name in public_names})
    return module


def export_module(module_name: str, anchor: str, namespace: MutableMapping[str, Any]) -> Callable[[], int | None]:
    """Export public names from a moved module and return its ``main`` callable."""
    module = export_public(module_name, anchor, namespace)
    main = getattr(module, "main")
    namespace["main"] = main
    return main
