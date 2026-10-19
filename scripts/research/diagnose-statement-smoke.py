#!/usr/bin/env python3
"""Read-only fixed synthetic report diagnosis. Never export usage or text."""
import json
import re
from pathlib import Path

LOCATOR = re.compile(r"p[0-9]+-b[0-9]+-l[0-9]+-w[0-9]+\Z")
FIELDS = ("operationDate", "bookingDate", "accountAmount", "accountCurrency",
          "transactionAmount", "transactionCurrency", "description", "directionEvidence")
HEADER = ("openingBookDate", "closingBookDate", "openingBookBalance", "closingBookBalance",
          "availableBalanceObserved", "debitTurnover", "creditTurnover", "currency")
ROLES = ("openingBookBalance", "closingBookBalance", "availableBalanceObserved", "debitTurnover", "creditTurnover")


def references(value):
    if not isinstance(value, list):
        return []
    return [item for item in value[:2000] if isinstance(item, str) and LOCATOR.fullmatch(item)]


def safe_view(report):
    if not isinstance(report, dict) or report.get("version") not in (
            "statement.synthetic.v1", "statement.synthetic.v2.document-alias"):
        raise ValueError("unsupported_synthetic_report")
    traces = []
    for trace in report.get("traces", [])[:2]:
        if trace.get("name") not in ("columns", "cards"):
            continue
        parts = []
        for part in trace.get("parts", [])[:4]:
            header = part.get("header", {})
            labels = header.get("labels", {})
            parts.append({
                "rows": [{field: references(row.get(field)) for field in FIELDS}
                         for row in part.get("rows", [])[:5000]],
                "header": {field: references(header.get(field)) for field in HEADER},
                "labels": {role: references(labels.get(role)) for role in ROLES},
                "service": [references(group.get("spans")) for group in part.get("service", [])[:2000]],
            })
        traces.append({"name": trace["name"], "parts": parts})
    return {"syntheticFieldLocators": traces}


def main():
    # No arbitrary file argument or document input. Operator reports stay on VPS.
    paths = sorted(Path("/opt/grace/shared").glob("statement-qualification-*.json"))
    if not paths:
        raise ValueError("synthetic_report_missing")
    path = paths[-1]
    if path.stat().st_size > 1048576:
        raise ValueError("synthetic_report_oversize")
    report = json.loads(path.read_text())
    print(json.dumps(safe_view(report), separators=(",", ":")))


if __name__ == "__main__":
    try:
        main()
    except Exception:
        print('{"error":"synthetic_locator_diagnostic_unavailable"}')
        raise SystemExit(1)
