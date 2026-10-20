#!/usr/bin/env python3
"""Exact SHA и все пять CI jobs: положительный и отрицательные случаи."""
import copy
import importlib.util
from pathlib import Path
import sys
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('verify', Path(__file__).with_name('verify-screening-source-ci.py'))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
sha = 'a'*40
valid = {'headSha':sha, 'status':'completed', 'conclusion':'success', 'jobs':[{'name':name,'status':'completed','conclusion':'success'} for name in m.JOBS]}
assert m.verified(valid, sha)
for mutate in [lambda d:d.update(headSha='b'*40),lambda d:d.update(status='in_progress'),lambda d:d.update(conclusion='failure'),lambda d:d['jobs'].pop(),lambda d:d['jobs'].append(d['jobs'][0]),lambda d:d['jobs'][0].update(conclusion='skipped'),lambda d:d['jobs'][0].update(status='in_progress'),lambda d:d['jobs'][0].update(name='other'),lambda d:d.update(jobs=[])]:
    data = copy.deepcopy(valid)
    mutate(data)
    assert not m.verified(data,sha)
print('Exact source CI: SHA, полный состав, без skipped/failed/pending PASS')
