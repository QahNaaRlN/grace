#!/usr/bin/env python3
"""Только чтение фиксированного S1.8 report/ledger, без ключей и inference."""
import datetime as dt
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import uuid

ROOT = Path('/opt/grace/shared')
RUN = 's18-gemini31-identity-20261021'
MODEL = 'google/gemini-3.1-flash-lite'
CANONICAL = 'google/gemini-3.1-flash-lite-20260507'
POLICY = {'model':MODEL,'provider':'google-vertex/eu','providerName':'Google','inputMicrosPerMillion':275000,'outputMicrosPerMillion':1650000}
PROFILE = {'Planner':'statement-document-atom.v2.lexical-grounding','Prompt':'statement-span-mapping.v12.turnover-roles','Schema':'52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf','Redaction':'statement-allowlist.v4.financial-labels','Alias':'statement-alias.source-order-base36.v2','Extractor':'poppler-25.04.0/bbox-v1','Scorer':'statement-screening.v1.19-20-23','Input':100000,'Output':16000,'Bytes':90000,'Spans':2000,'CallSeconds':120,'RunSeconds':420,'Calls':3,'CeilingMicros':161700}
CLASSES = {'not_observed','alias','pinned_canonical','other','empty','multiple'}
CODES = {'request','transport','timeout','http_4xx','http_5xx','http_other','read','oversize','envelope','metadata_missing','requested','strategy','attempt','byok','pipeline','selected_model','provider','count','display_provider','cost','usage','token_bound','choices','tools','finish_length','finish_other','content','unknown'}


def require(condition):
    if not condition:
        raise ValueError('unverified')


def timestamp(value):
    parsed = dt.datetime.fromisoformat(value.replace('Z','+00:00'))
    require(parsed.tzinfo is not None)
    return parsed.astimezone(dt.timezone.utc)


def private_read(path, limit):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, 'rb') as source:
        info = os.fstat(source.fileno())
        require(stat.S_ISREG(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o600)
        raw = source.read(limit+1)
    require(len(raw) <= limit)
    return raw


def route_verified(route, start, finish):
    if not isinstance(route, dict) or route.get('selectedModelClass') not in ('alias','pinned_canonical'):
        return False
    proof = route.get('proof') or {}
    if not (proof.get('alias') == MODEL and proof.get('canonical') == CANONICAL and proof.get('provider') == POLICY['provider'] and isinstance(proof.get('catalogSha256'),str) and re.fullmatch(r'[0-9a-f]{64}',proof['catalogSha256'])):
        return False
    try:
        observed, verified = timestamp(route['observedAt']), timestamp(proof['verifiedAt'])
        return start <= observed <= finish and 0 <= (observed-verified).total_seconds() <= 600
    except (KeyError, ValueError, TypeError, AttributeError):
        return False


def summary(data, now):
    failure = data.get('failure') or {}
    correlation = failure.get('unverifiedCorrelationId')
    calls, results = data.get('calls'), data.get('results')
    actual = data.get('actualMicros')
    out = {
        'expectedVersion':data.get('version') == 'statement.screening.v3.gemini31-identity',
        'expectedCandidate':data.get('policy') == POLICY,
        'expectedProfile':data.get('profile') == PROFILE and data.get('maximumMicros') == 161700,
        'billingKnown':type(actual) is int and 0 <= actual <= 161700,
        'recordedCalls':len(calls) if isinstance(calls,list) and len(calls)<=3 else -1,
        'scoredCases':len(results) if isinstance(results,list) and len(results)<=3 else -1,
        'diagnosticPresent':bool(failure),
        'diagnosticCode':failure.get('code') if failure.get('code') in CODES else 'unknown',
        'privateCorrelationPresent':isinstance(correlation,str) and len(correlation)<=128 and re.fullmatch(r'gen-[0-9A-Za-z-]+',correlation) is not None,
        'reportedPassed':data.get('passed') is True,
    }
    selected = (failure.get('route') or {}).get('selectedModelClass')
    out['selectedModelClass'] = selected if selected in CLASSES else 'not_observed'
    require(out['expectedVersion'] and out['expectedCandidate'] and out['expectedProfile'] and out['recordedCalls']>=0 and out['scoredCases']>=0)
    start, finish = timestamp(data['time']), timestamp(data['screeningFinishedAt'])
    require(start <= finish <= now)
    require(actual is None or out['billingKnown'])
    out['callProofsVerified'] = all(route_verified(call.get('screeningRoute'),start,finish) for call in calls)
    out['selectedModelClasses'] = [call.get('screeningRoute',{}).get('selectedModelClass') if call.get('screeningRoute',{}).get('selectedModelClass') in CLASSES else 'not_observed' for call in calls]
    known_sum = sum(call['costMicros'] for call in calls if type(call.get('costMicros')) is int and 0<=call['costMicros']<=53900)
    out['callCostsVerified'] = all(type(call.get('costMicros')) is int and 0<=call['costMicros']<=53900 for call in calls)
    require(out['callProofsVerified'] and out['callCostsVerified'])
    if out['billingKnown']:
        require(actual == known_sum)
    if out['reportedPassed']:
        require(out['billingKnown'] and len(calls)==3 and len(results)==3 and not failure and not data.get('error'))
        for result,name,checks in zip(results,['columns','cards','turnover-roles-ru'],[19,20,23]):
            require(result.get('name') == name and result.get('passed') is True and result.get('checks') == checks and result.get('correct') == checks)
    out['reportShapeVerified'] = True
    return out


def command(args):
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=15)
    require(result.returncode == 0)
    return result.stdout.strip()


