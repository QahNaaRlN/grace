"""S1.18 sealed-memfd bootstrap and strict closed output schema. No disk writes."""
import base64
import fcntl
import hashlib
import json
import os
import re
import subprocess
import sys

LIMIT = 32 * 1024 * 1024
ERROR = '{"error":"diagnostic_unavailable"}'
FAILURES = set('snapshot_unavailable context_mismatch replay_mismatch guard_mismatch unsupported_reason_code diagnosis_limit_exceeded unavailable_runtime'.split())
RELATIONS = {'missing','matches','differs','unavailable'}
CODES = set('missing_evidence invalid_span_reference noncontiguous_evidence noncontiguous_date_evidence field_evidence_limit overlapping_evidence operation_date_required invalid_date invalid_timezone ambiguous_local_time account_amount_required currency_required currency_conflict invalid_currency ambiguous_money amount_overflow unsupported_account_currency zero_posting invalid_direction direction_conflict direction_required transaction_direction_conflict description_evidence_limit financial_candidate_in_description description_evidence_gap description_evidence_unverified field_label_required available_is_not_book_balance negative_turnover turnover_role_unverified book_balance_role_unverified aggregate_overflow invalid_source_rows printed_turnover_mismatch book_balance_mismatch book_balance_period_not_proven source_coverage_incomplete unmapped_spans invalid_page_review owner_page_review_required owner_field_role_review_required service_reason_required'.split())
CHECKS = set('row_count account_amount_currency direction operation_clock booking_clock record_description row_issues fx_column turnover coverage blockers absent_book_balances available_balance book_balances opening_book_value closing_book_value absent_booking absent_fx printed_debit_turnover printed_credit_turnover absent_book_dates'.split())
FIELDS = set('openingBookDate closingBookDate openingBookBalance closingBookBalance availableBalanceObserved debitTurnover creditTurnover service'.split())
PUBLIC = set('row_count account_amount_currency direction operation_clock booking_clock record_description row_issues fx_column turnover coverage blockers absent_book_balances available_balance book_balances opening_book_value closing_book_value absent_booking absent_fx printed_debit_turnover printed_credit_turnover absent_book_dates account_amount_required aggregate_overflow ambiguous_local_time ambiguous_money amount_overflow available_is_not_book_balance book_balance_mismatch book_balance_not_checked book_balance_period_not_proven currency_conflict currency_required description_evidence_limit direction_conflict direction_required field_evidence_limit financial_candidate_in_description historical_anchor_not_proven invalid_currency invalid_date invalid_direction invalid_span_reference invalid_timezone legacy_field_evidence_invalid legacy_service_mapping_missing mapping_review_missing mapping_snapshot_invalid missing_evidence no_financial_confirmation noncontiguous_date_evidence noncontiguous_evidence operation_date_required overlapping_evidence owner_field_role_review_required owner_page_review_required printed_turnover_mismatch service_reason_required source_coverage_incomplete statement_row_limit transaction_direction_conflict turnover_not_checked turnover_role_unverified unsupported_account_currency zero_posting description_evidence_gap description_evidence_unverified book_balance_role_unverified openingBookBalance:book_balance_role_unverified closingBookBalance:book_balance_role_unverified'.split())

def require(ok):
    if not ok:
        raise ValueError('unverified')

def unique(pairs):
    out = {}; seen = set()
    for key,value in pairs:
        require(key.lower() not in seen); seen.add(key.lower()); out[key] = value
    return out

def integer(n, low=0, high=2000):
    require(type(n) is int and low <= n <= high)

def public_code(raw):
    if raw in PUBLIC: return raw
    m = re.fullmatch(r'row_([1-9][0-9]{0,3}):(description_evidence_gap|description_evidence_unverified)',raw)
    if m and int(m[1]) <= 5000: return raw
    return 'unknown'

def positions(ps):
    require(type(ps) is list and len(ps) <= 128)
    previous = 0
    for p in ps:
        require(type(p) is dict and set(p) == {'sourceOrdinal','page','block','line','word','lexicalClass'})
        integer(p['sourceOrdinal'],previous+1); previous=p['sourceOrdinal']; integer(p['page'],1,50)
        for k in ('block','line','word'): integer(p[k],1,1000000)
        require(p['lexicalClass'] in ('numeric','other'))

