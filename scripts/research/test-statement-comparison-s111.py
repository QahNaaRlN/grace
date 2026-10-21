#!/usr/bin/env python3
"""S1.11 inspector tests: private fixtures only, no real Docker/SQL/network."""
import contextlib
import copy
import datetime as dt
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import sys

sys.dont_write_bytecode=True
spec=importlib.util.spec_from_file_location('inspector',Path(__file__).with_name('inspect-statement-comparison-s111.py'))
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
now=dt.datetime.now(dt.timezone.utc);start=now-dt.timedelta(seconds=10);finish=now-dt.timedelta(seconds=1)
iso=lambda t:t.isoformat()
RID='11111111-2222-4333-8444-555555555555'
OWNER='66666666-7777-4888-8999-000000000000'

def fixture():
    data={'version':'statement.comparison.v1.grounding','authorization':m.RUN,'profile':copy.deepcopy(m.PROFILE),'candidates':copy.deepcopy(m.POLICIES),'maximumMicros':208500,'digest':'d'*64,'time':iso(start),'finishedAt':iso(finish),'reservationId':RID,'actualMicros':600,'passed':True,'error':'','tuples':[]}
    for i in range(6):
        c=copy.deepcopy(m.POLICIES[i%2]);name=m.NAMES[i//2];checks=[19,20,23][i//2]
        proof={'candidate':c,'catalogSha256':'a'*64,'endpointSha256':'b'*64,'zdrSha256':'c'*64,'catalogUrl':'https://openrouter.ai/api/v1/models','endpointUrl':'https://openrouter.ai/api/v1/models/'+c['canonical']+'/endpoints','zdrUrl':'https://openrouter.ai/api/v1/endpoints/zdr','verifiedAt':iso(start-dt.timedelta(seconds=1))}
        data['tuples'].append({'candidate':c['id'],'evidence':{'name':name,'fixtureHash':m.HASHES[i//2],'documentHash':'a'*64,'planHash':'b'*64,'inputBytes':1000},'requestHash':'c'*64,'state':'completed','reason':'','falseAcceptance':False,'call':{'generationId':'PRIVATE-generation-'+str(i),'costMicros':100,'inputTokens':1000,'outputTokens':100,'latencyMs':10,'route':{'proof':proof,'observedAt':iso(start+dt.timedelta(seconds=i)),'selectedModelClass':'pinned_canonical'}},'result':{'name':name,'rows':2,'passed':True,'checks':checks,'correct':checks,'codes':[],'validation':[],'coverage':[],'latencyMs':10},'proposal':{}})
    data['proofs']=[copy.deepcopy(data['tuples'][i]['call']['route']['proof']) for i in (0,1)]
    return data

def row(data):
    return {'id':RID,'owner_id':OWNER,'purpose':'statement','model':'statement-comparison.s111','provider':'fixed-A-B','reserved_micros':208500,'actual_micros':data['actualMicros'],'generation_id':'statement-qualification:'+RID,'created_at':iso(start+dt.timedelta(milliseconds=10)),'status':'settled' if data['actualMicros'] is not None else 'uncertain'}

good=fixture();assert m.summary(good,now)['reportedPassed']
for change in ('version','profile','order','hash','route','canonical','duplicate','cost','unknown','score','time','reservation'):
    d=fixture()
    if change=='version':d['version']='old'
    elif change=='profile':d['profile']['Scorer']='old'
    elif change=='order':d['tuples'][0]['candidate']='B'
    elif change=='hash':d['tuples'][0]['evidence']['fixtureHash']='bad'
    elif change=='route':d['tuples'][0]['call']['route']['observedAt']=iso(start-dt.timedelta(seconds=2))
    elif change=='canonical':d['tuples'][1]['call']['route']['proof']['candidate']['canonical']='other'
    elif change=='duplicate':d['tuples'][1]['call']['generationId']=d['tuples'][0]['call']['generationId']
    elif change=='cost':d['actualMicros']=0
    elif change=='unknown':d['actualMicros']=None
    elif change=='score':d['tuples'][0]['result']['correct']=18
    elif change=='time':d['finishedAt']=iso(now+dt.timedelta(seconds=1))
    elif change=='reservation':d['reservationId']='not-uuid'
    try:
        m.summary(d,now)
        if change=='reservation':m.usage_summary(row(d),d)
        raise RuntimeError('tampering accepted: '+change)
    except (ValueError,KeyError,TypeError):pass

failed=fixture();failed['passed']=False;failed['error']='qualification_quality_failed'
failed['tuples'][0]['reason']='quality_failed';failed['tuples'][0]['result'].update(passed=False,correct=18,codes=['PRIVATE-sentinel'])
assert m.summary(failed,now)['successfulA']==2
unknown=fixture();unknown.update(passed=False,error='comparison_provider_failed',actualMicros=None)
for i,t in enumerate(unknown['tuples']):
    t.update(state='attempted' if i==0 else 'not_sent',reason='provider_failed' if i==0 else 'campaign_not_started');t.pop('call');t.pop('result');t.pop('proposal')
assert not m.summary(unknown,now)['billingKnown']
assert m.usage_summary(row(unknown),unknown)['reservationUncertain']

with tempfile.TemporaryDirectory() as tmp:
    root=Path(tmp);m.ROOT=root;source='a'*40;marker=root/('statement-comparison-'+m.RUN+'.started');report=root/('statement-comparison-'+m.RUN+'.json')
    marker.write_text(source+'\n');report.write_text(json.dumps(failed));os.chmod(marker,0o600);os.chmod(report,0o600)
    before={p.name:(p.read_bytes(),p.stat().st_mtime_ns,p.stat().st_mode) for p in root.iterdir()}
    commands=[]
    def command(args):
        commands.append(args)
        if args[1]=='ps':return 'abc123'
        sql=args[-1]
        assert sql.startswith('BEGIN READ ONLY; SELECT ') and sql.endswith('; COMMIT;')
        assert not any(x in sql for x in ('UPDATE ','INSERT ','DELETE ','grace_ai_reserve','grace_ai_settle'))
        if 'json_agg' in sql:
            assert "WHERE id='"+RID+"'::uuid" in sql
            return json.dumps([row(failed)])
        assert "WHERE status='uncertain'" in sql
        return 'f'
    m.command=command;sys.argv=['inspect',source];out=io.StringIO()
    with contextlib.redirect_stdout(out):assert m.main()==0
    assert all(s not in out.getvalue() for s in ('PRIVATE',RID,OWNER,'actualMicros','generationId','costMicros'))
    after={p.name:(p.read_bytes(),p.stat().st_mtime_ns,p.stat().st_mode) for p in root.iterdir()};assert before==after
    assert len(commands)==3
    os.chmod(report,0o644);out=io.StringIO()
    with contextlib.redirect_stdout(out):assert m.main()==1
    os.chmod(report,0o600);report.unlink();report.symlink_to(root/'absent');out=io.StringIO()
    with contextlib.redirect_stdout(out):assert m.main()==1
try:json.loads('{"passed":true,"Passed":false}',object_pairs_hook=m.unique);raise RuntimeError('duplicate accepted')
except ValueError:pass
print('S1.11 inspector: pins/proof/cost/refusals, exact unique reservation, READ ONLY, privacy and file immutability PASS')
