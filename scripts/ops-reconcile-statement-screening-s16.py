#!/usr/bin/env python3
"""Сверка единственного S1.6: только GET metadata и точный учёт, без inference."""
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
import stat
import uuid
import urllib.error
import urllib.parse
import urllib.request

RUN = 's16-gemini31-diag-20261021'
SOURCE = '9e89ae08b7ec0b2b60d642b7f2c3d29a7f52598f'
BASE = Path('/opt/grace/shared')
MODEL = 'google/gemini-3.1-flash-lite'
CANONICAL_MODEL = 'google/gemini-3.1-flash-lite-20260507'
PROFILE = {'Planner': 'statement-document-atom.v2.lexical-grounding', 'Prompt': 'statement-span-mapping.v12.turnover-roles', 'Schema': '52a36f2b6159b7e9a3e2fc0509ba8c688142eebd12b60be1556d7165da56f0bf', 'Redaction': 'statement-allowlist.v4.financial-labels', 'Alias': 'statement-alias.source-order-base36.v2', 'Extractor': 'poppler-25.04.0/bbox-v1', 'Scorer': 'statement-screening.v1.19-20-23', 'Input': 100000, 'Output': 16000, 'Bytes': 90000, 'Spans': 2000, 'CallSeconds': 120, 'RunSeconds': 420, 'Calls': 3, 'CeilingMicros': 161700}

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
    parsed = dt.datetime.fromisoformat(value.replace('Z', '+00:00'))
    require(parsed.tzinfo is not None, 'timestamp_timezone_missing')
    return parsed.astimezone(dt.timezone.utc)

def micros(value):
    require(value is not None and not isinstance(value, bool), 'generation_cost_unknown')
    n = decimal.Decimal(str(value))
    require(n.is_finite() and n >= 0 and n <= 10, 'generation_cost_invalid')
    require(n == 0 or n.adjusted() >= -1000, 'generation_cost_precision_invalid')
    with decimal.localcontext() as context:
        context.prec = max(28, len(n.as_tuple().digits)+6)
        return int((n*1000000).to_integral_value(rounding=decimal.ROUND_CEILING))

def canonical_model(catalog):
    matches = [m for m in catalog.get('data', []) if m.get('id') == MODEL]
    require(len(matches) == 1 and matches[0].get('canonical_slug') == CANONICAL_MODEL, 'canonical_model_mapping_unverified')
    return CANONICAL_MODEL

def validate_report_row(report, row):
    require(report.get('version') == 'statement.screening.v2.gemini31-diagnostics' and report.get('error') == 'qualification_provider_failed_selected_model' and report.get('passed') is False and report.get('actualMicros') is None, 'unexpected_qualification_report')
    require(report.get('policy', {}).get('model') == 'google/gemini-3.1-flash-lite' and report['policy'].get('provider') == 'google-vertex/eu' and report['policy'].get('providerName') == 'Google', 'report_policy_mismatch')
    correlation(report)
    require(report.get('profile') == PROFILE, 'report_profile_mismatch')
    calls = report.get('calls')
    require(calls == [] and report.get('results') == [], 'generation_set_mismatch')
    require(report.get('maximumMicros') == 161700 and report['policy'].get('inputMicrosPerMillion') == 275000 and report['policy'].get('outputMicrosPerMillion') == 1650000, 'screening_profile_mismatch')
    require(row['purpose'] == 'statement' and row['model'] == report['policy']['model'] and row['provider'] == report['policy']['provider'], 'usage_policy_mismatch')
    require(valid_uuid(row['id']) and valid_uuid(row['owner_id']), 'usage_identity_invalid')
    require(row['generation_id'] == 'statement-qualification:'+row['id'] and row['status'] in ('uncertain', 'settled'), 'usage_marker_mismatch')
    start = timestamp(report['time'])
    require(abs((timestamp(row['created_at'])-start).total_seconds()) <= 5 and row['reserved_micros'] == report['maximumMicros'], 'usage_report_time_mismatch')

