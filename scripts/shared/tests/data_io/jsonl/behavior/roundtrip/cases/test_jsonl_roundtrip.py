#!/usr/bin/env python3
from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from data_io.shared.contracts import load_jsonl, write_jsonl
from data_io.jsonl.behavior.roundtrip.shared.samples import ROUNDTRIP_ROWS


class JsonlHelperTests(unittest.TestCase):
    def test_write_jsonl_creates_parent_directories_and_reports_written_count(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "nested" / "rows.jsonl"

            count = write_jsonl(path, ROUNDTRIP_ROWS)

            self.assertEqual(count, len(ROUNDTRIP_ROWS))
            self.assertEqual(load_jsonl(path), ROUNDTRIP_ROWS)

    def test_load_jsonl_skips_blank_lines_after_round_trip(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "rows.jsonl"
            write_jsonl(path, ROUNDTRIP_ROWS)
            path.write_text(path.read_text(encoding="utf-8") + "\n", encoding="utf-8")

            self.assertEqual(load_jsonl(path), ROUNDTRIP_ROWS)


if __name__ == "__main__":
    unittest.main()
