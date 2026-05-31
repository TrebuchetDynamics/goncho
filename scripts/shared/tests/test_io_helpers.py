#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))

try:
    from shared.checksums import sha256
    from shared.jsonl import load_jsonl, write_jsonl
except ModuleNotFoundError:  # pragma: no cover - package import path
    from scripts.shared.checksums import sha256
    from scripts.shared.jsonl import load_jsonl, write_jsonl


class IoHelperTests(unittest.TestCase):
    def test_write_then_load_jsonl_round_trips_objects_and_skips_blank_lines(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "nested" / "rows.jsonl"
            count = write_jsonl(path, [{"text": "café"}, {"n": 2}])
            path.write_text(path.read_text(encoding="utf-8") + "\n", encoding="utf-8")

            self.assertEqual(count, 2)
            self.assertEqual(load_jsonl(path), [{"text": "café"}, {"n": 2}])

    def test_sha256_matches_hashlib_for_file_bytes(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "payload.bin"
            payload = b"chunked payload"
            path.write_bytes(payload)

            self.assertEqual(sha256(path), hashlib.sha256(payload).hexdigest())


if __name__ == "__main__":
    unittest.main()
