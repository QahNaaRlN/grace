#!/usr/bin/env python3
"""S1.9: фиксированный private S1.8 -> закрытая проекция; никаких SQL/HTTP."""
import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import sys

ROOT = Path('/opt/grace/shared')
SOURCE = 'f450e9d629a4543821a21b8f7b3aa679788872c4'
RUN = 'statement-screening-s18-gemini31-identity-20261021'
MODEL = 'google/gemini-3.1-flash-lite'
POLICY = {'model':MODEL,'provider':'google-vertex/eu','providerName':'Google','inputMicrosPerMillion':275000,'outputMicrosPerMillion':1650000}
PROFILE = {'Planner':'statement-document-atom.v2.lexical-grounding','Prompt':'statement-span-mapping.v12.turnover-roles','Schema':'52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf','Redaction':'statement-allowlist.v4.financial-labels','Alias':'statement-alias.source-order-base36.v2','Extractor':'poppler-25.04.0/bbox-v1','Scorer':'statement-screening.v1.19-20-23','Input':100000,'Output':16000,'Bytes':90000,'Spans':2000,'CallSeconds':120,'RunSeconds':420,'Calls':3,'CeilingMicros':161700}
FIXTURES = {'columns':'097c3d35d4dd9787457b13f61bd524ff83c18d969ce50b93f7c8cecfeb569c8c','cards':'d4dd02153724c1599d4d89eb073e8cfaaf3fbce1696a4d691b69d150d0ccdce6','turnover-roles-ru':'648ac4477b9a78a7a1bb43eb9ae918fd5b6a2eb338e2ee07038146141781686f'}
ROW = ('operationDate','bookingDate','accountAmount','accountCurrency','transactionAmount','transactionCurrency','description','directionEvidence')
HEADER = ('openingBookDate','closingBookDate','openingBookBalance','closingBookBalance','availableBalanceObserved','debitTurnover','creditTurnover','currency')
ROLES = ('openingBookBalance','closingBookBalance','availableBalanceObserved','debitTurnover','creditTurnover')
REASONS = set('account_amount_required aggregate_overflow ambiguous_local_time ambiguous_money amount_overflow available_is_not_book_balance book_balance_mismatch book_balance_not_checked book_balance_period_not_proven currency_conflict currency_required description_evidence_limit direction_conflict direction_required field_evidence_limit financial_candidate_in_description historical_anchor_not_proven invalid_currency invalid_date invalid_direction invalid_page_review invalid_source_rows invalid_span_reference invalid_timezone missing_evidence negative_turnover no_financial_confirmation noncontiguous_date_evidence noncontiguous_evidence operation_date_required overlapping_evidence owner_field_role_review_required owner_page_review_required printed_turnover_mismatch service_reason_required source_coverage_incomplete statement_row_limit transaction_direction_conflict turnover_not_checked turnover_role_unverified unsupported_account_currency zero_posting field_label_required'.split())
SCORES = set('row_count account_amount_currency direction operation_clock booking_clock record_description row_issues fx_column turnover coverage blockers absent_book_balances available_balance'.split())


def require(ok):
    if not ok:
        raise ValueError('unverified')


def strict_pairs(pairs):
    out = {}
    for key, value in pairs:
        require(key not in out)
        out[key] = value
    return out


def decode(raw):
    data = json.loads(raw, object_pairs_hook=strict_pairs, parse_constant=lambda _: require(False))
    def bounded(value, depth=0):
        require(depth <= 20)
        if isinstance(value,dict):
            require(len(value)<=80)
            for v in value.values(): bounded(v,depth+1)
        elif isinstance(value,list):
            require(len(value)<=5000)
            for v in value: bounded(v,depth+1)
        elif isinstance(value,str): require(len(value)<=10000)
    bounded(data)
    return data


def stamp(value):
    require(isinstance(value,str))
    t = dt.datetime.fromisoformat(value.replace('Z','+00:00'))
    require(t.tzinfo is not None)
    return t.astimezone(dt.timezone.utc)


def selection(value):
    # Canonical Go nil selections serialize as null; preserve their empty semantics.
    if value is None: return []
    require(isinstance(value,list) and len(value)<=2000)
    require(all(isinstance(x,str) and re.fullmatch(r'p[1-9][0-9]{0,2}-b[1-9][0-9]{0,3}-l[1-9][0-9]{0,3}-w[1-9][0-9]{0,3}',x) for x in value))
    return value


def closed_reason(value):
    if not isinstance(value,str): return 'unknown'
    if value in REASONS: return value
    prefix, sep, code = value.partition(':')
    if sep and code in REASONS and (prefix in HEADER or prefix=='service' or re.fullmatch(r'row_[1-9][0-9]{0,3}',prefix)):
        return prefix+':'+code
    if re.fullmatch(r'unmapped_spans:[0-9]{1,4}',value): return value
    return 'unknown'


