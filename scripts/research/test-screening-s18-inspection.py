#!/usr/bin/env python3
"""S1.8 inspector: shape/proof/privacy и read-only ledger без внешних сервисов."""
import contextlib
import copy
import datetime as dt
import importlib.util
import io
import json
import os
from pathlib import Path
import sys
import tempfile
from unittest.mock import patch
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('inspect18',Path(__file__).with_name('inspect-statement-screening-s18.py'))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
now = dt.datetime.now(dt.timezone.utc)
stamp = (now-dt.timedelta(seconds=2)).isoformat()
proof = {'alias':m.MODEL,'canonical':m.CANONICAL,'provider':m.POLICY['provider'],'catalogSha256':'a'*64,'verifiedAt':stamp}
route = {'proof':proof,'observedAt':stamp,'selectedModelClass':'pinned_canonical'}
private = 'gen-SYNTHETIC-private-inspection'
report = {'version':'statement.screening.v3.gemini31-identity','policy':m.POLICY,'profile':m.PROFILE,'maximumMicros':161700,'actualMicros':None,'calls':[],'results':[],'time':stamp,'screeningFinishedAt':(now-dt.timedelta(seconds=1)).isoformat(),'passed':False,'failure':{'code':'PRIVATE-error','unverifiedCorrelationId':private,'route':{'selectedModelClass':'PRIVATE-model'}}}
out = m.summary(report,now)
assert out['reportShapeVerified'] and not out['billingKnown'] and out['recordedCalls']==0 and out['selectedModelClass']=='not_observed'
assert private not in json.dumps(out) and 'PRIVATE' not in json.dumps(out)
good = copy.deepcopy(report)
good.update(passed=True, failure=None,actualMicros=300,calls=[{'costMicros':100,'screeningRoute':copy.deepcopy(route)} for _ in range(3)],results=[{'name':name,'passed':True,'checks':checks,'correct':checks} for name,checks in [('columns',19),('cards',20),('turnover-roles-ru',23)]])
assert m.summary(good,now)['reportedPassed']
for mutate in [lambda d:d.update(version='old'),lambda d:d['policy'].update(model='other'),lambda d:d['profile'].update(Input=1),lambda d:d.update(actualMicros=1),lambda d:d['calls'][0]['screeningRoute']['proof'].update(canonical='other'),lambda d:d['calls'][0]['screeningRoute']['proof'].update(catalogSha256='PRIVATE'),lambda d:d['calls'][0]['screeningRoute']['proof'].update(verifiedAt=(now-dt.timedelta(hours=1)).isoformat()),lambda d:d['calls'][0]['screeningRoute'].update(observedAt=(now+dt.timedelta(hours=1)).isoformat()),lambda d:d['results'][0].update(correct=0),lambda d:d['calls'][0].update(costMicros=53901)]:
    data = copy.deepcopy(good)
    mutate(data)
    try: m.summary(data,now)
    except ValueError: pass
    else: raise AssertionError('invalid report accepted')
uid = '11111111-1111-1111-1111-111111111111'
row = {'id':uid,'owner_id':'22222222-2222-2222-2222-222222222222','purpose':'statement','model':m.MODEL,'provider':m.POLICY['provider'],'reserved_micros':161700,'generation_id':'statement-qualification:'+uid,'created_at':stamp,'status':'uncertain','actual_micros':None}
assert m.usage_summary(row,report)['reservationUncertain']
settled = dict(row,status='settled',actual_micros=300)
assert m.usage_summary(settled,good)['reservationSettled']
for change in [{'id':'not-uuid'},{'owner_id':'invalid'},{'generation_id':'other'},{'reserved_micros':1},{'actual_micros':1},{'created_at':(now-dt.timedelta(hours=1)).isoformat()}]:
    try:m.usage_summary(dict(settled,**change),good)
    except ValueError:pass
    else:raise AssertionError('other/conflicting reservation accepted')
with tempfile.TemporaryDirectory() as directory:
    root = Path(directory)
    sha = 'b'*40
    for suffix,raw in [('.started',sha.encode()),('.json',json.dumps(report).encode())]:
        path = root / ('statement-screening-'+m.RUN+suffix)
        path.write_bytes(raw);path.chmod(0o600)
    before = {p.name:p.read_bytes() for p in root.iterdir()}
    queries = []
    def command(args):
        if args[1:3]==['ps','-q']:return 'abcdef'
        query = args[-1]
        assert query.startswith('BEGIN READ ONLY; SELECT ') and query.endswith('; COMMIT;')
        assert not any(token in query for token in ('UPDATE ','INSERT ','DELETE ','grace_ai_settle'))
        queries.append(query)
        return json.dumps([row]) if 'json_agg' in query else 't'
    with patch.object(m,'ROOT',root),patch.object(m,'command',side_effect=command),patch.object(m.sys,'argv',['inspect',sha]),contextlib.redirect_stdout(io.StringIO()) as output:
        assert m.main()==0
    assert len(queries)==2 and '"reservationUncertain":true' in output.getvalue()
    assert all(s not in output.getvalue() for s in (private,uid,'PRIVATE','161700'))
    assert before=={p.name:p.read_bytes() for p in root.iterdir()}
    path = root / ('statement-screening-'+m.RUN+'.json')
    path.chmod(0o644)
    try:m.private_read(path,524288)
    except ValueError:pass
    else:raise AssertionError('public report accepted')
    link = root / 'symlink';link.symlink_to(path)
    try:m.private_read(link,524288)
    except OSError:pass
    else:raise AssertionError('symlink accepted')
print('S1.8 inspector: version/profile/proof, exact ledger, privacy, read-only и immutable artifacts PASS')
