"""Remote stdin/memfd bootstrap. No disk files, keys, SQL or provider clients."""
import base64
import fcntl
import hashlib
import json
import os
import subprocess
import sys

LIMIT = 32 * 1024 * 1024
ERROR = '{"error":"s112_diagnostic_unavailable","verified":false}'

def require(ok):
    if not ok:
        raise ValueError('unverified')

def unique(pairs):
    out = {}
    for key, value in pairs:
        require(key not in out)
        out[key] = value
    return out

def execute(raw):
    require(len(raw) <= LIMIT)
    data = json.loads(raw, object_pairs_hook=unique)
    require(set(data) == {'binary', 'sha256', 'context'})
    binary = base64.b64decode(data['binary'], validate=True)
    require(0 < len(binary) <= 16 * 1024 * 1024)
    require(hashlib.sha256(binary).hexdigest() == data['sha256'])
    context = data['context'].encode()
    require(len(context) <= 524288)
    fd = os.memfd_create('grace-s112', os.MFD_CLOEXEC | os.MFD_ALLOW_SEALING)
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
        require(result.returncode == 0 and len(result.stdout) <= 524288)
        # stdout is the allowlisted Go projection; never forward a failed process.
        value = json.loads(result.stdout, object_pairs_hook=unique)
        require(value.get('verified') is True and value.get('artifactsUnchanged') is True)
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
