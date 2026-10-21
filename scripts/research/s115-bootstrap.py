"""Remote stdin/memfd bootstrap. No disk files, keys, SQL or provider clients."""
import base64
import fcntl
import hashlib
import json
import os
import subprocess
import sys

LIMIT = 32 * 1024 * 1024
ERROR = '{"error":"diagnostic_unavailable"}'

def require(ok):
    if not ok:
        raise ValueError('unverified')

def unique(pairs):
    out = {}
    for key, value in pairs:
        require(key not in out)
        out[key] = value
    return out


def projection(raw):
    require(len(raw) <= 65536)
    d = json.loads(raw, object_pairs_hook=unique)
    require(type(d) is dict and set(d) == {'version','authorization','source','contextVerified','artifactsUnchanged','providerCalls','persistentWritesRequested','tuples'})
    require(d['version'] == 'statement.saved-proposal-diagnosis.v1' and d['authorization'] == 's114-comparison-20261021' and d['source'] == '301afd6bedc4a19559e731471336bbc4e2f74075')
    require(d['contextVerified'] is True and d['artifactsUnchanged'] is True)
    require(type(d['providerCalls']) is int and d['providerCalls'] == 0 and type(d['persistentWritesRequested']) is int and d['persistentWritesRequested'] == 0)
    require(type(d['tuples']) is list and len(d['tuples']) == 6)
    def integer(x, low=1, high=1000000):
        require(type(x) is int and low <= x <= high)
    def position(p, lexical=False):
        keys={'sourceOrdinal','page','block','line','word'}
        require(type(p) is dict and set(p) == keys | ({'lexicalClass'} if lexical else set()))
        integer(p['sourceOrdinal'],1,2000); integer(p['page'],1,50)
        for k in ('block','line','word'): integer(p[k])
        if lexical: require(p['lexicalClass'] in ('numeric','other'))
    for i,r in enumerate(d['tuples']):
        require(type(r) is dict and r.get('ordinal') == i+1 and type(r['ordinal']) is int)
        require(r.get('kind') == ('description_gap','unmapped','alias_relation','not_applicable','not_applicable','not_applicable')[i])
        base={'ordinal','kind','status'}
        if i>2:
            require(set(r)==base|{'reason'} and r['status']=='not_applicable' and r['reason']=='not_sent'); continue
        if r.get('status') == 'UNVERIFIED':
            require(set(r) in (base|{'reason'},base|{'reason','totalUnmapped'}) and r['reason'] in ('snapshot_unavailable','context_mismatch','guard_mismatch','replay_mismatch','diagnosis_limit_exceeded','unavailable_runtime'))
            if 'totalUnmapped' in r:
                require(i==1); integer(r['totalUnmapped'],0,2000)
            continue
        require(r.get('status') == 'VERIFIED')
        if i==0:
            require(set(r)==base|{'gap'})
            g=r['gap'];require(type(g) is dict and set(g)=={'rowOrdinal','left','right','intermediateUnselected'} and type(g['rowOrdinal']) is int and g['rowOrdinal']==2)
            position(g['left']);position(g['right'])
            a,b=g['left'],g['right']
            require(a['sourceOrdinal']<b['sourceOrdinal'] and all(a[k]==b[k] for k in ('page','block','line')) and b['word']>a['word']+1)
            integer(g['intermediateUnselected'],1,b['sourceOrdinal']-a['sourceOrdinal']-1)
        elif i==1:
            require(set(r) in (base|{'totalUnmapped'},base|{'totalUnmapped','witnesses'}))
            integer(r['totalUnmapped'],0,128)
            ws=r.get('witnesses',[]);require(type(ws) is list and len(ws)==r['totalUnmapped'])
            previous=0
            for w in ws:
                position(w,True);require(w['sourceOrdinal']>previous);previous=w['sourceOrdinal']
        else:
            require(set(r)==base|{'relation'} and r['relation'] in ('original_span_id','canonical_alias_syntax_absent','outside_alias_syntax'))
    return d

def execute(raw):
    require(len(raw) <= LIMIT)
    data = json.loads(raw, object_pairs_hook=unique)
    require(set(data) == {'binary', 'sha256', 'context'})
    binary = base64.b64decode(data['binary'], validate=True)
    require(0 < len(binary) <= 16 * 1024 * 1024)
    require(hashlib.sha256(binary).hexdigest() == data['sha256'])
    context = data['context'].encode()
    require(len(context) <= 524288)
    fd = os.memfd_create('grace-s115', os.MFD_CLOEXEC | os.MFD_ALLOW_SEALING)
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
        projection(result.stdout)
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
