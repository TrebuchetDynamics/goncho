#!/usr/bin/env python3
from __future__ import annotations

import unittest

from data_io.jsonl.behavior.roundtrip.cases.case_support.workspace import temp_jsonl_path
from data_io.shared.contracts import load_jsonl, write_jsonl
from data_io.jsonl.behavior.roundtrip.shared.samples import ROUNDTRIP_ROWS


class JsonlBlankLineRoundTripTests(unittest.TestCase):
    def test_load_jsonl_skips_blank_lines_after_round_trip(self) -> None:
        with temp_jsonl_path("rows.jsonl") as path:
            write_jsonl(path, ROUNDTRIP_ROWS)
            path.write_text(path.read_text(encoding="utf-8") + "\n", encoding="utf-8")

            self.assertEqual(load_jsonl(path), ROUNDTRIP_ROWS)


if __name__ == "__main__":
    unittest.main()
