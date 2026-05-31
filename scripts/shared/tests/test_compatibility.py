from __future__ import annotations

import importlib
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

try:
    from shared.compatibility.runtime.path import add_scripts_root
    from shared.compatibility.wrappers.legacy import export_module, export_public
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.compatibility.runtime.path import add_scripts_root
    from scripts.shared.compatibility.wrappers.legacy import export_module, export_public


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
            package = root / "moved"
            package.mkdir()
            (package / "__init__.py").write_text("", encoding="utf-8")
            (package / "module.py").write_text(
                "__all__ = ['VALUE']\nVALUE = 7\nHIDDEN = 9\n", encoding="utf-8"
            )
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
            package = root / "moved"
            package.mkdir()
            (package / "__init__.py").write_text("", encoding="utf-8")
            (package / "runner.py").write_text(
                "def main():\n    return 3\n", encoding="utf-8"
            )
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
                module = importlib.import_module(name)
                self.assertTrue(module.__all__)


if __name__ == "__main__":
    unittest.main()
