#!/usr/bin/env python3
"""Проверка реальных условий workflow: без API, SSH, ключей и inference."""
import itertools
from pathlib import Path
import re

source = (Path(__file__).resolve().parents[2] / '.github/workflows/deploy-production.yml').read_text()
job_source = source.split('\njobs:\n', 1)[1]
conditions = {}
for match in re.finditer(r'^  ([a-z][a-z0-9-]*):\n(.*?)(?=^  [a-z][a-z0-9-]*:\n|\Z)', job_source, re.M | re.S):
    condition = re.search(r'^    if: \$\{\{ (.*?) \}\}\s*$', match[2], re.M)
    if condition:
        conditions[match[1]] = condition[1]
assert set(conditions) == {'screen-statement', 'screen-statement-s16', 'inspect-statement', 'reconcile-statement-s16', 'deploy'}

def selected(values):
    active = []
    for name, condition in conditions.items():
        expression = re.sub(r'inputs\.([a-z0-9_]+)', lambda m: repr(values[m[1]]), condition)
        expression = expression.replace('&&', ' and ').replace('||', ' or ').replace('!', ' not ')
        assert re.fullmatch(r"[a-zA-Z0-9_' ()=]+", expression)
        if eval(expression, {'__builtins__': {}}, {}):
            active.append(name)
    return active

count = 0
for old, new, inspect, reconcile, action, recovery in itertools.product([False, True], [False, True], [False, True], ['', 'none', 'inspect', 'apply'], ['', 'preserve', 'enable', 'disable'], ['', 'none', 'inspect', 'apply']):
    values = dict(statement_screening_only=old, statement_screening_s16_only=new, inspect_statement_only=inspect, reconcile_screening_usage=reconcile, statement_action=action, reconcile_screening_s16_usage=recovery)
    active = selected(values)
    assert len(active) <= 1, (values, active)
    expected_new = recovery in ('', 'none') and new and not old and not inspect and reconcile in ('', 'none') and action in ('', 'preserve')
    assert ('screen-statement-s16' in active) == expected_new, (values, active)
    if new:
        assert active == (['screen-statement-s16'] if expected_new else []), (values, active)
    expected_recovery = recovery in ('inspect', 'apply') and not any((old, new, inspect)) and reconcile in ('', 'none') and action in ('', 'preserve')
    assert ('reconcile-statement-s16' in active) == expected_recovery, (values, active)
    if recovery in ('inspect', 'apply'):
        assert active == (['reconcile-statement-s16'] if expected_recovery else []), (values, active)
    if not any((old, new, inspect)) and reconcile in ('', 'none') and recovery in ('', 'none'):
        assert active == ['deploy']
    count += 1
for key in ('statement_screening_only', 'statement_screening_s16_only'):
    # Описание имеет восемь пробелов отступа, граница входного параметра — шесть.
    block = re.search(r'      '+key+r':\n(.*?)(?=^      [a-z_]+:|^permissions:)', source, re.M | re.S)[1]
    assert 'default: false' in block and 'type: boolean' in block
assert 'cp deployments/production/statement-screening-s16.sh' in source
assert '--commit="$GITHUB_SHA" --status=success' in source
assert 'sha256sum grace-statement-admin *.pdf SOURCE_SHA run.sh > SHA256SUMS' in source
print(f'Взаимоисключение режимов S1.6: {count} сочетаний PASS; inference по умолчанию выключен.')
