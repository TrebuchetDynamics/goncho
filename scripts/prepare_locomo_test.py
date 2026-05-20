#!/usr/bin/env python3
import json
import tempfile
import unittest
from pathlib import Path

import prepare_locomo


class PrepareLocomoTests(unittest.TestCase):
    def test_converter_edge_cases(self):
        raw = [
            {
                "sample_id": "edge",
                "conversation": {
                    "speaker_a": "A",
                    "speaker_b": "B",
                    "session_1_date_time": "",
                    "session_1": [
                        {"speaker": "A", "dia_id": "D1:1", "text": "Alex stores duplicate notes."},
                        {"speaker": "B", "dia_id": "D1:2", "text": "Alex stores duplicate notes."},
                    ],
                    "session_2_date_time": "2:00 pm on 2 June, 2023",
                    "session_2": [
                        {"speaker": "A", "dia_id": "D2:1", "text": "Alex changed the storage location to a cabinet."},
                    ],
                },
                "qa": [
                    {
                        "question": "Where did Alex change the storage location?",
                        "answer": "cabinet",
                        "evidence": ["D2:1"],
                        "category": 1,
                    },
                    {
                        "question": "Who mentioned duplicate notes?",
                        "answer": "Alex or B",
                        "evidence": ["D1:1", "D1:2"],
                        "category": 3,
                    },
                ],
            }
        ]
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            raw_path = root / "raw.json"
            memories = root / "memories.jsonl"
            questions = root / "questions.jsonl"
            metadata = root / "metadata.json"
            raw_path.write_text(json.dumps(raw))
            meta = prepare_locomo.convert(raw_path, memories, questions, metadata)
            mem_rows = [json.loads(line) for line in memories.read_text().splitlines()]
            q_rows = [json.loads(line) for line in questions.read_text().splitlines()]

        self.assertEqual(meta["memory_count"], 3)
        self.assertEqual(meta["question_count"], 2)
        self.assertEqual(mem_rows[0]["timestamp"], "")
        self.assertEqual(mem_rows[0]["speaker"], "A")
        self.assertEqual(mem_rows[1]["speaker"], "B")
        self.assertEqual(mem_rows[0]["content"], mem_rows[1]["content"])
        self.assertEqual(q_rows[1]["gold_memory_ids"], ["locomo-edge-D1-1", "locomo-edge-D1-2"])
        self.assertEqual(q_rows[1]["category"], "multi_hop_retrieval")

    def test_converter_records_missing_evidence(self):
        raw = [{
            "sample_id": "missing",
            "conversation": {"session_1": [{"speaker": "A", "dia_id": "D1:1", "text": "Only one turn."}]},
            "qa": [{"question": "What exists?", "answer": "one", "evidence": ["D1:1", "D9:9"], "category": 1}],
        }]
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            raw_path = root / "raw.json"
            memories = root / "memories.jsonl"
            questions = root / "questions.jsonl"
            metadata = root / "metadata.json"
            raw_path.write_text(json.dumps(raw))
            meta = prepare_locomo.convert(raw_path, memories, questions, metadata)
            q_rows = [json.loads(line) for line in questions.read_text().splitlines()]
        self.assertEqual(meta["missing_evidence_count"], 1)
        self.assertEqual(q_rows[0]["gold_memory_ids"], ["locomo-missing-D1-1"])

    def test_normalize_composite_evidence(self):
        got = prepare_locomo.normalize_evidence_items(["D8:6; D9:17", "D21-18 D21-22 D11-15"])
        self.assertEqual(got, ["D8:6", "D9:17", "D21:18", "D21:22", "D11:15"])


if __name__ == "__main__":
    unittest.main()
