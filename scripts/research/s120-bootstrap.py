"""S1.20 fixed sealed-memfd reader and independently checked closed output."""
import base64
import fcntl
import hashlib
import json
import os
import re
import selectors
import subprocess
import sys
import time

VERSION = 'statement.clock-diagnosis.s120.v1'
PRODUCER = '8da7f45b105105d6596032850893bb690febebd8'
RUN = 39983522098
CONTEXT_SHA = 'eefde6959b1bbfe8c94e4559fee1103dfeac98d10c0f415f8673d97b661cdbdf'
LIMIT = 32 * 1024 * 1024
FAILURES = set('snapshot_unavailable source_mismatch context_mismatch report_invalid snapshot_changed summary_mismatch unsupported_check_code diagnosis_limit_exceeded output_mismatch unavailable_runtime'.split())
CODES = 'row_count account_amount_currency direction operation_clock booking_clock record_description row_issues fx_column account_amount_currency direction operation_clock booking_clock record_description row_issues turnover coverage blockers book_balances opening_book_value closing_book_value'.split()
ROWS = [0]+[1]*7+[2]*6+[0]*6
ORDER = ['base','candidate','candidate','base','base','candidate']
COUNTS = [0,4,4,0,0,4]
CLOCKS = {'operation_clock','booking_clock'}


def require(ok):
    if not ok:
        raise ValueError('unverified')


def unique(pairs):
    out = {}; seen = set()
    for key, value in pairs:
        require(key.lower() not in seen)
        seen.add(key.lower()); out[key] = value
    return out


def shape(value, keys):
    require(type(value) is dict and set(value) == set(keys.split()))


def integer(value, low, high):
    require(type(value) is int and low <= value <= high)


def boolean(value):
    require(type(value) is bool)


def source_ok(source):
    return type(source) is str and re.fullmatch('[0-9a-f]{40}', source) is not None


def failure(source, reason='unavailable_runtime'):
    return json.dumps(dict(version=VERSION, readerSource=source, status='UNVERIFIED', reason=reason, activationAuthorized=False), separators=(',', ':')).encode()+b'\n'


def manifest(source):
    return dict(version='statement.clock-diagnosis.manifest.s120.v1', readerSource=source, producerSource=PRODUCER, producerRun=RUN, contextSha256=CONTEXT_SHA, providerCalls=0, sqlCalls=0, persistentWritesRequested=0, activationAuthorized=False)


