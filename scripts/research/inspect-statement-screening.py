#!/usr/bin/env python3
"""Read one fixed private screening report; emit no values, IDs or source text."""
import json
from pathlib import Path

root = Path('/opt/grace/shared')
report = root / 'statement-screening-s14-gemini31-20261020.json'
marker = root / 'statement-screening-s14-gemini31-20261020.started'
out = {'screeningInspectionReadOnly': True, 'authorizationConsumed': marker.exists(),
       'reportPresent': report.is_file()}
if report.is_file():
    try:
        if report.stat().st_size > 524288:
            raise ValueError('report_size')
        data = json.loads(report.read_text())
        calls = data.get('calls')
        results = data.get('results')
        out.update({
            'reportPrivate': report.stat().st_mode & 0o077 == 0,
            'expectedReportVersion': data.get('version') == 'statement.screening.v1.gemini31-ru-roles',
            'expectedCandidate': data.get('policy', {}).get('model') == 'google/gemini-3.1-flash-lite' and data.get('policy', {}).get('provider') == 'google-vertex/eu',
            'billingKnown': isinstance(data.get('actualMicros'), int),
            'recordedCalls': len(calls) if isinstance(calls, list) and len(calls) <= 3 else -1,
            'scoredCases': len(results) if isinstance(results, list) and len(results) <= 3 else -1,
            'providerFailureBeforeScoring': data.get('error') == 'qualification_provider_failed' and results == [],
            'passed': data.get('passed') is True,
        })
    except (ValueError, OSError, TypeError):
        out['reportInspectionFailed'] = True
print(json.dumps(out, separators=(',', ':')))
