#!/usr/bin/env python3
"""Fixed S1.17 read-only inspection. Private bytes only enter exact-source Go tools."""
import datetime as dt
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import sys

sys.dont_write_bytecode=True
spec=importlib.util.spec_from_file_location('comparison_legacy',Path(__file__).with_name('inspect-statement-comparison-s111.py'))
base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
ROOT=Path('/opt/grace/shared')
RUN='s117-comparison-20261023'
PREFIX='/tmp/grace-statement-comparison-s117-'
PROFILE={**base.PROFILE,'Planner':'statement-document-atom.v3.alias-enum','Schema':'1a6e755395da69883666f3eb61471ad8c4257f7d465451a4ad0050fbd253c3c7'}
require=base.require

def metrics_closed(rows):
    require(isinstance(rows,list) and len(rows)==2)
    keys={'falseAcceptances','candidate','attempted','completed','successful','quality','efficiency','knownCostMicros','costPerSuccessNumerator','costPerSuccessDenominator','latenciesMs'}
    for i,row in enumerate(rows):
        require(set(row)==keys and row['candidate']==('A','B')[i])
        for k in ('falseAcceptances','attempted','completed','successful','costPerSuccessDenominator'):
            require(type(row[k]) is int and 0<=row[k]<=3)
        require(type(row['knownCostMicros']) is int and 0<=row['knownCostMicros']<=(161700,46800)[i])
        n=row['costPerSuccessNumerator'];require(n is None or type(n) is int and n==row['knownCostMicros'])
        require(row['quality'] in ('PASS','FAIL','UNVERIFIED') and row['efficiency'] in ('UNVERIFIED','undefined','observed'))
        require(isinstance(row['latenciesMs'],list) and len(row['latenciesMs'])<=3 and all(type(n) is int and n>=0 for n in row['latenciesMs']))
    return rows

def replay(raw,source,observations):
    unavailable={'providerCalls':0,'results':[{'verified':False,'status':'UNVERIFIED'} for _ in range(6)]}
    try:
        api=base.command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service=api'])
        require(re.fullmatch(r'[a-f0-9]+',api))
        result=subprocess.run(['docker','exec','-i',api,PREFIX+source+'/grace-s117-replay','-',PREFIX+source,'/run/grace-pdf/extractor.sock'],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=50)
        require(result.returncode==0 and len(result.stdout)<=32768)
        value=json.loads(result.stdout,object_pairs_hook=base.unique)
        require(set(value)=={'providerCalls','results'} and type(value['providerCalls']) is int and value['providerCalls']==0 and isinstance(value['results'],list) and len(value['results'])==6)
        for row,observation in zip(value['results'],observations):
            require(set(row)<={'stage','verified','status','diagnostic'} and type(row.get('verified')) is bool and row.get('status') in ('matched','UNVERIFIED','unavailable_capture'))
            require(row['verified']==(row['status']=='matched') and row.get('stage') in (None,'not_reached','canonicalization_passed','canonicalization_failed'))
            if row.get('diagnostic') is not None: require(row['diagnostic']==observation.get('diagnostic'))
            if row['verified']: require(row.get('stage')==observation['stage'] and row.get('diagnostic')==observation.get('diagnostic'))
        return value
    except Exception:
        return unavailable

def signature(path):
    raw=base.private_read(path,524288)
    info=path.stat()
    return hashlib.sha256(raw).hexdigest(),info.st_mode,info.st_mtime_ns

def main():
    out={'s117InspectionReadOnly':True}
    try:
        require(len(sys.argv)==2 and re.fullmatch(r'[0-9a-f]{40}',sys.argv[1]))
        source=sys.argv[1]
        marker=ROOT/('statement-comparison-'+RUN+'.started');artifact=ROOT/('statement-comparison-'+RUN+'.json')
        before=(signature(marker),signature(artifact))
        require(base.private_read(marker,128).decode().strip()==source)
        raw=base.private_read(artifact,524288)
        envelope=json.loads(raw,object_pairs_hook=base.unique)
        require(envelope.get('version')=='statement.comparison.observed.s117.v1')
        verified=base._verify_observed_summary(raw,source,'comparison-s117-summary',PREFIX)
        observations=verified['observations']
        data=envelope['report']
        out.update(base._summary_for_profile(data,dt.datetime.now(dt.timezone.utc),RUN,PROFILE))
        out.update(observationsVerified=True,observations=observations,metrics=metrics_closed(verified['metrics']))
        rid=data['reservationId'];require(str(base.uuid.UUID(rid))==rid)
        pg=base.command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service=postgres']);require(re.fullmatch(r'[a-f0-9]+',pg))
        def sql(q): return base.command(['docker','exec',pg,'psql','-X','-q','-v','ON_ERROR_STOP=1','-U','grace','-d','grace','-Atc','BEGIN READ ONLY; '+q+'; COMMIT;'])
        rows=json.loads(sql("SELECT COALESCE(json_agg(u),'[]') FROM ai_usage u WHERE id='"+rid+"'::uuid"));require(len(rows)==1)
        out.update(base._usage_summary(rows[0],data,'statement-comparison.s117'))
        uncertain=sql("SELECT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain')");require(uncertain in ('t','f'));out['sharedUncertainBlock']=uncertain=='t'
        out['replay']=replay(raw,source,observations)
        require(before==(signature(marker),signature(artifact)))
        out['artifactsUnchanged']=True
    except Exception:
        # Never print exception text, raw payload, identities or partial private data.
        out={'s117InspectionReadOnly':True,'inspectionFailed':True,'activationAuthorized':False}
    print(json.dumps(out,separators=(',',':')))
    return 1 if out.get('inspectionFailed') else 0

if __name__=='__main__': sys.exit(main())