def projection(raw, source):
    require(source_ok(source) and len(raw) <= 65536)
    d = json.loads(raw.decode('utf-8'), object_pairs_hook=unique)
    require(type(d) is dict and d.get('version') == VERSION and d.get('readerSource') == source and d.get('activationAuthorized') is False)
    if d.get('status') == 'UNVERIFIED':
        shape(d, 'version readerSource status reason activationAuthorized')
        require(d['reason'] in FAILURES)
        return d
    shape(d, 'version readerSource producerSource reportSha256 reportDigest producerRun status replayVerified decision activationAuthorized attempts pairs rejectReasons accounting providerCalls sqlCalls persistentWritesRequested')
    require(d['status'] == 'VERIFIED' and d['producerSource'] == PRODUCER and d['decision'] == 'REJECT')
    integer(d['producerRun'], RUN, RUN); integer(d['replayVerified'], 6, 6)
    for key in ('providerCalls', 'sqlCalls', 'persistentWritesRequested'): integer(d[key], 0, 0)
    for key in ('reportSha256', 'reportDigest'): require(type(d[key]) is str and re.fullmatch('[0-9a-f]{64}', d[key]))
    attempts = d['attempts']; require(type(attempts) is list and len(attempts) == 6)
    for i, a in enumerate(attempts):
        shape(a, 'ordinal variant clockCount precision qualityPassed correct checksCount checks')
        integer(a['ordinal'], i+1, i+1); require(a['variant'] == ORDER[i]); integer(a['checksCount'],20,20)
        require(type(a['precision']) is list and len(a['precision']) == 4)
        for value in a['precision']: boolean(value)
        boolean(a['qualityPassed']); require(type(a['checks']) is list and len(a['checks']) == 20)
        count = correct = 0
        for j, c in enumerate(a['checks']):
            shape(c, 'ordinal rowOrdinal code passed')
            integer(c['ordinal'],j+1,j+1);integer(c['rowOrdinal'],ROWS[j],ROWS[j]); require(c['code'] == CODES[j]); boolean(c['passed'])
            if c['passed']:
                correct += 1
                count += int(c['code'] in CLOCKS)
        integer(a['clockCount'],count,count); require(count == COUNTS[i]); integer(a['correct'],correct,correct)
        require(a['qualityPassed'] == (correct == 20))
    pairs = []; target = precision = regression = False
    for i in range(0,6,2):
        b,c = attempts[i:i+2]
        if b['variant'] == 'candidate': b,c = c,b
        regressions = []
        for x,y in zip(b['checks'],c['checks']):
            if x['code'] not in CLOCKS and x['passed'] and not y['passed']:
                regressions.append(dict(ordinal=x['ordinal'],rowOrdinal=x['rowOrdinal'],code=x['code'],basePassed=True,candidatePassed=False))
        failures = [dict(rowOrdinal=k//2+1,field=('operation','booking')[k%2]) for k,ok in enumerate(c['precision']) if not ok]
        clock_failed = c['clockCount'] != 4
        pairs.append(dict(pairOrdinal=i//2+1,baseOrdinal=b['ordinal'],candidateOrdinal=c['ordinal'],nonClockRegressions=regressions,precisionFailures=failures,candidateClockTargetFailed=clock_failed))
        target |= clock_failed; precision |= bool(failures); regression |= bool(regressions)
    reasons = [code for enabled,code in [(target,'candidate_clock_target_failed'),(precision,'candidate_precision_failed'),(regression,'non_clock_regression')] if enabled]
    require(reasons and d['rejectReasons'] == reasons)
    # JSON equality alone considers true == 1 in Python. Require primitive types
    # before comparing the complete, independently reconstructed pair arrays.
    require(type(d['pairs']) is list and len(d['pairs']) == 3)
    for p in d['pairs']:
        shape(p,'pairOrdinal baseOrdinal candidateOrdinal nonClockRegressions precisionFailures candidateClockTargetFailed')
        for key in ('pairOrdinal','baseOrdinal','candidateOrdinal'): integer(p[key],1,6)
        boolean(p['candidateClockTargetFailed'])
        require(type(p['nonClockRegressions']) is list and type(p['precisionFailures']) is list)
        for r in p['nonClockRegressions']:
            shape(r,'ordinal rowOrdinal code basePassed candidatePassed'); integer(r['ordinal'],1,20); integer(r['rowOrdinal'],0,2);boolean(r['basePassed']);boolean(r['candidatePassed'])
        for f in p['precisionFailures']:
            shape(f,'rowOrdinal field');integer(f['rowOrdinal'],1,2)
    require(d['pairs'] == pairs)
    a = d['accounting'];shape(a,'callCostsMicros actualMicros maximumMicros billingKnown settlementRecorded')
    require(type(a['callCostsMicros']) is list and len(a['callCostsMicros']) == 6)
    for n in a['callCostsMicros']: integer(n,0,323400)
    total = sum(a['callCostsMicros']);integer(a['actualMicros'],total,total);integer(a['maximumMicros'],323400,323400)
    require(total <= 323400 and a['billingKnown'] is True and a['settlementRecorded'] is True)
    return d


def capture(args, payload, timeout, *, pass_fds=(), env=None):
    """Bound stdin/stdout in memory; kill on overflow/timeout, never retry."""
    end = time.monotonic()+timeout
    p = subprocess.Popen(args, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, pass_fds=pass_fds, env=env)
    result = bytearray(); remaining = memoryview(payload)
    try:
        os.set_blocking(p.stdin.fileno(), False);os.set_blocking(p.stdout.fileno(), False)
        with selectors.DefaultSelector() as events:
            events.register(p.stdin, selectors.EVENT_WRITE);events.register(p.stdout, selectors.EVENT_READ)
            while events.get_map():
                wait = end-time.monotonic();require(wait > 0)
                ready = events.select(wait);require(ready)
                for key, _ in ready:
                    if key.fileobj is p.stdin:
                        if remaining:
                            try: n=os.write(p.stdin.fileno(),remaining[:65536])
                            except BrokenPipeError: n=0;remaining=memoryview(b'')
                            remaining=remaining[n:]
                        if not remaining: events.unregister(p.stdin);p.stdin.close()
                    else:
                        chunk=os.read(p.stdout.fileno(),65537-len(result))
                        if chunk: result.extend(chunk);require(len(result)<=65536)
                        else: events.unregister(p.stdout)
        return p.wait(timeout=max(0.001,end-time.monotonic())),bytes(result)
    finally:
        if p.poll() is None: p.kill()
        p.wait()
        p.stdout.close()
        if not p.stdin.closed: p.stdin.close()


def checked_result(code, raw, source):
    d=projection(raw, source)
    require(code == 0 or d['status'] == 'UNVERIFIED')
    return raw


def execute(raw):
    require(len(raw) <= LIMIT)
    data=json.loads(raw.decode('utf-8'),object_pairs_hook=unique)
    shape(data,'binary sha256 context readerSource')
    source=data['readerSource'];require(source_ok(source))
    binary=base64.b64decode(data['binary'],validate=True)
    require(0 < len(binary) <= 16*1024*1024 and hashlib.sha256(binary).hexdigest() == data['sha256'])
    context=data['context'].encode(); require(len(context)<=32768 and hashlib.sha256(context).hexdigest()==CONTEXT_SHA)
    fd=os.memfd_create('grace-s120',os.MFD_CLOEXEC|os.MFD_ALLOW_SEALING)
    try:
        view=memoryview(binary)
        while view:
            n=os.write(fd,view);require(n>0);view=view[n:]
        fcntl.fcntl(fd,fcntl.F_ADD_SEALS,fcntl.F_SEAL_WRITE|fcntl.F_SEAL_GROW|fcntl.F_SEAL_SHRINK|fcntl.F_SEAL_SEAL)
        code,out=capture(['/proc/self/fd/'+str(fd),'-mode=read','-authorized-source='+source],context,45,pass_fds=(fd,),env={})
        return checked_result(code,out,source)
    finally:
        os.close(fd)


def main():
    source=''
    try:
        raw=sys.stdin.buffer.read(LIMIT+1)
        # Source is reflected only after strict syntactic validation; no error strings.
        data=json.loads(raw.decode('utf-8'),object_pairs_hook=unique)
        if type(data) is dict and source_ok(data.get('readerSource')):source=data['readerSource']
        out=execute(raw);d=projection(out,source)
        sys.stdout.buffer.write(out)
        return 0 if d['status']=='VERIFIED' else 1
    except Exception:
        sys.stdout.buffer.write(failure(source));return 1


if __name__=='__main__':sys.exit(main())
