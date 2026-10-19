#!/usr/bin/env python3
import importlib.util
import json
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("diagnostic", Path(__file__).with_name("diagnose-statement-smoke.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class DiagnosticPrivacy(unittest.TestCase):
    def test_only_fixed_corpus_and_canonical_references_escape(self):
        private = "PRIVATE-NOT-EXPORTED"
        report = {"version": "statement.synthetic.v2.document-alias", "calls": [{"generationId": private, "costMicros": 9234567}],
                  "traces": [{"name": "columns", "parts": [{"rows": [{"operationDate": ["p1-b1-l1-w1", private], "direction": private}],
                    "header": {"currency": ["p1-b2-l1-w1"], "labels": {"debitTurnover": ["p1-b3-l1-w1"]}},
                    "service": [{"spans": ["p1-b4-l1-w1"], "reason": private}]}]},
                    {"name": private, "parts": []}]}
        raw = json.dumps(module.safe_view(report))
        self.assertIn("p1-b1-l1-w1", raw)
        self.assertIn("p1-b4-l1-w1", raw)
        self.assertNotIn(private, raw)
        self.assertNotIn("9234567", raw)
        self.assertNotIn("generation", raw)
        self.assertNotIn("cost", raw)

    def test_unknown_report_rejected(self):
        with self.assertRaises(ValueError):
            module.safe_view({"version": "private-document"})


if __name__ == "__main__":
    unittest.main()
