#!/usr/bin/env python3
"""Local fixed-host SSH envelope. The private report never leaves the VPS."""
import base64
import hashlib
import json
from pathlib import Path
import shlex
import subprocess
import sys

ERROR = '{"error":"s112_diagnostic_unavailable","verified":false}'

def main():
    try:
        if len(sys.argv) != 3:
            raise ValueError('arguments')
        binary = Path(sys.argv[1]).read_bytes()
        context = Path(sys.argv[2]).read_text()
        if not 0 < len(binary) <= 16 * 1024 * 1024 or len(context.encode()) > 524288:
            raise ValueError('bounds')
        envelope = json.dumps({'binary': base64.b64encode(binary).decode(),
                               'sha256': hashlib.sha256(binary).hexdigest(),
                               'context': context}).encode()
        bootstrap = Path(__file__).with_name('s112-bootstrap.py').read_text()
        result = subprocess.run(['ssh', '-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=yes',
                                 '-o', 'ConnectTimeout=10', 'grace-deploy@195.133.195.190',
                                 'python3 -c ' + shlex.quote(bootstrap)],
                                input=envelope, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=75)
        if result.returncode != 0 or len(result.stdout) > 524288:
            raise ValueError('unavailable')
        value = json.loads(result.stdout)
        if value.get('verified') is not True:
            raise ValueError('unverified')
        sys.stdout.buffer.write(result.stdout)
        return 0
    except Exception:
        print(ERROR)
        return 1

if __name__ == '__main__':
    sys.exit(main())
