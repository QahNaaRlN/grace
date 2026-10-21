#!/usr/bin/env python3
"""Read exactly the fixed S1.11 private report and reservation; never inference/writes."""
import datetime as dt
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import uuid

ROOT=Path('/opt/grace/shared')
RUN='s111-comparison-20261021'
POLICIES=[{'id':'A','policy':{'model':'google/gemini-3.1-flash-lite','provider':'google-vertex/eu','providerName':'Google','inputMicrosPerMillion':275000,'outputMicrosPerMillion':1650000},'canonical':'google/gemini-3.1-flash-lite-20260507'}, {'id':'B','policy':{'model':'inception/mercury-3','provider':'inception','providerName':'Inception','inputMicrosPerMillion':100000,'outputMicrosPerMillion':350000},'canonical':'inception/mercury-3-20261019'}]
PROFILE={'Planner':'statement-document-atom.v2.lexical-grounding','Prompt':'statement-span-mapping.v12.turnover-roles','Schema':'52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf','Redaction':'statement-allowlist.v4.financial-labels','Alias':'statement-alias.source-order-base36.v2','Extractor':'poppler-25.04.0/bbox-v1','Scorer':'statement-screening.v2.grounding19-20-23','Input':100000,'Output':16000,'Bytes':90000,'Spans':2000,'CallSeconds':120,'RunSeconds':900,'Calls':6,'CeilingMicros':208500}
NAMES=['columns','cards','turnover-roles-ru']
HASHES=['097c3d35d4dd9787457b13f61bd524ff83c18d969ce50b93f7c8cecfeb569c8c','d4dd02153724c1599d4d89eb073e8cfaaf3fbce1696a4d691b69d150d0ccdce6','648ac4477b9a78a7a1bb43eb9ae918fd5b6a2eb338e2ee07038146141781686f']
ERRORS={'','qualification_budget_unavailable','comparison_provider_failed','qualification_timeout','qualification_protocol_failed','qualification_cost_overrun','qualification_quality_failed','qualification_settlement_failed'}

def require(ok):
    if not ok: raise ValueError('unverified')

def stamp(value):
    parsed=dt.datetime.fromisoformat(value.replace('Z','+00:00'))
    require(parsed.tzinfo is not None)
    return parsed.astimezone(dt.timezone.utc)

def digest(value):
    return isinstance(value,str) and re.fullmatch(r'[0-9a-f]{64}',value) is not None

def unique(pairs):
    out={};seen=set()
    for k,v in pairs:
        require(k.lower() not in seen);seen.add(k.lower());out[k]=v
    return out

def private_read(path,limit):
    fd=os.open(path,os.O_RDONLY|os.O_NOFOLLOW)
    with os.fdopen(fd,'rb') as source:
        info=os.fstat(source.fileno())
        require(stat.S_ISREG(info.st_mode) and stat.S_IMODE(info.st_mode)==0o600)
        raw=source.read(limit+1)
    require(len(raw)<=limit)
    return raw

def route_valid(route,c,start,finish):
    try:
        p=route['proof'];v=stamp(p['verifiedAt']);o=stamp(route['observedAt'])
        return (route['selectedModelClass'] in ('alias','pinned_canonical') and p['candidate']==c and start<=o<=finish and 0<=(o-v).total_seconds()<=600 and all(digest(p[k]) for k in ('catalogSha256','endpointSha256','zdrSha256')) and p['catalogUrl']=='https://openrouter.ai/api/v1/models' and p['endpointUrl']=='https://openrouter.ai/api/v1/models/'+c['canonical']+'/endpoints' and p['zdrUrl']=='https://openrouter.ai/api/v1/endpoints/zdr')
    except (KeyError,TypeError,ValueError,AttributeError): return False

