#!/usr/bin/env python3
"""Проверки сверки S1.6 без сети, SQL, ключей и приватного ID."""
import sys
sys.dont_write_bytecode = True
import copy
import importlib.util
from pathlib import Path

spec = importlib.util.spec_from_file_location('reconcile', Path(__file__).resolve().parents[1] / 'ops-reconcile-statement-screening-s16.py')
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
ident = 'gen-1792514513-SYNTHETIC'
uid = '11111111-1111-1111-1111-111111111111'
report = {'version': 'statement.screening.v2.gemini31-diagnostics', 'failure': {'code': 'selected_model', 'httpStatus': 200, 'unverifiedCorrelationId': ident}, 'error': 'qualification_provider_failed_selected_model', 'passed': False, 'actualMicros': None,
          'time': '2026-10-20T16:41:53Z', 'maximumMicros': 161700, 'calls': [], 'results': [],
          'policy': {'model': 'google/gemini-3.1-flash-lite', 'provider': 'google-vertex/eu', 'providerName': 'Google', 'inputMicrosPerMillion': 275000, 'outputMicrosPerMillion': 1650000}}
report['profile'] = dict(m.PROFILE)
row = {'id': uid, 'owner_id': '22222222-2222-2222-2222-222222222222', 'model': report['policy']['model'], 'provider': report['policy']['provider'], 'purpose': 'statement', 'status': 'uncertain', 'generation_id': 'statement-qualification:' + uid, 'reserved_micros': 161700, 'created_at': '2026-10-20T16:41:53Z'}
generation = {'id': ident, 'model': report['policy']['model'], 'provider_name': 'Google', 'is_byok': False, 'created_at': '2026-10-20T16:41:53Z', 'total_cost': '0.0010001'}
assert m.validate(report, row, [ident], {ident: generation}) == 1001
assert m.micros('0') == 0
assert m.micros('0.00100000000000000000000000000000000001') == 1001
cases = [
    ('wrong_profile', lambda r, u, g: r['profile'].update(Calls=4)),
    ('missing_calls', lambda r, u, g: r.pop('calls')),
    ('invalid_id', lambda r, u, g: r['failure'].update(unverifiedCorrelationId='gen-invalid?')),
    ('wrong_status', lambda r, u, g: u.update(status='reserved')),
    ('wrong_uuid', lambda r, u, g: u.update(owner_id='------------------------------------')),
    ('wrong_diagnostic', lambda r, u, g: r['failure'].update(code='other')),
    ('wrong_http', lambda r, u, g: r['failure'].update(httpStatus=201)),
    ('wrong_version', lambda r, u, g: r.update(version='other')),
    ('wrong_marker', lambda r, u, g: u.update(generation_id='other')),
    ('extreme_precision', lambda r, u, g: g.update(total_cost='1e-10000000')),
    ('nonfinite', lambda r, u, g: g.update(total_cost='NaN')),
    ('model', lambda r, u, g: g.update(model='other/model')),
    ('provider', lambda r, u, g: g.update(provider_name='other')),
    ('generation', lambda r, u, g: g.update(id='gen-other')),
    ('unknown_cost', lambda r, u, g: g.update(total_cost=None)),
    ('negative', lambda r, u, g: g.update(total_cost='-1')),
    ('overrun', lambda r, u, g: g.update(total_cost='0.053901')),
    ('byok', lambda r, u, g: g.update(is_byok=True)),
    ('generation_time', lambda r, u, g: g.update(created_at='2026-10-19T16:41:53Z')),
    ('reservation_time', lambda r, u, g: u.update(created_at='2026-10-19T16:41:53Z')),
    ('reservation', lambda r, u, g: u.update(reserved_micros=1)),
    ('scored', lambda r, u, g: r.update(results=[{}])),
    ('conflicting_settled', lambda r, u, g: u.update(status='settled', actual_micros=1)),
]
for name, mutate in cases:
    r, u, g = copy.deepcopy(report), copy.deepcopy(row), copy.deepcopy(generation)
    mutate(r, u, g)
    try:
        m.validate(r, u, [ident], {ident: g})
    except m.Stop:
        pass
    else:
        raise AssertionError('mismatch accepted: ' + name)
settled = dict(row, status='settled', actual_micros=1001)
assert m.validate(report, settled, [ident], {ident: generation}) == 1001
print('S1.6: совпадение, отказы, точный ceil и replay PASS')

canonical = dict(generation, model=m.CANONICAL_MODEL)
try:
    m.validate(report, row, [ident], {ident: canonical})
except m.Stop:
    pass
else:
    raise AssertionError('unverified alias mapping accepted')
assert m.canonical_model({'data':[{'id':m.MODEL,'canonical_slug':m.CANONICAL_MODEL}]}) == m.CANONICAL_MODEL
assert m.validate(report,row,[ident],{ident:canonical},m.CANONICAL_MODEL) == 1001
for slug in ['other/model', 'google/gemini-3.1-flash-lite-NEW']:
    try:
        m.canonical_model({'data':[{'id':m.MODEL,'canonical_slug':slug}]})
    except m.Stop:
        pass
    else:
        raise AssertionError('different canonical model accepted')
