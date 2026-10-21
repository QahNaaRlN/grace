#!/usr/bin/env python3
"""Local fixed-host SSH envelope. The private report never leaves the VPS."""
import importlib.util
import base64
import hashlib
import json
from pathlib import Path
import shlex
import subprocess
import sys

ERROR = '{"error":"diagnostic_unavailable"}'

def schema():
    path = Path(__file__).with_name('s115-bootstrap.py')
    spec = importlib.util.spec_from_file_location('s115schema', path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

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
        if len(envelope)>32*1024*1024:
            raise ValueError('bounds')
        bootstrap = Path(__file__).with_name('s115-bootstrap.py').read_text()
        result = subprocess.run(['ssh', '-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=yes',
                                 '-o', 'ConnectTimeout=10', 'grace-deploy@195.133.195.190',
                                 'python3 -B -c ' + shlex.quote(bootstrap)],
                                input=envelope, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=75)
        if result.returncode != 0 or len(result.stdout) > 65536:
            raise ValueError('unavailable')
        schema().projection(result.stdout)
        sys.stdout.buffer.write(result.stdout)
        return 0
    except Exception:
        print(ERROR)
        return 1

if __name__ == '__main__':
    sys.exit(main())