def validate(report, row, ids, data, verified_canonical=None):
    validate_report_row(report, row)
    require(ids == [correlation(report)], 'report_correlation_mismatch')
    start = timestamp(report['time'])
    total = 0
    for ident in ids:
        d = data[ident]
        require(d.get('id') == ident and (d.get('model') == report['policy']['model'] or verified_canonical == CANONICAL_MODEL and d.get('model') == verified_canonical) and d.get('provider_name') == 'Google' and d.get('is_byok') is False, 'generation_route_mismatch')
        require(-2 <= (timestamp(d['created_at'])-start).total_seconds() <= 120, 'generation_time_mismatch')
        cost = micros(d.get('total_cost'))
        total += cost
    require(total <= 53900, 'actual_exceeds_single_call_review')
    if row['status'] == 'settled':
        require(row['actual_micros'] == total, 'settlement_conflict')
    return total

def valid_uuid(value):
    try:
        return isinstance(value, str) and str(uuid.UUID(value)) == value
    except (ValueError, AttributeError):
        return False


def private_read(path, limit=524288):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, 'rb') as f:
        info = os.fstat(f.fileno())
        require(stat.S_ISREG(info.st_mode) and stat.S_IMODE(info.st_mode) == 0o600, 'artifact_not_private')
        raw = f.read(limit+1)
    require(len(raw) <= limit, 'artifact_too_large')
    return raw


def correlation(report):
    failure = report.get('failure') or {}
    ident = failure.get('unverifiedCorrelationId')
    require(failure.get('code') == 'selected_model' and type(failure.get('httpStatus')) is int and failure['httpStatus'] == 200, 'unexpected_failure_diagnostic')
    require(isinstance(ident, str) and len(ident) <= 128 and re.fullmatch(r'gen-[0-9A-Za-z-]+', ident), 'invalid_generation_id')
    return ident


def save_private(path, value):
    require(not path.is_symlink(), 'audit_path_unsafe')
    tmp = str(path)+'.tmp'
    fd = os.open(tmp, os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as f:
        json.dump(value, f, default=str)
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp, path)


