#!/usr/bin/env python3
"""One-run recovery of our synthetic qualifier only; never print billing/IDs."""
import argparse
import datetime as dt
import decimal
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

RUN = 's14-gemini31-20261020'
BASE = Path('/opt/grace/shared')
MODEL = 'google/gemini-3.1-flash-lite'
CANONICAL_MODEL = 'google/gemini-3.1-flash-lite-20260507'
class Stop(Exception):
    pass

def require(ok, code):
    if not ok:
        raise Stop(code)

def command(args, stdin=None):
    p = subprocess.run(args, input=stdin, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=20)
    require(p.returncode == 0, 'operator_command_failed')
    return p.stdout.strip()

def container(service):
    out = command(['docker', 'ps', '-q', '--filter', 'label=com.docker.compose.project=grace-production', '--filter', 'label=com.docker.compose.service='+service]).splitlines()
    require(len(out) == 1 and re.fullmatch(r'[a-f0-9]+', out[0]), 'runtime_not_unique')
    return out[0]

def timestamp(value):
    return dt.datetime.fromisoformat(value.replace('Z', '+00:00')).astimezone(dt.timezone.utc)

def micros(value):
    require(value is not None and not isinstance(value, bool), 'generation_cost_unknown')
    n = decimal.Decimal(str(value))
    require(n.is_finite() and n >= 0 and n <= 10, 'generation_cost_invalid')
    return int((n*1000000).to_integral_value(rounding=decimal.ROUND_CEILING))

def canonical_model(catalog):
    matches = [m for m in catalog.get('data', []) if m.get('id') == MODEL]
    require(len(matches) == 1 and matches[0].get('canonical_slug') == CANONICAL_MODEL, 'canonical_model_mapping_unverified')
    return CANONICAL_MODEL

