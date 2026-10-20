#!/usr/bin/env python3
"""Bounded owner-authorized read-only diagnosis; export fixed booleans only.

The exact target and document hash travel through an encrypted Actions secret,
then SSH stdin. No financial text, values, locators, or identifiers leave VPS.
"""
import json
import os
import re
import subprocess
import sys
from decimal import Decimal, InvalidOperation
from pathlib import Path

KEYS = ("target_verified", "row_sums_consistent", "debit_header_mismatch",
        "credit_header_mismatch", "debit_omitted_prefix_confirmed",
        "credit_omitted_prefix_confirmed", "source_coverage_incomplete",
        "service_assignments_not_retained")
UNAVAILABLE = {"error": "owner_statement_diagnostic_unavailable"}


def money(text):
    try:
        compact = re.sub(r"[ \u00a0\u202f]", "", text).replace(",", ".")
        if not re.fullmatch(r"[0-9]+(?:\.[0-9]{2})?", compact):
            return None
        return int(Decimal(compact) * 100)
    except (InvalidOperation, ValueError):
        return None


def safe_view(data):
    review, layout = data["review"], data["layout"]
    rec, header = review["reconciliation"], review["header"]
    sums = {"debit": 0, "credit": 0}
    currencies = set()
    for row in review["rows"]:
        amount = row["accountAmount"]
        if row.get("issues") or row["direction"] not in sums:
            raise ValueError()
        value = amount["amountMinor"]
        if type(value) is not int or value <= 0:
            raise ValueError()
        sums[row["direction"]] += value
        currencies.add(amount["currency"])
    if len(currencies) != 1 or not review["rows"]:
        raise ValueError()
    currency = next(iter(currencies))
    result = {key: False for key in KEYS}
    result["target_verified"] = True
    result["row_sums_consistent"] = all(
        sums[side] == rec.get(side + "Minor") for side in sums)
    for side in sums:
        role = side + "Turnover"
        expected = header.get(role)
        result[side + "_header_mismatch"] = bool(expected and (
            expected["currency"] != currency or expected["amountMinor"] != sums[side]))
        evidences = [e for e in header["evidence"] if e["role"] == role]
        if len(evidences) != 1 or not expected or expected["currency"] != currency:
            continue
        evidence = evidences[0]
        selected = evidence["spans"]
        if not selected or money(evidence["literal"]) != expected["amountMinor"]:
            continue
        first = min(selected, key=lambda s: (s["page"], s["block"], s["line"], s["word"]))
        # Only an immediately preceding word in the very same printed line.
        prefixes = [s for s in layout["spans"] if
                    (s["page"], s["block"], s["line"], s["word"]) ==
                    (first["page"], first["block"], first["line"], first["word"] - 1)]
        if len(prefixes) == 1 and re.fullmatch(r"[0-9]+", prefixes[0]["text"]):
            result[side + "_omitted_prefix_confirmed"] = bool(
                result[side + "_header_mismatch"] and
                money(prefixes[0]["text"] + " " + evidence["literal"]) == sums[side])
    result["source_coverage_incomplete"] = (
        "source_coverage_incomplete" in rec.get("blockers", []) and
        any(re.fullmatch(r"unmapped_spans:[0-9]+", x) for x in rec["rowCoverage"].get("reasons", [])))
    # Materialization persists Review, which has no service assignment field.
    result["service_assignments_not_retained"] = "service" not in review
    return result


def remote(target):
    if (not isinstance(target, dict) or set(target) != {"statementId", "sha256"} or
        not re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}", target["statementId"]) or
        not re.fullmatch(r"[0-9a-f]{64}", target["sha256"])):
        raise ValueError()
    found = subprocess.run([
        "docker", "ps", "-q", "--filter", "label=com.docker.compose.project=grace-production",
        "--filter", "label=com.docker.compose.service=postgres"],
        capture_output=True, text=True, check=True, timeout=5).stdout.split()
    if len(found) != 1 or not re.fullmatch(r"[0-9a-f]{12,64}", found[0]):
        raise ValueError()
    # UUID/hash strict validation above; no owner enumeration and no content read.
    sql = ("BEGIN READ ONLY; SET LOCAL statement_timeout='5s'; "
           "SELECT json_build_object('review',s.review,'layout',s.layout) "
           "FROM statement_sources s JOIN documents d "
           "ON d.id=s.document_id AND d.owner_id=s.owner_id "
           "WHERE s.id='" + target["statementId"] + "'::uuid "
           "AND d.sha256='" + target["sha256"] + "' AND d.kind='bank_statement'; ROLLBACK;")
    raw = subprocess.run(["docker", "exec", "-i", found[0], "psql", "-XqAt", "-v", "ON_ERROR_STOP=1",
                          "-U", "grace", "-d", "grace"], input=sql, capture_output=True,
                         text=True, check=True, timeout=12).stdout
    if len(raw.encode()) > 2 * 1024 * 1024:
        raise ValueError()
    return safe_view(json.loads(raw))


def dispatch():
    target = json.loads(os.environ["GRACE_STATEMENT_DIAGNOSTIC_TARGET"])
    payload = json.dumps({"script": Path(__file__).read_text(), "target": target})
    bootstrap = ("import json,sys; p=json.load(sys.stdin); "
                 "exec(compile(p['script'],'<owner-diagnostic>','exec'),"
                 "{'__name__':'__main__','TARGET':p['target']})")
    # shlex quoting is necessary for SSH's remote shell, never interpolate target.
    import shlex
    out = subprocess.run(["ssh", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
                          "-o", "ConnectTimeout=8", "grace-deploy@195.133.195.190",
                          "python3 -c " + shlex.quote(bootstrap)],
                         input=payload, capture_output=True, text=True, timeout=30)
    if out.returncode or len(out.stdout) > 4096:
        raise ValueError()
    result = json.loads(out.stdout)
    if not isinstance(result, dict) or set(result) != set(KEYS) or any(type(v) is not bool for v in result.values()):
        raise ValueError()
    return result


if __name__ == "__main__":
    try:
        result = dispatch() if sys.argv[1:] == ["--dispatch"] else remote(globals()["TARGET"])
        print(json.dumps(result, sort_keys=True, separators=(",", ":")))
    except Exception:
        print(json.dumps(UNAVAILABLE, separators=(",", ":")))
        sys.exit(1)
