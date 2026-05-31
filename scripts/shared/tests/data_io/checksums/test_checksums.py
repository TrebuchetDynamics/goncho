#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import tempfile
import unittest
from pathlib import Path

from data_io.shared.contracts import sha256


class ChecksumHelperTests(unittest.TestCase):
    def test_sha256_matches_hashlib_for_file_bytes(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "payload.bin"
            payload = b"chunked payload"
            path.write_bytes(payload)

            self.assertEqual(sha256(path), hashlib.sha256(payload).hexdigest())


if __name__ == "__main__":
    unittest.main()