print('Точная официальная alias/canonical пара PASS')

import tempfile
import os
with tempfile.TemporaryDirectory() as directory:
    path = Path(directory) / 'audit.json'
    m.save_private(path, {'private': 'SYNTHETIC'})
    assert path.stat().st_mode & 0o777 == 0o600
    assert m.private_read(path) == b'{"private": "SYNTHETIC"}'
    os.chmod(path, 0o644)
    try:
        m.private_read(path)
    except m.Stop:
        pass
    else:
        raise AssertionError('public artifact accepted')
    link = Path(directory) / 'link'
    link.symlink_to(path)
    try:
        m.private_read(link)
    except OSError:
        pass
    else:
        raise AssertionError('symlink accepted')
audit = {'reportDigest': 'digest', 'reportName': 'fixed', 'actualMicros':1001, 'generations': {ident:generation}, 'usageBefore': row}
m.audit_matches(audit, audit)
for field in ('owner_id', 'id', 'generation_id', 'reserved_micros'):
    altered = copy.deepcopy(audit)
    altered['usageBefore'][field] = 'other'
    try:
        m.audit_matches(altered, audit)
    except m.Stop:
        pass
    else:
        raise AssertionError('conflicting audit accepted: '+field)
print('S1.6: приватность, symlink, идентичность audit и конфликтный replay PASS')

# Полный operator main: только подменённые GET и SQL, ни одного реального IO сервиса.
import contextlib
import io
from unittest.mock import patch
class Response(io.BytesIO):
    pass
class Opener:
    def open(self, request, timeout):
        url = request if isinstance(request, str) else request.full_url
        assert timeout == 15
        if not isinstance(request, str):
            assert request.get_method() == 'GET'
        urls.append(url)
        if url == 'https://openrouter.ai/api/v1/models':
            return Response(m.json.dumps({'data':[{'id':m.MODEL,'canonical_slug':m.CANONICAL_MODEL}]}).encode())
        assert url.startswith('https://openrouter.ai/api/v1/generation?id=')
        return Response(m.json.dumps({'data':generation}).encode())
with tempfile.TemporaryDirectory() as directory:
    base = Path(directory)
    report_path = base / ('statement-screening-'+m.RUN+'.json')
    marker_path = base / ('statement-screening-'+m.RUN+'.started')
    for path, raw in ((report_path,m.json.dumps(report).encode()),(marker_path,m.SOURCE.encode())):
        path.write_bytes(raw)
        path.chmod(0o600)
    originals = (report_path.read_bytes(), marker_path.read_bytes())
    state = copy.deepcopy(row)
    writes = []
    urls = []
    def fake_command(args, stdin=None):
        if args[-1] == '/run/secrets/openrouter_api_key':
            return 'SYNTHETIC-KEY'
        assert 'psql' in args and stdin is not None
        if 'json_agg' in stdin:
            return m.json.dumps([state])
        if stdin.startswith('BEGIN;'):
            assert "set_config('grace.owner_id','"+row['owner_id']+"'" in stdin
            assert 'FOR UPDATE' in stdin and 'grace_ai_settle' in stdin
            writes.append(stdin)
            state.update(status='settled', actual_micros=1001)
            return ''
        if 'row_to_json' in stdin:
            return m.json.dumps(state)
        assert 'NOT EXISTS' in stdin
        return 't' if state['status']=='settled' else 'f'
    def run(apply=False):
        with patch.object(m, 'BASE', base), patch.object(m, 'container', return_value='abcdef'), patch.object(m, 'command', side_effect=fake_command), patch.object(m.urllib.request, 'build_opener', return_value=Opener()), patch.object(m.sys, 'argv', ['operator']+(['--apply'] if apply else [])), contextlib.redirect_stdout(io.StringIO()) as out:
            m.main()
        assert ident not in out.getvalue() and 'SYNTHETIC-KEY' not in out.getvalue() and '1001' not in out.getvalue()
        return out.getvalue()
    try:
        run(True)
    except m.Stop as e:
        assert str(e) == 'read_only_inspection_required'
    else:
        raise AssertionError('apply without inspect accepted')
    assert not writes
    run()
    assert state['status']=='uncertain' and not writes
    assert '"billingReconciled": true' in run(True)
    assert len(writes)==1
    assert '"idempotentReplay": true' in run(True)
    assert len(writes)==1
    state['actual_micros'] = 1
    try:
        run(True)
    except m.Stop as e:
        assert str(e) == 'settlement_conflict'
    else:
        raise AssertionError('conflicting live replay accepted')
    assert len(writes)==1 and originals==(report_path.read_bytes(),marker_path.read_bytes())
    assert all('/generation?' in url or url.endswith('/models') for url in urls)
print('S1.6: inspect → apply → replay, запрет apply без inspect, конфликт, неизменность report/marker, GET-only PASS')
