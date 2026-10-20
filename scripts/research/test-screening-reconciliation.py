#!/usr/bin/env python3
"""Offline test: no key, network, SQL or private generation ID."""
import sys
sys.dont_write_bytecode = True
import copy
import importlib.util
from pathlib import Path

spec = importlib.util.spec_from_file_location('reconcile', Path(__file__).resolve().parents[1] / 'ops-reconcile-statement-screening.py')
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
ident = 'gen-1792514513-SYNTHETIC'
uid = '11111111-1111-1111-1111-111111111111'
report = {'version': 'statement.screening.v1.gemini31-ru-roles', 'error': 'qualification_provider_failed', 'passed': False, 'actualMicros': None,
          'time': '2026-10-20T16:41:53Z', 'maximumMicros': 161700, 'calls': [], 'results': [],
          'policy': {'model': 'google/gemini-3.1-flash-lite', 'provider': 'google-vertex/eu', 'providerName': 'Google', 'inputMicrosPerMillion': 275000, 'outputMicrosPerMillion': 1650000}}
row = {'id': uid, 'owner_id': '22222222-2222-2222-2222-222222222222', 'model': report['policy']['model'], 'provider': report['policy']['provider'], 'purpose': 'statement', 'status': 'uncertain', 'generation_id': 'statement-qualification:' + uid, 'reserved_micros': 161700, 'created_at': '2026-10-20T16:41:53Z'}
generation = {'id': ident, 'model': report['policy']['model'], 'provider_name': 'Google', 'is_byok': False, 'created_at': '2026-10-20T16:41:53Z', 'total_cost': '0.0010001'}
assert m.validate(report, row, [ident], {ident: generation}) == 1001
assert m.micros('0') == 0
cases = [
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
print('Screening reconciliation offline match/mismatch/ceil/replay checks PASS')

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
print('Exact official alias/canonical mapping checks PASS')