def audit_matches(old, audit):
    require(old.get('reportDigest') == audit['reportDigest'] and old.get('actualMicros') == audit['actualMicros'] and old.get('reportName') == audit['reportName'] and set(old.get('generations', {})) == set(audit['generations']), 'audit_conflict')
    for field in ('id', 'owner_id', 'purpose', 'model', 'provider', 'reserved_micros', 'generation_id', 'created_at'):
        require(old.get('usageBefore', {}).get(field) == audit['usageBefore'][field], 'audit_usage_conflict')


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    os.umask(0o077)
    lock_fd = os.open(BASE/'statement-screening.lock', os.O_WRONLY|os.O_CREAT|os.O_NOFOLLOW, 0o600)
    lock = os.fdopen(lock_fd, 'w')
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    paths = [BASE/('statement-screening-'+RUN+'.json')]
    marker = BASE/('statement-screening-'+RUN+'.started')
    require(private_read(marker, 128).decode().strip() == SOURCE, 'screening_source_mismatch')
    raw = private_read(paths[0])
    report = json.loads(raw)
    ids = [correlation(report)]
    start = timestamp(report['time']).isoformat()
    api, postgres = container('api'), container('postgres')
    def sql(query):
        return command(['docker', 'exec', '-i', postgres, 'psql', '-X', '-v', 'ON_ERROR_STOP=1', '-U', 'grace', '-d', 'grace', '-At'], query)
    rows = json.loads(sql("SELECT COALESCE(json_agg(u),'[]') FROM ai_usage u WHERE purpose='statement' AND model='"+MODEL+"' AND provider='google-vertex/eu' AND reserved_micros=161700 AND generation_id LIKE 'statement-qualification:%' AND created_at BETWEEN '"+start+"'::timestamptz-interval '5 seconds' AND '"+start+"'::timestamptz+interval '5 seconds';"))
    require(len(rows) == 1, 'usage_row_not_unique')
    row = rows[0]
    validate_report_row(report, row)
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
    evidence_path = BASE/('statement-generation-inspection-'+RUN+'.json')
    save_private(evidence_path, {'reportDigest':hashlib.sha256(raw).hexdigest(), 'generations':data, 'verifiedAt':dt.datetime.now(dt.timezone.utc).isoformat()})
    with opener.open('https://openrouter.ai/api/v1/models', timeout=15) as response:
        catalog_raw = response.read((8<<20)+1)
    require(len(catalog_raw) <= 8<<20, 'model_catalog_too_large')
    verified_canonical = canonical_model(json.loads(catalog_raw))
    total = validate(report, row, ids, data, verified_canonical)
    audit_path = BASE/('statement-reconciliation-'+RUN+'.json')
    audit = {'version':'statement-reconciliation.v1', 'canonicalModelMapping':{'model':MODEL,'canonicalModel':verified_canonical,'catalogSha256':hashlib.sha256(catalog_raw).hexdigest()}, 'reportDigest':hashlib.sha256(raw).hexdigest(), 'reportName':paths[0].name, 'usageBefore':row, 'generations':data, 'actualMicros':total, 'verifiedAt':dt.datetime.now(dt.timezone.utc).isoformat(), 'committed':False}
    old = None
    if audit_path.exists() or audit_path.is_symlink():
        old = json.loads(private_read(audit_path, 2<<20))
        audit_matches(old, audit)
        if old.get('committed'):
            require(row['status'] == 'settled', 'audit_state_conflict')
            ready = sql("SELECT NOT EXISTS(SELECT 1 FROM ai_usage WHERE status='uncertain');") == 't'
            print(json.dumps({'billingReconciled':True,'idempotentReplay':True,'sharedUncertainBlockCleared':ready,'newInferenceAuthorized':False}))
            return
    require(not args.apply or old is not None, 'read_only_inspection_required')
    if old is not None:
        audit['usageBefore'] = old['usageBefore']
        audit['inspectionVerifiedAt'] = old['verifiedAt']
    def save():
        save_private(audit_path, audit)
    save()
    finish = data[ids[0]].get('finish_reason')
    safe_finish = finish if finish in ('stop', 'length', 'error', 'content_filter') else 'other'
    print(json.dumps({'metadataMatched':True, 'unknownBillingVerified':True, 'failedCallFinishReason':safe_finish}))
    if args.apply:
        uid, owner = row['id'], row['owner_id']
        # Существующая функция блокирует общий ledger и проверяет owner/replay.
        sql("BEGIN; SET LOCAL lock_timeout='5s'; SELECT singleton FROM ai_experiment_budget WHERE singleton FOR UPDATE; SELECT set_config('grace.owner_id','"+owner+"',true); SELECT id FROM ai_usage WHERE id='"+uid+"'::uuid FOR UPDATE; DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM ai_usage WHERE id='"+uid+"'::uuid AND owner_id='"+owner+"'::uuid AND purpose='statement' AND model='"+MODEL+"' AND provider='google-vertex/eu' AND reserved_micros=161700 AND generation_id='statement-qualification:"+uid+"' AND created_at='"+timestamp(row['created_at']).isoformat()+"'::timestamptz AND (status='uncertain' OR status='settled' AND actual_micros="+str(total)+")) THEN RAISE EXCEPTION 'usage_changed'; END IF; END $$; SET LOCAL ROLE grace_financial; SELECT grace_ai_settle('"+uid+"'::uuid,"+str(total)+",'statement-qualification:"+uid+"'); COMMIT;")
        after = json.loads(sql("SELECT row_to_json(u) FROM ai_usage u WHERE id='"+uid+"'::uuid;"))
        require(after['id'] == row['id'] and after['owner_id'] == row['owner_id'] and after['reserved_micros'] == row['reserved_micros'] and after['status'] == 'settled' and after['actual_micros'] == total and after['generation_id'] == row['generation_id'], 'settlement_not_verified')
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
