#!/usr/bin/env python3
"""Read one fixed private screening report; emit no values, IDs or source text."""
import hashlib
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

# Only inspect already retained metadata. Never contact the gateway, retrieve
# content, read credentials, or mutate the report/ledger. No upstream strings.
try:
    def private_json(path):
        if path.is_symlink() or not path.is_file() or path.stat().st_mode & 0o077 or path.stat().st_size > 1048576:
            raise ValueError('unsafe_evidence')
        return json.loads(path.read_text())
    if not report.is_symlink() and report.is_file():
        raw = report.read_bytes()
        if len(raw) > 524288:
            raise ValueError('report_size')
        digest = hashlib.sha256(raw).hexdigest()
        inspection = private_json(root / 'statement-generation-inspection-s14-gemini31-20261020.json')
        audit = private_json(root / 'statement-reconciliation-s14-gemini31-20261020.json')
        if inspection.get('reportDigest') != digest or audit.get('reportDigest') != digest:
            raise ValueError('digest_mismatch')
        generations = inspection.get('generations', {})
        if len(generations) != 1 or generations != audit.get('generations'):
            raise ValueError('generation_mismatch')
        d = next(iter(generations.values()))
        def bounded_int(key, minimum, maximum):
            value = d.get(key)
            return type(value) is int and minimum <= value <= maximum
        responses = d.get('provider_responses')
        items = responses if isinstance(responses, list) else []
        print(json.dumps({
            'savedGenerationDiagnosisReadOnly': True,
            'boundToOriginalReport': True,
            'settlementAuditCommitted': audit.get('committed') is True,
            'gatewayFinishStop': d.get('finish_reason') == 'stop',
            'gatewayNotCancelled': d.get('cancelled') is False,
            'gatewayNonStreaming': d.get('streamed') is False,
            'gatewayPromptWithinBound': bounded_int('tokens_prompt', 1, 100000),
            'gatewayCompletionWithinBound': bounded_int('tokens_completion', 1, 16000),
            'gatewayGenerationWithinCallTimeout': bounded_int('generation_time', 0, 120000),
            'gatewayLatencyWithinCallTimeout': bounded_int('latency', 0, 120000),
            'gatewayRegionEU': d.get('data_region') == 'eu',
            'gatewayRegionMissing': d.get('data_region') is None,
            'providerResponsesPresent': bool(items),
            'providerResponseErrorPresent': any(isinstance(x, dict) and bool(x.get('error')) for x in items),
            'chatEnvelopeRetainedInMetadata': any(k in d for k in ('openrouter_metadata', 'choices')),
            'originalChatEnvelopeRetainedByHarness': False,
        }, separators=(',', ':')))
except (ValueError, OSError, TypeError, StopIteration, AttributeError):
    print('{"savedGenerationDiagnosisReadOnly":true,"savedEvidenceUnavailableOrUnmatched":true}')
