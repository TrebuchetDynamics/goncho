from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

from support import ensure_scripts_parent


REPO_ROOT = ensure_scripts_parent()
SCRIPTS_ROOT = REPO_ROOT / "scripts"


WRAPPER_IMPORT_CHECK = """
import importlib.util
import sys
from pathlib import Path
wrapper = Path(sys.argv[1])
spec = importlib.util.spec_from_file_location(f'legacy_wrapper_{wrapper.stem}', wrapper)
if spec is None or spec.loader is None:
    raise SystemExit(f'could not load import spec for {wrapper}')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
text = wrapper.read_text(encoding='utf-8')
if 'export_module' in text and not callable(getattr(module, 'main', None)):
    raise SystemExit(f'{wrapper} did not export callable main')
"""


def test_legacy_script_wrappers_import_their_moved_modules() -> None:
    wrappers = sorted(
        path
        for path in SCRIPTS_ROOT.glob("*.py")
        if "Compatibility wrapper" in path.read_text(encoding="utf-8")
    )

    assert wrappers, "expected at least one legacy script wrapper"

    env = {**os.environ, "PYTHONPATH": str(SCRIPTS_ROOT)}
    for wrapper in wrappers:
        result = subprocess.run(
            [sys.executable, "-c", WRAPPER_IMPORT_CHECK, str(wrapper)],
            cwd=REPO_ROOT,
            env=env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        assert result.returncode == 0, result.stdout + result.stderr
