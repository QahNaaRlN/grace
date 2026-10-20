#!/usr/bin/env python3
"""Только чтение фиксированных S1.6 report/ledger; без ключей и inference."""
import datetime as dt
import json
from pathlib import Path
import re
import subprocess
import sys


def summary(data):
    failure = data.get('failure') or {}
    correlation = failure.get('unverifiedCorrelationId')
    calls, results = data.get('calls'), data.get('results')
    return {
        'expectedVersion': data.get('version') == 'statement.screening.v2.gemini31-diagnostics',
        'expectedCandidate': data.get('policy', {}).get('model') == 'google/gemini-3.1-flash-lite' and data.get('policy', {}).get('provider') == 'google-vertex/eu',
        'billingKnown': type(data.get('actualMicros')) is int,
        'recordedCalls': len(calls) if isinstance(calls, list) and len(calls) <= 3 else -1,
        'scoredCases': len(results) if isinstance(results, list) and len(results) <= 3 else -1,
        'diagnosticPresent': bool(failure),
        'httpStatusIs200': type(failure.get('httpStatus')) is int and failure['httpStatus'] == 200,
        'privateCorrelationPresent': isinstance(correlation, str) and len(correlation) <= 128 and re.fullmatch(r'gen-[0-9A-Za-z-]+', correlation) is not None,
        'passed': data.get('passed') is True,
    }


def command(args):
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=15)
    if result.returncode:
        raise ValueError('read_failed')
    return result.stdout.strip()


def main():
    out = {'s16InspectionReadOnly': True}
    try:
        root = Path('/opt/grace/shared')
        marker = root / 'statement-screening-s16-gemini31-diag-20261021.started'
        report = root / 'statement-screening-s16-gemini31-diag-20261021.json'
        out['authorizationConsumed'] = marker.exists()
        out['reportPresent'] = report.is_file()
        if marker.is_symlink() or report.is_symlink():
            raise ValueError('unsafe_path')
        if marker.exists():
            out['expectedSource'] = marker.read_text().strip() == sys.argv[1]
        if not report.is_file():
            print(json.dumps(out));return
        if report.stat().st_mode & 0o077 or report.stat().st_size > 524288:
            raise ValueError('report_not_private')
        data = json.loads(report.read_bytes())
        out['reportPrivate'] = True
        out.update(summary(data))
        pg = command(['docker', 'ps', '-q', '--filter', 'label=com.docker.compose.project=grace-production', '--filter', 'label=com.docker.compose.service=postgres'])
        if re.fullmatch(r'[a-f0-9]+', pg) is None:
            raise ValueError('runtime_not_unique')
        stamp = dt.datetime.fromisoformat(data['time'].replace('Z', '+00:00')).isoformat()
        query = "SELECT count(*)=1, COALESCE(bool_and(status='settled'),false), COALESCE(bool_or(status='uncertain'),false) FROM ai_usage WHERE purpose='statement' AND model='google/gemini-3.1-flash-lite' AND provider='google-vertex/eu' AND reserved_micros=161700 AND generation_id LIKE 'statement-qualification:%' AND created_at BETWEEN '"+stamp+"'::timestamptz-interval '5 seconds' AND '"+stamp+"'::timestamptz+interval '5 seconds';"
        row = command(['docker', 'exec', pg, 'psql', '-X', '-U', 'grace', '-d', 'grace', '-Atc', query]).split('|')
        if len(row) != 3 or any(x not in ('t', 'f') for x in row):
            raise ValueError('unexpected_usage')
        out.update(reservationUnique=row[0]=='t', reservationSettled=row[1]=='t', reservationUncertain=row[2]=='t')
        uncertain = command(['docker', 'exec', pg, 'psql', '-X', '-U', 'grace', '-d', 'grace', '-Atc', "SELECT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain');"])
        if uncertain not in ('t', 'f'):
            raise ValueError('unexpected_ledger')
        out['sharedUncertainBlock'] = uncertain == 't'
    except (ValueError, OSError, TypeError, KeyError, IndexError, AttributeError, subprocess.SubprocessError):
        out['inspectionFailed'] = True
    print(json.dumps(out, separators=(',', ':')))


if __name__ == '__main__':
    main()