def usage_summary(row, data):
    for field in ('id','owner_id'):
        require(isinstance(row.get(field),str) and str(uuid.UUID(row[field])) == row[field])
    require(row.get('purpose') == 'statement' and row.get('model') == MODEL and row.get('provider') == POLICY['provider'] and row.get('reserved_micros') == 161700 and row.get('generation_id') == 'statement-qualification:'+row['id'])
    require(abs((timestamp(row['created_at'])-timestamp(data['time'])).total_seconds()) <= 5)
    known = type(data.get('actualMicros')) is int
    status = row.get('status')
    require(status in ('settled','uncertain'))
    matches = status == 'settled' and known and row.get('actual_micros') == data['actualMicros'] or status == 'uncertain' and not known and row.get('actual_micros') is None
    require(matches)
    return {'reservationUnique':True,'reservationSettled':status=='settled','reservationUncertain':status=='uncertain','ledgerMatchesReport':True}


def main():
    out = {'s18InspectionReadOnly':True}
    try:
        require(len(sys.argv)==2 and re.fullmatch(r'[0-9a-f]{40}',sys.argv[1]))
        marker = ROOT / ('statement-screening-'+RUN+'.started')
        report = ROOT / ('statement-screening-'+RUN+'.json')
        out['authorizationConsumed'] = marker.exists()
        out['reportPresent'] = report.is_file()
        out['expectedSource'] = private_read(marker,128).decode().strip() == sys.argv[1]
        require(out['expectedSource'])
        data = json.loads(private_read(report,524288))
        out['reportPrivate'] = True
        out.update(summary(data, dt.datetime.now(dt.timezone.utc)))
        pg = command(['docker','ps','-q','--filter','label=com.docker.compose.project=grace-production','--filter','label=com.docker.compose.service=postgres'])
        require(re.fullmatch(r'[a-f0-9]+',pg))
        def sql(query):
            return command(['docker','exec',pg,'psql','-X','-q','-v','ON_ERROR_STOP=1','-U','grace','-d','grace','-Atc','BEGIN READ ONLY; '+query+'; COMMIT;'])
        stamp = timestamp(data['time']).isoformat()
        rows = json.loads(sql("SELECT COALESCE(json_agg(u),'[]') FROM ai_usage u WHERE purpose='statement' AND model='"+MODEL+"' AND provider='google-vertex/eu' AND reserved_micros=161700 AND generation_id LIKE 'statement-qualification:%' AND created_at BETWEEN '"+stamp+"'::timestamptz-interval '5 seconds' AND '"+stamp+"'::timestamptz+interval '5 seconds'"))
        require(len(rows)==1)
        out.update(usage_summary(rows[0],data))
        uncertain = sql("SELECT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain')")
        require(uncertain in ('t','f'))
        out['sharedUncertainBlock'] = uncertain == 't'
    except Exception:
        out['inspectionFailed'] = True
    print(json.dumps(out,separators=(',',':')))
    return 1 if out.get('inspectionFailed') else 0


if __name__ == '__main__':
    sys.exit(main())
