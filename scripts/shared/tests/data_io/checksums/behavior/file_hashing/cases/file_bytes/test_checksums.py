#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import unittest

from data_io.checksums.behavior.file_hashing.cases.case_support.workspace import temp_binary_file
from data_io.shared.contracts import sha256


class ChecksumHelperTests(unittest.TestCase):
    def test_sha256_matches_hashlib_for_file_bytes(self) -> None:
        payload = b"chunked payload"

        with temp_binary_file("payload.bin", payload) as path:
            self.assertEqual(sha256(path), hashlib.sha256(payload).hexdigest())


if __name__ == "__main__":
    unittest.main()
