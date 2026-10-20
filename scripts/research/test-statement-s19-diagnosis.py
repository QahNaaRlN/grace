#!/usr/bin/env python3
"""S1.9: privacy, exact source, strict schema, immutable read and lock."""
import contextlib
import copy
import datetime as dt
import importlib.util
import io
import json
from pathlib import Path
import tempfile
from unittest.mock import patch
import sys
sys.dont_write_bytecode=True
spec=importlib.util.spec_from_file_location('s19',Path(__file__).with_name('diagnose-statement-s19.py'))
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
now=dt.datetime.now(dt.timezone.utc);stamp=(now-dt.timedelta(seconds=2)).isoformat()
sentinel='PRIVATE-GENERATION-COST-TEXT'
empty={k:[] for k in m.HEADER};empty['labels']={}
row=dict({k:[] for k in m.ROW},direction='')
report={'version':'statement.screening.v3.gemini31-identity','policy':m.POLICY,'profile':m.PROFILE,'maximumMicros':161700,'time':stamp,'screeningFinishedAt':(now-dt.timedelta(seconds=1)).isoformat(),'passed':False,'error':'qualification_quality_failed','calls':[{'generationId':sentinel,'costMicros':12345,'screeningRoute':{'selectedModelClass':'pinned_canonical','observedAt':stamp,'proof':{'alias':m.MODEL,'canonical':'google/gemini-3.1-flash-lite-20260507','provider':m.POLICY['provider'],'catalogSha256':'a'*64,'verifiedAt':stamp}}}],'corpus':[{'name':n,'fixtureHash':h,'documentHash':'a'*64,'planHash':'b'*64,'inputBytes':1000} for n,h in m.FIXTURES.items()],'results':[{'name':'columns','checks':19,'correct':17,'rows':2,'passed':False,'codes':['record_description','blockers'],'validation':['debitTurnover:field_label_required',sentinel],'coverage':[]}],'traces':[{'name':'columns','parts':[{'rows':[dict(row),dict(row)],'header':empty,'service':[{'spans':['p1-b1-l1-w1'],'reason':sentinel}]}]}]}
out=m.project(report,now)
assert out['result']['validation']==['debitTurnover:field_label_required','unknown']
assert sentinel not in json.dumps(out) and '12345' not in json.dumps(out)
assert out['parts'][0]['service'][0]['reason']=='Reviewed fixed synthetic service grouping'
for mutate in [lambda d:d.update(version='old'),lambda d:d['profile'].update(Prompt=sentinel),lambda d:d['policy'].update(model=sentinel),lambda d:d['traces'][0]['parts'][0]['rows'][0].update(direction=sentinel),lambda d:d['traces'][0]['parts'][0]['rows'][0].update(description=[sentinel]),lambda d:d['traces'][0]['parts'][0]['header'].update(labels={sentinel:[]}),lambda d:d['traces'][0]['parts'][0]['service'][0].update(reason=''),lambda d:d['traces'][0]['parts'][0]['service'][0].update(reason='я'*1001),lambda d:d['corpus'][0].update(fixtureHash='f'*64),lambda d:d['corpus'][0].update(documentHash=sentinel),lambda d:d['calls'][0]['screeningRoute']['proof'].update(canonical=sentinel),lambda d:d['calls'][0]['screeningRoute']['proof'].update(verifiedAt=(now-dt.timedelta(days=1)).isoformat()),lambda d:d.update(screeningFinishedAt=(now+dt.timedelta(hours=1)).isoformat()),lambda d:d['traces'][0]['parts'].append({}),lambda d:d['traces'][0]['parts'][0].update(extra=sentinel),lambda d:d['results'][0].update(checks=20)]:
    data=copy.deepcopy(report);mutate(data)
    try:m.project(data,now)
    except (ValueError,TypeError):pass
    else:raise AssertionError('invalid accepted')
for raw in ('{"x":1,"x":2}','{} {}','NaN','['*22+'0'+']'*22):
    try:m.decode(raw)
    except (ValueError,TypeError):pass
    else:raise AssertionError('invalid JSON accepted')
# Duplicates/order must remain visible to Go validation, not cleaned away.
value=['p1-b1-l1-w2','p1-b1-l1-w1','p1-b1-l1-w1']
assert m.selection(value)==value
with tempfile.TemporaryDirectory() as directory:
    root=Path(directory)
    (root/'statement-screening.lock').touch()
    marker=root/(m.RUN+'.started');marker.write_text(m.SOURCE);marker.chmod(0o600)
    path=root/(m.RUN+'.json');path.write_text(json.dumps(report));path.chmod(0o600)
    before={p.name:(p.read_bytes(),p.stat().st_mode,p.stat().st_mtime_ns) for p in root.iterdir()}
    with patch.object(m,'ROOT',root),contextlib.redirect_stdout(io.StringIO()) as output:
        assert m.main()==0
    assert json.loads(output.getvalue())['artifactsUnchanged'] and sentinel not in output.getvalue()
    assert before=={p.name:(p.read_bytes(),p.stat().st_mode,p.stat().st_mtime_ns) for p in root.iterdir()}
    def failure():
        with patch.object(m,'ROOT',root),contextlib.redirect_stdout(io.StringIO()) as output:
            assert m.main()==1
        assert json.loads(output.getvalue())=={'error':'s19_diagnostic_unavailable','verified':False}
    marker.write_text('0'*40);failure();marker.write_text(m.SOURCE)
    path.chmod(0o644);failure();path.chmod(0o600)
    path.unlink();path.symlink_to(marker);failure();path.unlink();failure()
    path.write_text(json.dumps(report));path.chmod(0o600)
    with (root/'statement-screening.lock').open('rb') as lock:
        m.fcntl.flock(lock,m.fcntl.LOCK_EX|m.fcntl.LOCK_NB);failure()
    marker.unlink();marker.symlink_to(path);failure();marker.unlink();marker.write_text(m.SOURCE);marker.chmod(0o600)
    path.write_bytes(b' '*524289);failure()
# Reader must never gain a provider/SQL or subprocess path.
source=Path(m.__file__).read_text()
assert all(token not in source for token in ('import subprocess','import socket','urllib','psql','docker exec','http.client'))
print('S1.9 reader: private/source/schema/privacy/lock/immutability PASS')

# The projection profile/pins must match the independent S1.8 inspector.
other=importlib.util.spec_from_file_location('inspect18',Path(__file__).with_name('inspect-statement-screening-s18.py'))
module=importlib.util.module_from_spec(other);other.loader.exec_module(module)
assert m.PROFILE==module.PROFILE and m.POLICY==module.POLICY