def validate(report, row, ids, data, verified_canonical=None):
    require(report.get('version') == 'statement.screening.v1.gemini31-ru-roles' and report.get('error') == 'qualification_provider_failed' and not report.get('passed') and report.get('actualMicros') is None, 'unexpected_qualification_report')
    require(report.get('policy', {}).get('model') == 'google/gemini-3.1-flash-lite' and report['policy'].get('provider') == 'google-vertex/eu' and report['policy'].get('providerName') == 'Google', 'report_policy_mismatch')
    calls = report.get('calls', [])
    known = {c['generationId']: c for c in calls}
    require(calls == [] and report.get('results') == [] and len(ids) == 1 and len(known) == 0, 'generation_set_mismatch')
    require(report.get('maximumMicros') == 161700 and report['policy'].get('inputMicrosPerMillion') == 275000 and report['policy'].get('outputMicrosPerMillion') == 1650000, 'screening_profile_mismatch')
    require(row['purpose'] == 'statement' and row['model'] == report['policy']['model'] and row['provider'] == report['policy']['provider'], 'usage_policy_mismatch')
    require(re.fullmatch(r'[0-9a-f-]{36}', row['id']) and re.fullmatch(r'[0-9a-f-]{36}', row['owner_id']), 'usage_identity_invalid')
    require(row['generation_id'] == 'statement-qualification:'+row['id'] and row['status'] in ('uncertain', 'settled'), 'usage_marker_mismatch')
    start = timestamp(report['time'])
    require(abs((timestamp(row['created_at'])-start).total_seconds()) <= 5 and row['reserved_micros'] == report['maximumMicros'], 'usage_report_time_mismatch')
    total = 0
    for ident in ids:
        d = data[ident]
        require(d.get('id') == ident and (d.get('model') == report['policy']['model'] or verified_canonical == CANONICAL_MODEL and d.get('model') == verified_canonical) and d.get('provider_name', '').lower() == 'google' and d.get('is_byok') is False, 'generation_route_mismatch')
        require(-2 <= (timestamp(d['created_at'])-start).total_seconds() <= 120, 'generation_time_mismatch')
        cost = micros(d.get('total_cost'))
        if ident in known:
            require(cost == known[ident]['costMicros'], 'known_cost_conflict')
        total += cost
    require(total <= 53900, 'actual_exceeds_single_call_review')
    if row['status'] == 'settled':
        require(row['actual_micros'] == total, 'settlement_conflict')
    return total

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    ids = sys.stdin.read(1000).split()
    require(len(ids) == 1 and len(set(ids)) == 1 and all(re.fullmatch(r'gen-[0-9]+-[A-Za-z0-9]+', s) for s in ids), 'invalid_generation_ids')
    os.umask(0o077)
    lock = open(BASE/'statement-screening.lock', 'a')
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    paths = [BASE/('statement-screening-'+RUN+'.json')]
    marker = BASE/('statement-screening-'+RUN+'.started')
    require(not marker.is_symlink() and marker.read_text().strip() == 'e36dbcbb5c57ed8b2060cc2bcbc1ebc8482fd728', 'screening_source_mismatch')
    require(len(paths) == 1 and paths[0].is_file() and not paths[0].is_symlink() and paths[0].stat().st_mode & 0o077 == 0, 'qualification_report_not_unique_or_private')
    raw = paths[0].read_bytes()
    report = json.loads(raw)
    api, postgres = container('api'), container('postgres')
    def sql(query):
        return command(['docker', 'exec', '-i', postgres, 'psql', '-X', '-v', 'ON_ERROR_STOP=1', '-U', 'grace', '-d', 'grace', '-At'], query)
    rows = json.loads(sql("SELECT COALESCE(json_agg(u),'[]') FROM ai_usage u WHERE purpose='statement' AND generation_id LIKE 'statement-qualification:%' AND created_at BETWEEN '2026-10-20T16:41:00Z'::timestamptz AND '2026-10-20T16:42:30Z'::timestamptz;"))
    require(len(rows) == 1, 'usage_row_not_unique')
    row = rows[0]
    key = command(['docker', 'exec', api, 'cat', '/run/secrets/openrouter_api_key'])
    require(key and '\n' not in key, 'provider_key_unavailable')
    opener = urllib.request.build_opener(NoRedirect)
    data = {}
    for ident in ids:
        req = urllib.request.Request('https://openrouter.ai/api/v1/generation?'+urllib.parse.urlencode({'id': ident}), headers={'Authorization':'Bearer '+key})
        with opener.open(req, timeout=15) as response:
            payload = response.read(1048577)
        require(len(payload) <= 1048576, 'generation_metadata_too_large')
        data[ident] = json.loads(payload, parse_float=decimal.Decimal)['data']
    # Preserve authoritative metadata privately even when identity validation
    # refuses settlement. Public output contains only match booleans.
    evidence_path = BASE/('statement-generation-inspection-'+RUN+'.json')
    require(not evidence_path.is_symlink(), 'inspection_path_unsafe')
    tmp = str(evidence_path)+'.tmp'
    fd = os.open(tmp, os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as f:
        json.dump({'reportDigest':hashlib.sha256(raw).hexdigest(), 'generations':data, 'verifiedAt':dt.datetime.now(dt.timezone.utc).isoformat()}, f, default=str)
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp, evidence_path)
    d = data[ids[0]]
    provider = d.get('provider_name')
    print(json.dumps({'generationIdentityInspection':True,
        'idMatches':d.get('id') == ids[0],
        'modelAliasMatches':d.get('model') == report['policy']['model'],
        'modelCanonicalMatches':d.get('model') == 'google/gemini-3.1-flash-lite-20260507',
        'providerGoogleMatches':isinstance(provider,str) and provider.lower() == 'google',
        'providerMissing':provider is None or provider == '',
        'providerGoogleAIStudio':provider == 'Google AI Studio',
        'nonByokMatches':d.get('is_byok') is False,
        'costPresent':d.get('total_cost') is not None,
        'costZero':d.get('total_cost') is not None and decimal.Decimal(str(d['total_cost'])) == 0,
        'generationWithinWindow':-2 <= (timestamp(d['created_at'])-timestamp(report['time'])).total_seconds() <= 120,
        'finishError':d.get('finish_reason') == 'error'}))
    with opener.open('https://openrouter.ai/api/v1/models', timeout=15) as response:
        catalog_raw = response.read((8<<20)+1)
    require(len(catalog_raw) <= 8<<20, 'model_catalog_too_large')
    verified_canonical = canonical_model(json.loads(catalog_raw))
    total = validate(report, row, ids, data, verified_canonical)
    audit_path = BASE/('statement-reconciliation-'+RUN+'.json')
    audit = {'version':'statement-reconciliation.v1', 'canonicalModelMapping':{'model':MODEL,'canonicalModel':verified_canonical,'catalogSha256':hashlib.sha256(catalog_raw).hexdigest()}, 'reportDigest':hashlib.sha256(raw).hexdigest(), 'reportName':paths[0].name, 'usageBefore':row, 'generations':data, 'actualMicros':total, 'verifiedAt':dt.datetime.now(dt.timezone.utc).isoformat(), 'committed':False}
    if audit_path.exists():
        require(not audit_path.is_symlink() and audit_path.stat().st_mode & 0o077 == 0, 'audit_not_private')
        old = json.loads(audit_path.read_text())
        require(old['reportDigest'] == audit['reportDigest'] and old['actualMicros'] == total and set(old['generations']) == set(ids), 'audit_conflict')
        if old.get('committed'):
            require(row['status'] == 'settled', 'audit_state_conflict')
            print('{"billingReconciled":true,"idempotentReplay":true}')
            return
    def save():
        tmp = str(audit_path)+'.tmp'
        fd = os.open(tmp, os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, 'w') as f:
            json.dump(audit, f, default=str)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, audit_path)
    save()
    unknown = next(ident for ident in ids if ident not in {c['generationId'] for c in report['calls']})
    finish = data[unknown].get('finish_reason')
    safe_finish = finish if finish in ('stop', 'length', 'error', 'content_filter') else 'other'
    print(json.dumps({'metadataMatched':True, 'unknownBillingVerified':True, 'failedCallFinishReason':safe_finish}))
    if args.apply:
        uid, owner = row['id'], row['owner_id']
        # No budget/configuration/financial mutation. Existing function locks the
        # shared ledger, scopes owner, and detects conflicting settled replays.
        sql("BEGIN; SET LOCAL lock_timeout='5s'; SELECT set_config('grace.owner_id','"+owner+"',true); SET LOCAL ROLE grace_financial; SELECT grace_ai_settle('"+uid+"'::uuid,"+str(total)+",'statement-qualification:"+uid+"'); COMMIT;")
        after = json.loads(sql("SELECT row_to_json(u) FROM ai_usage u WHERE id='"+uid+"'::uuid;"))
        require(after['status'] == 'settled' and after['actual_micros'] == total and after['generation_id'] == row['generation_id'], 'settlement_not_verified')
        audit['committed'] = True
        audit['usageAfter'] = after
        audit['committedAt'] = dt.datetime.now(dt.timezone.utc).isoformat()
        save()
        ready = sql("SELECT NOT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain');") == 't'
        print(json.dumps({'billingReconciled':True,'sharedUncertainBlockCleared':ready,'newInferenceAuthorized':False}))

if __name__ == '__main__':
    try:
        main()
    except Stop as e:
        print(json.dumps({'error':str(e)}))
        sys.exit(1)
    except Exception:
        print('{"error":"reconciliation_failed_private_details_retained"}')
        sys.exit(1)
