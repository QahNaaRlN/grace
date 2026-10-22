#!/usr/bin/env python3
"""Actual S1.14 Go engine/writer/CLI/replay + fixed Python inspector, offline only."""
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
here=Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('s114_inspector',here/'inspect-statement-comparison-s114.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
real_run=subprocess.run
source='a'*40
with tempfile.TemporaryDirectory(prefix='grace-s114-test-') as tmp:
    root=Path(tmp);binary=root/'admin';replay=root/'replay'
    env=dict(os.environ,S114_TEST_EXPORT_DIR=tmp)
    real_run(['go','test','./cmd/statement-admin','-run','^TestS114SummaryPrivatePipeline$','-count=1'],env=env,check=True,stdout=subprocess.DEVNULL)
    current=json.loads((root/'observed.json').read_bytes())['report']
    assert current['profile']['Planner']=='statement-document-atom.v3.alias-enum'
    assert current['profile']!=m.base.PROFILE
    try:
        m.base._summary(current,dt.datetime.now(dt.timezone.utc),current['authorization'])
        raise AssertionError('frozen inspector accepted current contract')
    except ValueError:
        pass
    overlay=frozen_v2_overlay(root)
    real_run(['go','test',*overlay,'./cmd/statement-admin','-run','^TestS114SummaryPrivatePipeline$','-count=1'],env=env,check=True,stdout=subprocess.DEVNULL)
    real_run(['go','build',*overlay,'-o',str(binary),'./cmd/statement-admin'],check=True,stdout=subprocess.DEVNULL)
    real_run(['go','build',*overlay,'-o',str(replay),'./scripts/research/s113-replay'],check=True,stdout=subprocess.DEVNULL)
    raw=(root/'observed.json').read_bytes();data=json.loads(raw);report=data['report']
    assert report['profile']==m.base.PROFILE
    summary=real_run([str(binary),'-mode=comparison-s114-summary'],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE,check=True)
    assert json.loads(summary.stdout)==json.loads((root/'summary.json').read_bytes())
    assert all(secret not in summary.stdout for secret in (b'PRIVATE',b'snapshot',b'generationId',b'reservationId'))
    legacy=real_run([str(binary),'-mode=comparison-summary'],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    assert legacy.returncode!=0 and b'PRIVATE' not in legacy.stdout+legacy.stderr
    # Replay stdin is S1.14-only. Missing local helper is honest UNVERIFIED, no key/SQL/config.
    replay_result=real_run([str(replay),'-',str(root),str(root/'missing.sock')],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    assert replay_result.returncode!=0 and json.loads(replay_result.stdout)=={'verified':False,'status':'UNVERIFIED','providerCalls':0}
    m.ROOT=root;marker=root/('statement-comparison-'+m.RUN+'.started');artifact=root/('statement-comparison-'+m.RUN+'.json')
    marker.write_text(source+'\n');marker.chmod(0o600)
    row=dict(id=report['reservationId'],owner_id='66666666-7777-4888-8999-000000000000',purpose='statement',model='statement-comparison.s114',provider='fixed-A-B',reserved_micros=208500,actual_micros=report['actualMicros'],generation_id='statement-qualification:'+report['reservationId'],created_at=report['time'],status='settled')
    rows=[row];commands=[];replay_mode='actual'
    def command(args):
        commands.append(args)
        if args[1]=='ps':return 'abc123'
        sql=args[-1];assert sql.startswith('BEGIN READ ONLY; ') and sql.endswith('; COMMIT;')
        assert not any(text in sql for text in ('INSERT ','UPDATE ','DELETE ','grace_ai_reserve','grace_ai_settle'))
        if 'json_agg' in sql:
            assert "id='"+report['reservationId']+"'::uuid" in sql
            return json.dumps(rows)
        return 'f'
    m.base.command=command
    def verifier(args,**kwargs):
        assert args[:4]==['docker','exec','-i','abc123']
        if args[4].endswith('/grace-s114-replay'):
            assert args[5:]==['-',m.PREFIX+source,'/run/grace-pdf/extractor.sock'] and kwargs['timeout']==50
            if replay_mode=='private':return subprocess.CompletedProcess(args,0,b'{"PRIVATE":"PRIVATE"}',b'PRIVATE')
            return real_run([str(replay),'-',str(root),str(root/'missing.sock')],input=kwargs['input'],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        assert args[4:]==[m.PREFIX+source+'/grace-statement-admin','-mode=comparison-s114-summary']
        return real_run([str(binary),'-mode=comparison-s114-summary'],input=kwargs['input'],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    old_run=subprocess.run;old_argv=sys.argv
    subprocess.run=verifier;sys.argv=['inspect',source]
    try:
        for kind in ('valid','private-replay','legacy','inner','outer','unknown-field','duplicate','oversize','symlink','mode','wrong-marker','old-ledger','duplicate-ledger','wrong-cost','wrong-reservation'):
            artifact.unlink(missing_ok=True);marker.write_text(source+'\n');marker.chmod(0o600)
            value=copy.deepcopy(data);rows[:]=[copy.deepcopy(row)];replay_mode='private' if kind=='private-replay' else 'actual'
            if kind=='legacy':value=report
            if kind=='inner':value['report']['digest']='0'*64
            if kind=='outer':value['digest']='0'*64
            if kind=='unknown-field':value['observations'][0]['PRIVATE']='PRIVATE'
            if kind=='wrong-marker':marker.write_text('b'*40+'\n')
            if kind=='old-ledger':rows[0]['model']='statement-comparison.s111'
            if kind=='duplicate-ledger':rows.append(copy.deepcopy(row))
            if kind=='wrong-cost':rows[0]['actual_micros']+=1
            if kind=='wrong-reservation':rows[0]['id']='66666666-7777-4888-8999-000000000000'
            encoded=json.dumps(value).encode()
            if kind=='duplicate':encoded=encoded.replace(b'"version":',b'"Version":"PRIVATE","version":',1)
            if kind=='oversize':encoded=b' '*524289
            if kind=='symlink':artifact.symlink_to(marker)
            else:artifact.write_bytes(encoded);artifact.chmod(0o644 if kind=='mode' else 0o600)
            before={p:(p.read_bytes(),p.stat().st_mtime_ns,p.stat().st_mode) for p in (marker,artifact)}
            output=io.StringIO()
            with contextlib.redirect_stdout(output):status=m.main()
            text=output.getvalue();result=json.loads(text)
            assert all(secret not in text for secret in ('PRIVATE','snapshot','generationId',report['reservationId']))
            if kind in ('valid','private-replay'):
                assert status==0 and result['ledgerMatchesReport'] and result['artifactsUnchanged']
                assert result['replay']['providerCalls']==0 and all(x['status']=='UNVERIFIED' for x in result['replay']['results'])
            else:assert status==1 and result['inspectionFailed'],kind
            assert before=={p:(p.read_bytes(),p.stat().st_mtime_ns,p.stat().st_mode) for p in (marker,artifact)}
    finally:subprocess.run=old_run;sys.argv=old_argv
print('S1.14 actual engine/writer/Go CLI/replay + inspector: context, reservation, corruption, privacy, read-only PASS')