def project(data, now):
    require(data.get('version')=='statement.screening.v3.gemini31-identity' and data.get('policy')==POLICY and data.get('profile')==PROFILE)
    require(data.get('maximumMicros')==161700 and data.get('passed') is False and data.get('error')=='qualification_quality_failed' and not data.get('failure'))
    start,finish=stamp(data['time']),stamp(data['screeningFinishedAt'])
    require(start<=finish<=now)
    calls,traces,results,corpus = (data.get(k) for k in ('calls','traces','results','corpus'))
    require(all(isinstance(v,list) for v in (calls,traces,results,corpus)))
    require(len(calls)==len(traces)==len(results)==1 and len(corpus)==3)
    route=calls[0].get('screeningRoute') or {}; proof=route.get('proof') or {}
    require(route.get('selectedModelClass') in ('alias','pinned_canonical') and proof.get('alias')==MODEL and proof.get('canonical')=='google/gemini-3.1-flash-lite-20260507' and proof.get('provider')==POLICY['provider'])
    require(isinstance(proof.get('catalogSha256'),str) and re.fullmatch('[0-9a-f]{64}',proof['catalogSha256']))
    observed,verified=stamp(route['observedAt']),stamp(proof['verifiedAt'])
    require(start<=observed<=finish and 0<=(observed-verified).total_seconds()<=600)
    safe_corpus=[]
    for case,name in zip(corpus,FIXTURES):
        require(case.get('name')==name and case.get('fixtureHash')==FIXTURES[name])
        for key in ('documentHash','planHash'):
            require(isinstance(case.get(key),str) and re.fullmatch('[0-9a-f]{64}',case[key]))
        require(type(case.get('inputBytes')) is int and 0<case['inputBytes']<=90000)
        safe_corpus.append({key:case[key] for key in ('name','fixtureHash','documentHash','planHash','inputBytes')})
    result=results[0]
    require(result.get('name')=='columns' and result.get('passed') is False and result.get('checks')==19 and result.get('correct')==17 and result.get('rows')==2)
    for field in ('codes','validation','coverage'):
        require(isinstance(result.get(field),list) and len(result[field])<=100)
    require(result['codes']==['record_description','blockers'])
    safe_result={k:result[k] for k in ('name','rows','checks','correct','passed','codes')}
    for field in ('validation','coverage'):
        safe_result[field]=[closed_reason(v) for v in result[field]]
    require(traces[0].get('name')=='columns' and isinstance(traces[0].get('parts'),list) and len(traces[0]['parts'])==1)
    part=traces[0]['parts'][0]
    require(isinstance(part,dict) and set(part)=={'rows','header','service'})
    require(isinstance(part['rows'],list) and len(part['rows'])==2)
    rows=[]
    for row in part['rows']:
        require(isinstance(row,dict) and set(row)==set(ROW)|{'direction'})
        require(row['direction'] in ('','debit','credit'))
        rows.append(dict({k:selection(row[k]) for k in ROW},direction=row['direction']))
    h=part['header'];require(isinstance(h,dict) and set(h)==set(HEADER)|{'labels'})
    labels=h['labels'] or {};require(isinstance(labels,dict) and set(labels)<=set(ROLES))
    header={k:selection(h[k]) for k in HEADER}
    header['labels']={k:selection(v) for k,v in labels.items()}
    require(isinstance(part['service'],list) and len(part['service'])<=2000)
    service=[]
    for group in part['service']:
        require(isinstance(group,dict) and set(group)=={'spans','reason'})
        reason=group['reason'];require(isinstance(reason,str) and bool(reason.strip()) and len(reason.encode())<=2000)
        service.append({'spans':selection(group['spans']),'reason':'Reviewed fixed synthetic service grouping'})
    return {'version':'statement.s19.projection.v1','expectedSource':True,'reportIdentityVerified':True,'serviceReasonsValidated':True,'corpus':safe_corpus,'result':safe_result,'parts':[{'rows':rows,'header':header,'service':service}]}


def read_private(path,limit):
    fd=os.open(path,os.O_RDONLY|os.O_NOFOLLOW)
    with os.fdopen(fd,'rb') as f:
        info=os.fstat(f.fileno())
        require(stat.S_ISREG(info.st_mode) and stat.S_IMODE(info.st_mode)==0o600)
        raw=f.read(limit+1)
    require(len(raw)<=limit)
    return raw,(info.st_ino,info.st_size,info.st_mode,info.st_mtime_ns,hashlib.sha256(raw).digest())


def main():
    try:
        # Existing operator lock opened read-only: do not create/truncate anything.
        fd=os.open(ROOT/'statement-screening.lock',os.O_RDONLY|os.O_NOFOLLOW)
        with os.fdopen(fd,'rb') as lock:
            require(stat.S_ISREG(os.fstat(lock.fileno()).st_mode))
            fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
            marker=ROOT/(RUN+'.started');report=ROOT/(RUN+'.json')
            marker_raw,mb=read_private(marker,128)
            require(marker_raw.decode().strip()==SOURCE)
            raw,rb=read_private(report,524288)
            out=project(decode(raw),dt.datetime.now(dt.timezone.utc))
            require(read_private(marker,128)[1]==mb and read_private(report,524288)[1]==rb)
            out['artifactsUnchanged']=True
            encoded=json.dumps(out,separators=(',',':'))
            require(len(encoded.encode())<=524288)
            print(encoded)
        return 0
    except Exception:
        print('{"error":"s19_diagnostic_unavailable","verified":false}')
        return 1


if __name__=='__main__':
    sys.exit(main())
