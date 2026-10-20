#!/usr/bin/env python3
"""Локальная проверка: приватные поля не выводятся, SQL/сеть не вызываются."""
import importlib.util
import json
from pathlib import Path
import sys
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('s16_inspector', Path(__file__).with_name('inspect-statement-screening-s16.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
private = 'gen-SYNTHETIC-private-correlation'
report = {'version':'statement.screening.v2.gemini31-diagnostics', 'policy':{'model':'google/gemini-3.1-flash-lite','provider':'google-vertex/eu'}, 'actualMicros':None, 'calls':[], 'results':[], 'failure':{'code':'PRIVATE-error', 'httpStatus':200, 'unverifiedCorrelationId':private}}
out = module.summary(report)
assert out['expectedVersion'] and out['expectedCandidate'] and out['httpStatusIs200'] and out['privateCorrelationPresent'] and not out['billingKnown'] and not out['passed']
raw = json.dumps(out)
assert private not in raw and 'PRIVATE-error' not in raw and all(type(x) in (bool,int) for x in out.values())
report.update(actualMicros=123456,calls=[{'generationId':private,'costMicros':123456}],passed=True)
out = module.summary(report)
assert out['recordedCalls']==1 and out['billingKnown'] and out['passed'] and '123456' not in json.dumps(out) and private not in json.dumps(out)
print('Приватность чтения S1.6: PASS, ID и расход не выводятся.')
