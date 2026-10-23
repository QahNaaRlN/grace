#!/usr/bin/env python3
"""Run actual manual-workflow shell blocks with fake git/gh/ssh/scp only."""
import json
import os
from pathlib import Path
import subprocess
import tempfile

repo = Path(__file__).resolve().parents[2]
workflow = (repo / '.github/workflows/statement-row-pilot.yml').read_text()
assert 'default: false' in workflow and 'if: ${{ inputs.execute_approved_pilot }}' in workflow
assert 'workflow_dispatch:' in workflow and 'pull_request:' not in workflow
assert 'PRODUCTION_STATEMENT_OPERATOR_OWNER_ID' in workflow
assert 's119-cards-document.json' in workflow

def block(name):
    section = workflow.split('      - name: ' + name + '\n', 1)[1]
    body = section.split('        run: |\n', 1)[1]
    lines = []
    for line in body.splitlines():
        if line and not line.startswith('          '):
            break
        lines.append(line[10:])
    return '\n'.join(lines) + '\n'

sha = 'a' * 40
jobs = ['Quality gates', 'PostgreSQL integration gates', 'Build api image', 'Build web image', 'Build pdf-extractor image']
with tempfile.TemporaryDirectory(prefix='grace-s121-workflow-') as tmp:
    root = Path(tmp)
    bins = root / 'bin'; bins.mkdir()
    log = root / 'commands'
    for tool, body in {
        'git': 'test "${ANCESTOR:-true}" = true\n',
        'gh': 'case "$*" in run\\ list*) echo 123 ;; run\\ view*) cat "$CI_FIXTURE" ;; *) exit 9 ;; esac\n',
        'scp': 'echo scp >> "$COMMANDS"\n',
        'ssh': '''echo ssh >> "$COMMANDS"
case "$*" in *'mkdir '*) exit 0 ;; esac
case "$FAKE_SSH" in private) echo '{"PRIVATE":"model raw"}' ;; loss) echo PRIVATE >&2; exit 255 ;; refusal) echo '{"error":"row_pilot_operator_owner_unavailable"}';exit 1 ;; *) exit 99 ;; esac
''',
    }.items():
        p = bins / tool; p.write_text('#!/bin/sh\nset -eu\n' + body); p.chmod(0o755)
    bundle = root / ('grace-statement-row-s121-' + sha); bundle.mkdir()
    subprocess.run(['go', 'build', '-ldflags', '-X main.sourceSHA=' + sha, '-o', str(bundle / 'grace-row-pilot'), './cmd/statement-row-pilot'], cwd=repo, check=True)
    ci = root / 'ci.json'
    base = dict(os.environ, PATH=str(bins) + ':' + os.environ['PATH'], RUNNER_TEMP=tmp, GITHUB_SHA=sha, AUTHORIZED_SHA=sha, GITHUB_RUN_ATTEMPT='1', OPERATOR_OWNER_ID='00000000-0000-0000-0000-000000000001', COMMANDS=str(log), CI_FIXTURE=str(ci), FAKE_SSH='private')
    for mode in ['pass', 'wrong_source', 'retry', 'not_main', 'skipped_ci', 'wrong_ci_source']:
        data = dict(headSha=sha, status='completed', conclusion='success', jobs=[dict(name=n, status='completed', conclusion='success') for n in jobs])
        env = base.copy()
        if mode == 'wrong_source': env['AUTHORIZED_SHA'] = 'b' * 40
        if mode == 'retry': env['GITHUB_RUN_ATTEMPT'] = '2'
        if mode == 'not_main': env['ANCESTOR'] = 'false'
        if mode == 'skipped_ci': data['jobs'][0]['conclusion'] = 'skipped'
        if mode == 'wrong_ci_source': data['headSha'] = 'b' * 40
        ci.write_text(json.dumps(data))
        result = subprocess.run(['bash', '-e', '-o', 'pipefail', '-c', block('Verify exact authorized source and CI 5/5')], cwd=repo, env=env, capture_output=True)
        assert (result.returncode == 0) == (mode == 'pass'), mode
        assert not log.exists(), mode
    for mode in ['private', 'loss', 'refusal', 'retry', 'wrong_source', 'owner', 'not_main']:
        log.unlink(missing_ok=True)
        resultfile = root / 's121-row-summary.json'; resultfile.unlink(missing_ok=True)
        env = base.copy(); env['FAKE_SSH'] = mode
        if mode == 'retry': env['GITHUB_RUN_ATTEMPT'] = '2'
        if mode == 'wrong_source': env['AUTHORIZED_SHA'] = 'b' * 40
        if mode == 'owner': env['OPERATOR_OWNER_ID'] = "invalid'PRIVATE"
        if mode == 'not_main': env['ANCESTOR'] = 'false'
        result = subprocess.run(['bash', '-e', '-o', 'pipefail', '-c', block('Execute approved pilot once')], cwd=repo, env=env, capture_output=True)
        assert result.returncode != 0, mode
        assert b'PRIVATE' not in result.stdout + result.stderr, mode
        if mode in ['private', 'loss', 'refusal']:
            data = json.loads(resultfile.read_text())
            assert data['decision'] == 'INCONCLUSIVE' and data['replayVerified'] is False, mode
            assert data['recordedCalls'] is None and data['actualMicros'] is None and data['settlementRecorded'] is None, mode
            assert log.read_text().splitlines() == ['ssh', 'scp', 'ssh'], mode
        else:
            assert not log.exists(), mode
print('S121 workflow guards, fake SSH privacy/loss/refusal: PASS')
