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
assert set(conditions) == {'screen-statement', 'screen-statement-s16', 'screen-statement-s18', 'inspect-statement', 'reconcile-statement-s16', 'diagnose-statement-s19', 'compare-statement-s111', 'compare-statement-s114', 'compare-statement-s117', 'diagnose-statement-s112', 'diagnose-statement-s115', 'diagnose-statement-s118', 'deploy'}

compiled = {}
for name, condition in conditions.items():
    expression = re.sub(r'inputs\.([a-z0-9_]+)', lambda m: "values["+repr(m[1])+"]", condition)
    expression = expression.replace('&&', ' and ').replace('||', ' or ').replace('!', ' not ')
    assert re.fullmatch(r"[a-zA-Z0-9_' ()=\[\]]+", expression)
    compiled[name] = compile(expression.strip(), '<workflow condition>', 'eval')

def selected(values):
    return [name for name, expression in compiled.items()
            if eval(expression, {'__builtins__': {}}, {'values': values})]

count = 0
for old, new, inspect, reconcile, action, recovery, identity, diagnose, comparison, diagnosis, observed, saved, alias, quality in itertools.product([False, True], [False, True], [False, True], ['', 'none', 'inspect', 'apply'], ['', 'preserve', 'enable', 'disable'], ['', 'none', 'inspect', 'apply'], [False, True], [False, True], [False, True], [False, True], [False, True], [False, True], [False, True], [False, True]):
    values = dict(statement_screening_only=old, statement_screening_s16_only=new, inspect_statement_only=inspect, reconcile_screening_usage=reconcile, statement_action=action, reconcile_screening_s16_usage=recovery, statement_screening_s18_only=identity, diagnose_statement_s19_only=diagnose, statement_comparison_s111_only=comparison, diagnose_statement_s112_only=diagnosis, statement_comparison_s114_only=observed, diagnose_statement_s115_only=saved, statement_comparison_s117_only=alias, diagnose_statement_s118_only=quality)
    active = selected(values)
    if quality:
        expected=not any((old,new,inspect,identity,diagnose,comparison,diagnosis,observed,saved,alias)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
        assert active == (['diagnose-statement-s118'] if expected else []), (values,active)
        count+=1
        continue
    if alias:
        expected=not any((old,new,inspect,identity,diagnose,comparison,diagnosis,observed,saved)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
        assert active == (['compare-statement-s117'] if expected else []), (values,active)
        count+=1
        continue
    if saved:
        expected=not any((old,new,inspect,identity,diagnose,comparison,diagnosis,observed)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
        assert active == (['diagnose-statement-s115'] if expected else []), (values,active)
        count+=1
        continue
    if observed:
        expected=not any((old,new,inspect,identity,diagnose,comparison,diagnosis)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
        assert active == (['compare-statement-s114'] if expected else []), (values,active)
        count+=1
        continue
    if diagnosis:
        expected=not any((old,new,inspect,identity,diagnose,comparison)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
        assert active == (['diagnose-statement-s112'] if expected else []), (values,active)
        count+=1
        continue
    if comparison:
        expected=not any((old,new,inspect,identity,diagnose)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
        assert active == (['compare-statement-s111'] if expected else []), (values,active)
        count+=1
        continue
    assert len(active) <= 1, (values, active)
    expected_new = not diagnose and not identity and recovery in ('', 'none') and new and not old and not inspect and reconcile in ('', 'none') and action in ('', 'preserve')
    assert ('screen-statement-s16' in active) == expected_new, (values, active)
    expected_identity = not diagnose and identity and not any((old, new, inspect)) and reconcile in ('', 'none') and recovery in ('', 'none') and action in ('', 'preserve')
    assert ('screen-statement-s18' in active) == expected_identity, (values, active)
    if identity:
        assert active == (['screen-statement-s18'] if expected_identity else []), (values, active)
    if new:
        assert active == (['screen-statement-s16'] if expected_new else []), (values, active)
    expected_recovery = not diagnose and not identity and recovery in ('inspect', 'apply') and not any((old, new, inspect)) and reconcile in ('', 'none') and action in ('', 'preserve')
    assert ('reconcile-statement-s16' in active) == expected_recovery, (values, active)
    if recovery in ('inspect', 'apply'):
        assert active == (['reconcile-statement-s16'] if expected_recovery else []), (values, active)
    expected_diagnose = diagnose and not any((old,new,inspect,identity)) and reconcile in ('','none') and recovery in ('','none') and action in ('','preserve')
    assert ('diagnose-statement-s19' in active) == expected_diagnose, (values,active)
    if diagnose:
        assert active == (['diagnose-statement-s19'] if expected_diagnose else []), (values,active)
    if not any((old, new, inspect, identity, diagnose)) and reconcile in ('', 'none') and recovery in ('', 'none'):
        assert active == ['deploy']
    count += 1
for key in ('statement_screening_only', 'statement_screening_s16_only', 'statement_screening_s18_only', 'diagnose_statement_s19_only', 'statement_comparison_s111_only', 'diagnose_statement_s112_only', 'statement_comparison_s114_only', 'diagnose_statement_s115_only', 'statement_comparison_s117_only', 'diagnose_statement_s118_only'):
    # Описание имеет восемь пробелов отступа, граница входного параметра — шесть.
    block = re.search(r'      '+key+r':\n(.*?)(?=^      [a-z_]+:|^permissions:)', source, re.M | re.S)[1]
    assert 'default: false' in block and 'type: boolean' in block
assert 'cp deployments/production/statement-screening-s16.sh' in source
s18 = source.split('  screen-statement-s18:\n',1)[1].split('  compare-statement-s111:\n',1)[0]
assert 'cp deployments/production/statement-screening-s18.sh' in s18
assert s18.count('python3 scripts/research/verify-screening-source-ci.py') == 2
assert 'gh run view 39502669662 --json headSha,status,conclusion,jobs' in s18
assert 'inspect-statement-screening-s18.py' in s18
assert '--commit="$GITHUB_SHA" --status=success' in source
assert 'sha256sum grace-statement-admin *.pdf SOURCE_SHA run.sh > SHA256SUMS' in source
print(f'Взаимоисключение режимов S1.9: {count} сочетаний PASS; inference по умолчанию выключен.')

s19 = source.split('  diagnose-statement-s19:\n',1)[1].split('  inspect-statement:\n',1)[0]
assert 'diagnose-statement-s19.py' in s19 and 'replay-statement-s19.sh' in s19
assert all(x not in s19 for x in ('scp ', 'screening-s18.sh', 'openrouter', 'reconcile-', 'GRACE_AI_'))
assert source.count('group: production') == 1

comparison_job=source.split('  compare-statement-s111:\n',1)[1].split('  compare-statement-s114:\n',1)[0]
assert comparison_job.count('test "$AUTHORIZED_SHA" = "$GITHUB_SHA"') == 2
assert 'verify-screening-source-ci.py "$GITHUB_SHA"' in comparison_job
assert 'inspect-statement-comparison-s111.py' in comparison_job
assert 'statement-comparison-s111.sh' in comparison_job
assert 'SHA256SUMS' in comparison_job and 'inspect.py' in comparison_job
print('S1.11 exact-source authorization, exclusive/default-off mode and complete bundle guards PASS')

s112=source.split('  diagnose-statement-s112:\n',1)[1].split('  diagnose-statement-s118:\n',1)[0]
assert 'run-s112-diagnosis.py' in s112 and 's112-verifier" replay' in s112
assert all(x not in s112 for x in ('scp ', 'docker exec', 'statement-comparison-s111.sh', 'openrouter', 'GRACE_AI_'))
assert s112.count('test "$DIAGNOSTIC_SHA" = "$GITHUB_SHA"')==2
assert 'verify-screening-source-ci.py "$GITHUB_SHA"' in s112
assert 's112-bootstrap' not in source  # executed only inside the fixed-host wrapper
print('S1.12 default-off, exhaustive exclusivity and exact-source gates PASS')

s114=source.split('  compare-statement-s114:\n',1)[1].split('  compare-statement-s117:\n',1)[0]
assert s114.count('test "$AUTHORIZED_SHA" = "$GITHUB_SHA"')==2
assert 'comparison_s114_authorized_sha' in s114 and 'inputs.comparison_authorized_sha' not in s114
for name in ('statement-comparison-s114.sh','inspect-statement-comparison-s114.py','inspect-statement-comparison-s111.py','grace-s114-replay','verify-screening-source-ci.py'):
    assert name in s114
assert 'sha256sum grace-statement-admin grace-s114-replay *.pdf SOURCE_SHA run.sh inspect.py inspect-statement-comparison-s111.py > SHA256SUMS' in s114
assert '^[0-9a-f]{40}$' in s114
print('S1.14 exact-source/full-manifest/default-off/mixed-input isolation PASS')

s115=source.split('  diagnose-statement-s115:\n',1)[1].split('  diagnose-statement-s112:\n',1)[0]
assert s115.count('test "$DIAGNOSTIC_SHA" = "$GITHUB_SHA"')==2
assert s115.count('verify-screening-source-ci.py "$GITHUB_SHA"')==2
assert 'sha256sum verifier context.json SOURCE_SHA run-s115-diagnosis.py s115-bootstrap.py > SHA256SUMS' in s115
assert s115.index('sha256sum -c SHA256SUMS') < s115.index('python3 "$bundle/run-s115-diagnosis.py"')
assert all(x not in s115 for x in ('scp ', 'docker exec', 'statement-comparison-s114.sh', 'openrouter', 'GRACE_AI_'))
print('S1.15 exhaustive mode isolation, exact source, five-job CI and complete manifest PASS')

# A new mode has its own exact source and complete manifest; no neighbour action.
s117=source.split('  compare-statement-s117:\n',1)[1].split('  diagnose-statement-s19:\n',1)[0]
assert s117.count('test "$AUTHORIZED_SHA" = "$GITHUB_SHA"')==2
assert 'comparison_s117_authorized_sha' in s117
assert 'inputs.comparison_s114_authorized_sha' not in s117
assert './scripts/research/s117-replay' in s117
assert 'sha256sum grace-statement-admin grace-s117-replay *.pdf SOURCE_SHA run.sh inspect.py inspect-statement-comparison-s111.py > SHA256SUMS' in s117
assert all(x not in s117 for x in ('statement_action.sh', 'docker compose up', 'reconcile-', 'diagnose-', 'GRACE_AI_'))
print('S1.17 exact-source/default-off/complete-manifest/exhaustive mixed-mode isolation PASS')

s118=source.split('  diagnose-statement-s118:\n',1)[1]
assert s118.count('test "$DIAGNOSTIC_SHA" = "$GITHUB_SHA"')==2
assert s118.count('verify-screening-source-ci.py "$GITHUB_SHA"')==2
assert 'main.verifierSource=$GITHUB_SHA' in s118
assert 'sha256sum verifier context.json SOURCE_SHA run-s118-diagnosis.py s118-bootstrap.py > SHA256SUMS' in s118
assert s118.index('sha256sum -c SHA256SUMS') < s118.index('python3 "$bundle/run-s118-diagnosis.py"')
assert '"$bundle/context.json" "$GITHUB_SHA"' in s118
artifact=s118.split('uses: actions/upload-artifact@v7',1)[1]
assert 's118-verified.json' not in artifact and 's118-result.json' not in artifact
assert all(x not in s118 for x in ('scp ', 'docker exec', 'statement-comparison-s117.sh','openrouter','GRACE_AI_'))
print('S1.18 exclusive/default-off/full manifest/compiled source/no witness artifact PASS')
