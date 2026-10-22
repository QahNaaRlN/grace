#!/usr/bin/env python3
"""O3 end-to-end writer/Go-summary/Python-inspector, entirely offline."""
import contextlib
import copy
import datetime as dt
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

sys.dont_write_bytecode=True
from test_comparison_contracts import frozen_v2_overlay
HERE=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('observed_inspector',HERE/'inspect-statement-comparison-s111.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
real_run=subprocess.run
source='a'*40
with tempfile.TemporaryDirectory(prefix='grace-s113-offline-') as tmp:
    root=Path(tmp);binary=root/'admin'
    env=dict(os.environ,S113_TEST_EXPORT_DIR=tmp)
    real_run(['go','test','./cmd/statement-admin','-run','^TestObservedComparisonSummaryPrivatePipeline$','-count=1'],env=env,check=True,stdout=subprocess.DEVNULL)
    current=json.loads((root/'observed.json').read_bytes())['report']
    assert current['profile']['Planner']=='statement-document-atom.v3.alias-enum'
    assert current['profile']!=m.PROFILE
    try:
        m._summary(current,dt.datetime.now(dt.timezone.utc),current['authorization'])
        raise AssertionError('frozen inspector accepted current contract')
    except ValueError:
        pass
    overlay=frozen_v2_overlay(root)
    real_run(['go','test',*overlay,'./cmd/statement-admin','-run','^TestObservedComparisonSummaryPrivatePipeline$','-count=1'],env=env,check=True,stdout=subprocess.DEVNULL)
    real_run(['go','build',*overlay,'-o',str(binary),'./cmd/statement-admin'],check=True,stdout=subprocess.DEVNULL)
    raw=(root/'observed.json').read_bytes();data=json.loads(raw);report=data['report']
    assert report['profile']==m.PROFILE
    summary=real_run([str(binary),'-mode=comparison-summary'],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE,check=True)
    assert json.loads(summary.stdout)==json.loads((root/'summary.json').read_bytes())
    assert all(s not in summary.stdout for s in (b'PRIVATE',b'snapshot',b'actualMicros',b'generationId',b'reservationId'))
    m.ROOT=root;marker=root/('statement-comparison-'+m.RUN+'.started');artifact=root/('statement-comparison-'+m.RUN+'.json')
    marker.write_text(source+'\n');marker.chmod(0o600);artifact.write_bytes(raw);artifact.chmod(0o600)
    calls=[]
    def command(args):
        calls.append(args)
        if args[1]=='ps':return 'abc123'
        sql=args[-1];assert sql.startswith('BEGIN READ ONLY; ') and sql.endswith('; COMMIT;')
        assert not any(s in sql for s in ('INSERT ','UPDATE ','DELETE ','grace_ai_reserve','grace_ai_settle'))
        if 'json_agg' in sql:
            rid=report['reservationId']
            return json.dumps([dict(id=rid,owner_id='66666666-7777-4888-8999-000000000000',purpose='statement',model='statement-comparison.s111',provider='fixed-A-B',reserved_micros=208500,actual_micros=report['actualMicros'],generation_id='statement-qualification:'+rid,created_at=report['time'],status='settled')])
        return 'f'
    m.command=command
    def go_verifier(args,**kwargs):
        assert args==['docker','exec','-i','abc123','/tmp/grace-statement-comparison-'+source+'/grace-statement-admin','-mode=comparison-summary']
        assert kwargs['input'] and kwargs['timeout']==15
        return real_run([str(binary),'-mode=comparison-summary'],input=kwargs['input'],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    old_run=m.subprocess.run;old_argv=sys.argv
    m.subprocess.run=go_verifier;sys.argv=['inspect',source]
    try:
        for kind in ('observed','legacy','inner','outer','private-diagnostic','unknown-field','duplicate','oversize','symlink'):
            artifact.unlink(missing_ok=True)
            value=copy.deepcopy(data)
            if kind=='legacy':value=report
            if kind=='inner':value['report']['digest']='0'*64
            if kind=='outer':value['digest']='0'*64
            if kind=='private-diagnostic':value['observations'][0]['diagnostic']['code']='PRIVATE'
            if kind=='unknown-field':value['observations'][0]['PRIVATE']='PRIVATE'
            encoded=json.dumps(value).encode()
            if kind=='duplicate':encoded=encoded.replace(b'"version":',b'"Version":"PRIVATE","version":',1)
            if kind=='oversize':encoded=b' '*524289
            if kind=='symlink':artifact.symlink_to(marker)
            else:artifact.write_bytes(encoded);artifact.chmod(0o600)
            before={p:(p.read_bytes(),p.stat().st_mtime_ns,p.stat().st_mode) for p in (marker,artifact)}
            out=io.StringIO()
            with contextlib.redirect_stdout(out):status=m.main()
            text=out.getvalue();assert 'PRIVATE' not in text and 'snapshot' not in text and 'actualMicros' not in text and report['reservationId'] not in text
            result=json.loads(text)
            if kind in ('observed','legacy'):
                assert status==0 and result['ledgerMatchesReport']
                assert result['observationUnavailableLegacy']==(kind=='legacy')
            else:assert status==1 and result['inspectionFailed']
            after={p:(p.read_bytes(),p.stat().st_mtime_ns,p.stat().st_mode) for p in (marker,artifact)}
            assert before==after
    finally:
        m.subprocess.run=old_run;sys.argv=old_argv
print('S1.13 actual Go writer/CLI validation + Python inspector, legacy, corruption, privacy and immutability PASS')