def canonical_optionals(value):
    if isinstance(value,list):
        for child in value: canonical_optionals(child)
    elif isinstance(value,dict):
        for key,child in value.items():
            if key in ('positions','checks','reasons','links'): require(type(child) is list and len(child)>0)
            if key=='rowOrdinal': integer(child,1)
            if key in ('reason','field','publicCode','relation'): require(type(child) is str and child!='')
            require(child is not None)
            canonical_optionals(child)

def projection(raw, source=None):
    require(len(raw) <= 65536)
    d=json.loads(raw,object_pairs_hook=unique)
    canonical_optionals(d)
    require(type(d) is dict and set(d)=={'version','authorization','artifactSource','verifierSource','contextVerified','artifactsUnchanged','providerCalls','persistentWritesRequested','tuples'})
    require(d['version']=='statement.saved-quality-diagnosis.s118.v1' and d['authorization']=='s117-comparison-20261023' and d['artifactSource']=='24f73223b9b3370ca78c41d34e89edca9a276488')
    require(type(d['verifierSource']) is str and re.fullmatch('[0-9a-f]{40}',d['verifierSource']))
    if source is not None: require(d['verifierSource']==source)
    require(d['contextVerified'] is True and d['artifactsUnchanged'] is True)
    for k in ('providerCalls','persistentWritesRequested'): integer(d[k],0,0)
    require(type(d['tuples']) is list and len(d['tuples'])==6)
    for i,t in enumerate(d['tuples']):
        base={'ordinal','status'}; require(type(t) is dict and base<=set(t)); integer(t['ordinal'],i+1,i+1)
        if t['status']=='UNVERIFIED':
            require(set(t)==base|{'reason'} and t['reason'] in FAILURES); continue
        require(t['status']=='VERIFIED' and set(t)<=base|{'checks','reasons','coverage'})
        require(('coverage' in t)==(i<4))
        if i<4:
            c=t['coverage'];require(type(c) is dict and {'status','totalUnmapped'}<=set(c));integer(c['totalUnmapped'])
            if c['status']=='UNVERIFIED': require(set(c)=={'status','totalUnmapped','reason'} and c['reason'] in FAILURES)
            else:
                require(c['status']=='VERIFIED' and set(c)<= {'status','totalUnmapped','positions'});positions(c.get('positions',[]));require(len(c.get('positions',[]))==c['totalUnmapped'])
        reasons=t.get('reasons',[]);require(type(reasons) is list and len(reasons)<=128);counters={}
        for r in reasons:
            base_r={'ordinal','scope','status'};require(type(r) is dict and base_r<=set(r))
            require(r['scope'] in ('row','blocker','coverage','turnover','book_balance'));integer(r.get('rowOrdinal',0))
            key=(r['scope'],r.get('rowOrdinal',0) if r['scope']=='row' else 0);counters[key]=counters.get(key,0)+1;integer(r['ordinal'],counters[key],counters[key])
            if r['status']=='UNVERIFIED':require(set(r)<=base_r|{'rowOrdinal','reason'} and r.get('reason') in FAILURES);continue
            require(r['status']=='VERIFIED' and set(r)<=base_r|{'rowOrdinal','code','field','publicCode','relation','count','positions'})
            require(r.get('code') in CODES and ('field' not in r or r['field'] in FIELDS))
            require(not(r.get('rowOrdinal',0)>0 and 'field' in r) and (r['scope']!='row' or r.get('rowOrdinal',0)>0))
            positions(r.get('positions',[]));require(('count' in r)==(r['code']=='unmapped_spans'))
            code=r['code']
            if 'count' in r:
                integer(r['count']);code+=':'+str(r['count'])
                if i<4:require(r['count']==t['coverage']['totalUnmapped'])
            if r['scope']=='blocker' and r.get('rowOrdinal',0):code='row_'+str(r['rowOrdinal'])+':'+code
            if 'field' in r:code=r['field']+':'+code
            if r['scope'] in ('blocker','coverage'):require(r.get('relation')=='mapped' and r.get('publicCode')==public_code(code))
            else: require(r.get('relation')=='not_exposed' and 'publicCode' not in r)
        checks=t.get('checks',[]);require(type(checks) is list and 1<=len(checks)<=23);previous=0
        for c in checks:
            base_c={'ordinal','code','status'};require(type(c) is dict and base_c<=set(c));integer(c['ordinal'],previous+1,(19,19,20,20,23,23)[i]);previous=c['ordinal'];integer(c.get('rowOrdinal',0),0,2)
            require(c['code'] in CHECKS)
            if c['status']=='UNVERIFIED': require(set(c)<=base_c|{'rowOrdinal','reason'} and c.get('reason') in FAILURES);continue
            require(c['status']=='VERIFIED' and set(c)<=base_c|{'rowOrdinal','positions','clock','money','state','links'})
            positions(c.get('positions',[]));last=0
            for n in c.get('links',[]):integer(n,last+1,len(reasons));last=n
            if 'clock' in c: require(type(c['clock']) is dict and set(c['clock'])=={'date','time','precision','utc'} and all(v in RELATIONS for v in c['clock'].values()))
            if 'money' in c: require(type(c['money']) is dict and set(c['money'])=={'amount','currency'} and all(v in RELATIONS for v in c['money'].values()))
            code=c['code'];row=c.get('rowOrdinal',0)
            if code in ('operation_clock','booking_clock'):require(row>0 and 'clock' in c and not(set(c)&{'money','state','links'}))
            elif code in ('printed_debit_turnover','printed_credit_turnover'):require(i>=4 and row==0 and 'money' in c and not(set(c)&{'clock','state'}))
            elif code=='row_issues':require(row>0 and c.get('links') and not(set(c)&{'clock','money','state'}))
            elif code=='coverage':require(row==0 and c.get('state') in ('matched','incomplete','not_reviewed') and not(set(c)&{'clock','money'}))
            elif code=='turnover':require(row==0 and c.get('state') in ('matched','mismatch','incomplete','not_provided') and not(set(c)&{'clock','money'}))
            elif code=='blockers':require(row==0 and c.get('links') and not(set(c)&{'clock','money','state'}))
            else:require(False)
    return d

