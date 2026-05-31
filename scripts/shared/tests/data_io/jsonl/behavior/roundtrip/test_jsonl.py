#!/usr/bin/env python3
from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from data_io.shared.contracts import load_jsonl, write_jsonl


class JsonlHelperTests(unittest.TestCase):
    def test_write_then_load_jsonl_round_trips_objects_and_skips_blank_lines(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "nested" / "rows.jsonl"
            count = write_jsonl(path, [{"text": "café"}, {"n": 2}])
            path.write_text(path.read_text(encoding="utf-8") + "\n", encoding="utf-8")

            self.assertEqual(count, 2)
            self.assertEqual(load_jsonl(path), [{"text": "café"}, {"n": 2}])


if __name__ == "__main__":
    unittest.main()
