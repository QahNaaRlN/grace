"""Test-only frozen v2 wire contract for historical exact-source inspectors.

The old inspectors remain unchanged. Current Go rejects old profiles and the
frozen inspectors reject the new profile. Use a separate v2 compiler overlay
for their positive plumbing/corruption tests, never relabel or rehash reports.
"""
import hashlib
import json
from pathlib import Path
import re


def frozen_v2_overlay(directory):
    root = Path(__file__).resolve().parents[2]
    schema = root / 'internal/statementextract/testdata/s116-v2-schema.go.txt'
    # Exact public repository source snapshot, not an implementation of v3.
    assert hashlib.sha256(schema.read_bytes()).hexdigest() == '000763493446237c731f305e7491ed4f57d776a6c149e6897f3bf679fce91600'
    source = root / 'internal/statementextract/plan.go'
    plan, count = re.subn(r'PlannerVersion\s*=\s*"statement-document-atom.v3.alias-enum"',
                         'PlannerVersion = "statement-document-atom.v2.lexical-grounding"',
                         source.read_text())
    assert count == 1
    old_plan = directory / 'v2-plan.go'
    old_plan.write_text(plan)
    overlay = directory / 'v2-overlay.json'
    overlay.write_text(json.dumps({'Replace': {
        str(source): str(old_plan),
        str(root / 'internal/statementextract/schema.go'): str(schema),
    }}))
    return ['-overlay', str(overlay)]
