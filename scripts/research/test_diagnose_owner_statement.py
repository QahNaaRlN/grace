"""Synthetic-only regression/privacy tests for the one-off read-only probe."""
import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

PATH = Path(__file__).with_name("diagnose-owner-statement.py")
spec = importlib.util.spec_from_file_location("probe", PATH)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


def fixture():
    spans = [dict(id="synthetic-prefix", page=1, block=1, line=1, word=1, text="2"),
             dict(id="synthetic-suffix", page=1, block=1, line=1, word=2, text="456,78")]
    return {"layout": {"spans": spans}, "review": {
        "rows": [dict(accountAmount=dict(amountMinor=245678, currency="RUB"),
                      direction=side, issues=[], description="PRIVATE_CANARY") for side in ("debit", "credit")],
        "header": {**{side + "Turnover": dict(amountMinor=45678, currency="RUB") for side in ("debit", "credit")},
                   "evidence": [dict(role=side + "Turnover", literal="456,78", spans=[spans[1]]) for side in ("debit", "credit")]},
        "reconciliation": dict(debitMinor=245678, creditMinor=245678,
                               blockers=["source_coverage_incomplete"], rowCoverage=dict(reasons=["unmapped_spans:7"]))}}


class ProbeTests(unittest.TestCase):
    def test_confirmed_adjacent_prefix(self):
        result = probe.safe_view(fixture())
        self.assertEqual(result, {key: True for key in probe.KEYS})
        output = json.dumps(result)
        for private in ("PRIVATE_CANARY", "245678", "456,78", "synthetic-prefix", "unmapped_spans:7"):
            self.assertNotIn(private, output)

    def test_no_prefix_inference_across_lines(self):
        data = fixture()
        data["layout"]["spans"][0]["line"] = 2
        self.assertFalse(probe.safe_view(data)["debit_omitted_prefix_confirmed"])

    def test_wrong_prefix_not_confirmed(self):
        data = fixture()
        data["layout"]["spans"][0]["text"] = "9"
        self.assertFalse(probe.safe_view(data)["credit_omitted_prefix_confirmed"])

    def test_matching_header_no_false_positive(self):
        data = fixture()
        for side in ("debit", "credit"):
            data["review"]["header"][side + "Turnover"]["amountMinor"] = 245678
        result = probe.safe_view(data)
        self.assertFalse(result["debit_header_mismatch"])
        self.assertFalse(result["debit_omitted_prefix_confirmed"])

    def test_bad_row_is_unavailable(self):
        data = fixture()
        data["review"]["rows"][0]["issues"] = ["PRIVATE_CANARY"]
        with self.assertRaises(ValueError):
            probe.safe_view(data)

    def test_invalid_target_never_runs_sql(self):
        with patch.object(probe.subprocess, "run") as run:
            with self.assertRaises(ValueError):
                probe.remote(dict(statementId="' OR TRUE --", sha256="a" * 64))
            run.assert_not_called()

    def test_sql_read_only_exact_target_and_no_content(self):
        responses = [subprocess.CompletedProcess([], 0, stdout="a" * 12),
                     subprocess.CompletedProcess([], 0, stdout=json.dumps(fixture()))]
        with patch.object(probe.subprocess, "run", side_effect=responses) as run:
            probe.remote(dict(statementId="11111111-2222-3333-4444-555555555555", sha256="a" * 64))
            sql = run.call_args.kwargs["input"]
            self.assertTrue(sql.startswith("BEGIN READ ONLY; SET LOCAL statement_timeout='5s'; SELECT"))
            self.assertTrue(sql.endswith("ROLLBACK;"))
            self.assertIn("AND d.sha256=", sql)
            self.assertNotIn("content", sql)
            self.assertNotIn("ai_usage", sql)
            self.assertEqual(sql.upper().count("SELECT"), 1)

    def test_exception_never_prints_trace_or_target(self):
        result = subprocess.run(["python3", str(PATH)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(json.loads(result.stdout), probe.UNAVAILABLE)
        self.assertEqual(result.stderr, "")

    def test_workflow_excludes_deploy(self):
        text = PATH.parents[2].joinpath(".github/workflows/deploy-production.yml").read_text()
        self.assertIn("if: ${{ !inputs.inspect_statement_only && !inputs.diagnose_owner_statement }}", text)
        self.assertIn("if: ${{ inputs.diagnose_owner_statement && inputs.inspect_statement_only && inputs.statement_action == 'preserve' }}", text)


if __name__ == "__main__":
    unittest.main()
