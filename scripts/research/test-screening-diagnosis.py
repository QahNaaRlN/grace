#!/usr/bin/env python3
"""Offline disclosure and source-binding checks for the read-only inspector."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile

source = (Path(__file__).with_name('inspect-statement-screening.py')).read_text()
with tempfile.TemporaryDirectory(prefix='grace-diagnosis-') as directory:
    root = Path(directory)
    executable = source.replace("Path('/opt/grace/shared')", 'Path('+repr(directory)+')')
    report = root / 'statement-screening-s14-gemini31-20261020.json'
    report.write_text(json.dumps({'version':'statement.screening.v1.gemini31-ru-roles','policy':{},'calls':[],'results':[],'error':'qualification_provider_failed'}))
    report.chmod(0o600)
    digest = hashlib.sha256(report.read_bytes()).hexdigest()
    private = 'SYNTHETIC-PRIVATE-NOT-FOR-STDOUT'
    data = {'id':private, 'total_cost':123.456789,'finish_reason':'stop','cancelled':False,'streamed':False,'tokens_prompt':100,'tokens_completion':10,'generation_time':2000,'latency':100,'data_region':'eu','provider_responses':[{'error':private}]}
    inspection = root / 'statement-generation-inspection-s14-gemini31-20261020.json'
    audit = root / 'statement-reconciliation-s14-gemini31-20261020.json'
    def write_audit(value):
        audit.write_text(json.dumps(value));audit.chmod(0o600)
    inspection.write_text(json.dumps({'reportDigest':digest,'generations':{private:data}}));inspection.chmod(0o600)
    good = {'reportDigest':digest,'committed':True,'generations':{private:data}}
    def run():
        p = subprocess.run([sys.executable,'-c',executable],text=True,capture_output=True,check=True)
        assert not p.stderr
        assert private not in p.stdout and '123.456789' not in p.stdout
        rows = [json.loads(line) for line in p.stdout.splitlines()]
        assert all(type(v) in (bool,int) for row in rows for v in row.values())
        return rows[-1]
    write_audit(good)
    result = run()
    assert result['boundToOriginalReport'] and result['settlementAuditCommitted']
    assert result['gatewayFinishStop'] and result['providerResponseErrorPresent']
    assert not result['chatEnvelopeRetainedInMetadata']
    write_audit(dict(good,reportDigest='different'))
    assert run()['savedEvidenceUnavailableOrUnmatched']
    write_audit(good);inspection.chmod(0o644)
    assert run()['savedEvidenceUnavailableOrUnmatched']
    inspection.chmod(0o600)
    saved = inspection.with_suffix('.saved');inspection.rename(saved);inspection.symlink_to(saved)
    assert run()['savedEvidenceUnavailableOrUnmatched']
print('Read-only screening diagnosis: private output, digest, permissions and symlink checks PASS')