def summary(data,now):
    require(data.get('version')=='statement.comparison.v1.grounding' and data.get('authorization')==RUN and data.get('profile')==PROFILE and data.get('candidates')==POLICIES and data.get('maximumMicros')==208500 and digest(data.get('digest')))
    start,finish=stamp(data['time']),stamp(data['finishedAt']);require(start<=finish<=now and (finish-start).total_seconds()<=906)
    tuples=data.get('tuples');require(isinstance(tuples,list) and len(tuples)==6)
    require(data.get('error') in ERRORS and type(data.get('passed')) is bool)
    actual=data.get('actualMicros');require(actual is None or type(actual) is int and 0<=actual<=208500)
    proofs=data.get('proofs');require(isinstance(proofs,list) and len(proofs)==2)
    for p,c in zip(proofs,POLICIES):
        v=stamp(p['verifiedAt']);require(v<=finish and route_valid({'proof':p,'observedAt':p['verifiedAt'],'selectedModelClass':'alias'},c,min(start,v),finish))
    total=0;calls=0;completed=0;successful=[0,0];seen=set();stopped=False;all_passed=True;rows=[]
    for i,t in enumerate(tuples):
        c=POLICIES[i%2];e=t['evidence'];name=NAMES[i//2];checks=[19,20,23][i//2]
        require(t.get('candidate')==c['id'] and e.get('name')==name and e.get('fixtureHash')==HASHES[i//2] and all(digest(e.get(k)) for k in ('documentHash','planHash')) and digest(t.get('requestHash')) and type(e.get('inputBytes')) is int and 0<e['inputBytes']<=90000)
        if i%2: require(e['documentHash']==tuples[i-1]['evidence']['documentHash'] and e['inputBytes']==tuples[i-1]['evidence']['inputBytes'])
        state=t.get('state');reason=t.get('reason');call=t.get('call');result=t.get('result')
        require(state in ('not_sent','attempted','completed') and reason in ('','campaign_not_started','provider_failed','protocol_failed','grounding_failed','quality_failed'))
        if state=='not_sent':
            stopped=True;all_passed=False
            require(call is None and result is None and t.get('proposal') is None and reason=='campaign_not_started')
        else:
            require(not stopped)
            if state=='attempted': stopped=True;all_passed=False;require(reason in ('provider_failed','protocol_failed') and result is None)
            if call is not None:
                gid=call.get('generationId');require(isinstance(gid,str) and 0<len(gid)<=200 and gid not in seen);seen.add(gid)
                require(type(call.get('costMicros')) is int and 0<=call['costMicros']<=[53900,15600][i%2] and type(call.get('inputTokens')) is int and 1<=call['inputTokens']<=100000 and type(call.get('outputTokens')) is int and 0<=call['outputTokens']<=16000 and type(call.get('latencyMs')) is int and call['latencyMs']>=0)
                total+=call['costMicros'];calls+=1
            if state=='completed':
                completed+=1
                require(call is not None and call['outputTokens']>0 and route_valid(call.get('route'),c,start,finish) and isinstance(t.get('proposal'),dict))
                if reason=='grounding_failed': require(result is None);all_passed=False
                else:
                    require(reason in ('','quality_failed') and isinstance(result,dict) and result.get('name')==name)
                    good=(result.get('passed') is True and result.get('rows')==2 and result.get('checks')==checks and result.get('correct')==checks and not result.get('codes') and not result.get('validation') and not result.get('coverage'))
                    require((reason=='')==good and not(reason=='quality_failed' and result.get('passed') is True))
                    all_passed=all_passed and good;successful[i%2]+=int(good)
        rows.append({'candidate':c['id'],'name':name,'state':state,'reason':reason})
    if actual is not None: require(total==actual)
    require(data['passed']==(all_passed and completed==6 and actual is not None and not data['error']))
    require(data['passed'] or bool(data['error']))
    return {'reportShapeVerified':True,'reportedPassed':data['passed'],'billingKnown':actual is not None,'recordedCalls':calls,'completedTuples':completed,'successfulA':successful[0],'successfulB':successful[1],'error':data['error'],'tuples':rows,'activationAuthorized':False}

def usage_summary(row,data):
    rid=data.get('reservationId');require(isinstance(rid,str) and str(uuid.UUID(rid))==rid and row.get('id')==rid)
    require(str(uuid.UUID(row['owner_id']))==row['owner_id'])
    require(row.get('purpose')=='statement' and row.get('model')=='statement-comparison.s111' and row.get('provider')=='fixed-A-B' and row.get('reserved_micros')==208500 and row.get('generation_id')=='statement-qualification:'+rid)
    require(stamp(data['time'])<=stamp(row['created_at'])<=stamp(data['finishedAt']))
    known=type(data.get('actualMicros')) is int;status=row.get('status')
    require(status=='settled' and known and row.get('actual_micros')==data['actualMicros'] or status=='uncertain' and not known and row.get('actual_micros') is None)
    return {'reservationUnique':True,'reservationSettled':status=='settled','reservationUncertain':status=='uncertain','ledgerMatchesReport':True}

def command(args):
    result=subprocess.run(args,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,timeout=15)
    require(result.returncode==0);return result.stdout.strip()

def verify_observed(raw,source):
    # The exact-source binary is already staged by the existing one-shot runner.
    # comparison-summary returns before config/key/SQL/provider constructors.
    api=command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service=api'])
    require(re.fullmatch(r'[a-f0-9]+',api))
    binary='/tmp/grace-statement-comparison-'+source+'/grace-statement-admin'
    result=subprocess.run(['docker','exec','-i',api,binary,'-mode=comparison-summary'],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=15)
    require(result.returncode==0 and len(result.stdout)<=32768)
    out=json.loads(result.stdout,object_pairs_hook=unique)
    require(out.get('reportShapeVerified') is True and out.get('observationsVerified') is True and out.get('observationUnavailableLegacy') is False)
    observations=out.get('observations');require(isinstance(observations,list) and len(observations)==6)
    for i,o in enumerate(observations):
        require(set(o)<= {'ordinal','stage','captureStatus','diagnostic'} and type(o.get('ordinal')) is int and o.get('ordinal')==i+1)
        require(o.get('stage') in ('not_reached','canonicalization_passed','canonicalization_failed') and o.get('captureStatus') in ('not_reached','captured','unavailable_limit','unavailable_report_limit'))
        d=o.get('diagnostic')
        if d is not None:
            require(set(d)=={'version','stage','code','scope','field','ordinal','selectionIndex','role'})
            require(d['version']=='statement.canonicalization-diagnostic.v1' and d['stage']=='canonicalization' and d['code'] in ('invalid_plan','unknown_alias','duplicate_alias','non_numeric_money_selection'))
            require(d['scope'] in ('plan','header','row','header_label','service') and d['field'] in ('plan','operationDate','bookingDate','accountAmount','accountCurrency','transactionAmount','transactionCurrency','description','directionEvidence','openingBookDate','closingBookDate','openingBookBalance','closingBookBalance','availableBalanceObserved','debitTurnover','creditTurnover','currency','labels','spans'))
            require(type(d['ordinal']) is int and d['ordinal']>=0 and type(d['selectionIndex']) is int and d['selectionIndex']>=0 and d['role'] in ('','unknown','openingBookBalance','closingBookBalance','availableBalanceObserved','debitTurnover','creditTurnover'))
    return observations

def main():
    out={'s111InspectionReadOnly':True}
    try:
        require(len(sys.argv)==2 and re.fullmatch(r'[0-9a-f]{40}',sys.argv[1]))
        marker=ROOT/('statement-comparison-'+RUN+'.started');report=ROOT/('statement-comparison-'+RUN+'.json')
        require(private_read(marker,128).decode().strip()==sys.argv[1])
        raw=private_read(report,524288)
        data=json.loads(raw,object_pairs_hook=unique)
        if data.get('version')=='statement.comparison.observed.v1':
            observations=verify_observed(raw,sys.argv[1])
            data=data['report']
            out['observationsVerified']=True;out['observations']=observations;out['observationUnavailableLegacy']=False
        else: out['observationUnavailableLegacy']=True
        out.update(summary(data,dt.datetime.now(dt.timezone.utc)))
        rid=data['reservationId'];require(str(uuid.UUID(rid))==rid)
        pg=command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service=postgres']);require(re.fullmatch(r'[a-f0-9]+',pg))
        def sql(q): return command(['docker','exec',pg,'psql','-X','-q','-v','ON_ERROR_STOP=1','-U','grace','-d','grace','-Atc','BEGIN READ ONLY; '+q+'; COMMIT;'])
        rows=json.loads(sql("SELECT COALESCE(json_agg(u),'[]') FROM ai_usage u WHERE id='"+rid+"'::uuid"));require(len(rows)==1)
        out.update(usage_summary(rows[0],data))
        uncertain=sql("SELECT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain')");require(uncertain in ('t','f'));out['sharedUncertainBlock']=uncertain=='t'
    except Exception: out['inspectionFailed']=True
    print(json.dumps(out,separators=(',',':')))
    return 1 if out.get('inspectionFailed') else 0

if __name__=='__main__': sys.exit(main())