def execute(raw):
    require(len(raw) <= LIMIT)
    data = json.loads(raw, object_pairs_hook=unique)
    require(set(data) == {'binary', 'sha256', 'context', 'verifierSource'})
    require(type(data['verifierSource']) is str and re.fullmatch('[0-9a-f]{40}',data['verifierSource']))
    binary = base64.b64decode(data['binary'], validate=True)
    require(0 < len(binary) <= 16 * 1024 * 1024)
    require(hashlib.sha256(binary).hexdigest() == data['sha256'])
    context = data['context'].encode()
    require(len(context) <= 524288)
    fd = os.memfd_create('grace-s118', os.MFD_CLOEXEC | os.MFD_ALLOW_SEALING)
    try:
        view = memoryview(binary)
        while view:
            count = os.write(fd, view)
            require(count > 0)
            view = view[count:]
        fcntl.fcntl(fd, fcntl.F_ADD_SEALS, fcntl.F_SEAL_WRITE | fcntl.F_SEAL_GROW | fcntl.F_SEAL_SHRINK | fcntl.F_SEAL_SEAL)
        result = subprocess.run(['/proc/self/fd/' + str(fd)], input=context,
                                pass_fds=(fd,), stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, env={}, timeout=45)
        require(result.returncode == 0 and len(result.stdout) <= 65536)
        # stdout is the allowlisted Go projection; never forward a failed process.
        projection(result.stdout,data['verifierSource'])
        return result.stdout
    finally:
        os.close(fd)

def main():
    try:
        raw = sys.stdin.buffer.read(LIMIT + 1)
        out = execute(raw)
        sys.stdout.buffer.write(out)
        return 0
    except Exception:
        print(ERROR)
        return 1

if __name__ == '__main__':
    sys.exit(main())
