from __future__ import annotations

import importlib
import sys
import tempfile
import unittest
from pathlib import Path

from support import import_attrs, import_first, write_package_module

add_scripts_root, = import_attrs(
    ("shared.compatibility.path", "scripts.shared.compatibility.path"),
    "add_scripts_root",
)
export_module, export_public = import_attrs(
    ("shared.compatibility.legacy", "scripts.shared.compatibility.legacy"),
    "export_module",
    "export_public",
)


class CompatibilityHelpersTest(unittest.TestCase):
    def test_add_scripts_root_uses_wrapper_parent(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            wrapper = Path(td) / "legacy_wrapper.py"
            add_scripts_root(str(wrapper))
            self.assertEqual(sys.path[0], td)
            add_scripts_root(str(wrapper))
            self.assertEqual(sys.path.count(td), 1)
            sys.path.remove(td)

    def test_export_public_exports_declared_names_from_moved_module(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            write_package_module(root, "moved", "module", "__all__ = ['VALUE']\nVALUE = 7\nHIDDEN = 9\n")
            namespace: dict[str, object] = {}
            try:
                module = export_public("moved.module", str(root / "wrapper.py"), namespace)
                self.assertEqual(module.VALUE, 7)
                self.assertEqual(namespace["VALUE"], 7)
                self.assertNotIn("HIDDEN", namespace)
            finally:
                sys.path.remove(td)
                sys.modules.pop("moved", None)
                sys.modules.pop("moved.module", None)

    def test_export_module_returns_and_exports_main(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            write_package_module(root, "moved", "runner", "def main():\n    return 3\n")
            namespace: dict[str, object] = {}
            try:
                main = export_module("moved.runner", str(root / "wrapper.py"), namespace)
                self.assertEqual(main(), 3)
                self.assertIs(namespace["main"], main)
            finally:
                sys.path.remove(td)
                sys.modules.pop("moved", None)
                sys.modules.pop("moved.runner", None)

    def test_legacy_facades_keep_existing_import_paths(self) -> None:
        modules = [
            "shared.compat",
            "shared.legacy",
            "shared.compatibility.path",
            "shared.compatibility.legacy",
        ]
        for name in modules:
            with self.subTest(name=name):
                module = import_first(name, f"scripts.{name}")
                self.assertTrue(module.__all__)


if __name__ == "__main__":
    unittest.main()
