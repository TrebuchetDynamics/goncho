#!/usr/bin/env python3
from __future__ import annotations

import unittest

from data_io.jsonl.behavior.roundtrip.cases.case_support.workspace import temp_jsonl_path
from data_io.shared.contracts import load_jsonl, write_jsonl
from data_io.jsonl.behavior.roundtrip.shared.payloads.fixtures import ROUNDTRIP_ROWS


class JsonlWriteRoundTripTests(unittest.TestCase):
    def test_write_jsonl_creates_parent_directories_and_reports_written_count(self) -> None:
        with temp_jsonl_path("nested", "rows.jsonl") as path:
            count = write_jsonl(path, ROUNDTRIP_ROWS)

            self.assertEqual(count, len(ROUNDTRIP_ROWS))
            self.assertEqual(load_jsonl(path), ROUNDTRIP_ROWS)


if __name__ == "__main__":
    unittest.main()
