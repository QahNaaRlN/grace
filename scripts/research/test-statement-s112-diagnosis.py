#!/usr/bin/env python3
"""Offline S1.12 transport tests: real sealed memfd, no SSH/provider/server calls."""
import base64
import contextlib
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent

def module(name, file):
    spec = importlib.util.spec_from_file_location(name, HERE / file)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m

bootstrap = module('bootstrap', 's112-bootstrap.py')
runner = module('runner', 'run-s112-diagnosis.py')

with tempfile.TemporaryDirectory() as tmp:
    root = Path(tmp)
    # This executable asserts its own kernel memfd identity and all four seals.
    # It does not imitate a model response or a saved C6 report.
    code = root / 'memfd.go'
    code.write_text('''package main
import("os";"strings";"syscall")
func main(){
 p,e:=os.Readlink("/proc/self/exe");if e!=nil||!strings.Contains(p,"memfd:grace-s112"){os.Exit(2)}
 // The inherited fd is the /proc/self/exe link target supplied by bootstrap.
 ok:=false
 for fd:=uintptr(3);fd<32;fd++{v,_,err:=syscall.Syscall(syscall.SYS_FCNTL,fd,1034,0);if err==0&&v&15==15{ok=true}}
 if !ok{os.Exit(3)}
 os.Stderr.WriteString("PRIVATE-STDERR")
 os.Stdout.WriteString(`{"verified":true,"artifactsUnchanged":true,"memfd":true}`)
}''')
    binary = root / 'memfd'
    subprocess.run(['go', 'build', '-o', str(binary), str(code)], check=True, stdout=subprocess.DEVNULL)
    raw = binary.read_bytes()
    good = {'binary': base64.b64encode(raw).decode(), 'sha256': hashlib.sha256(raw).hexdigest(), 'context': '{}'}
    before = {p.name: (p.read_bytes(), p.stat().st_mtime_ns) for p in root.iterdir()}
    out = bootstrap.execute(json.dumps(good).encode())
    assert json.loads(out)['memfd'] and b'PRIVATE' not in out
    after = {p.name: (p.read_bytes(), p.stat().st_mtime_ns) for p in root.iterdir()}
    assert before == after
    for kind in ('hash', 'base64', 'context', 'extra', 'duplicate', 'oversize'):
        d = dict(good)
        if kind == 'hash': d['sha256'] = 'a' * 64
        if kind == 'base64': d['binary'] = 'PRIVATE-INVALID'
        if kind == 'context': d['context'] = 'x' * 524289
        if kind == 'extra': d['private'] = 'PRIVATE'
        encoded = json.dumps(d).encode()
        if kind == 'duplicate': encoded = b'{"binary":"PRIVATE","binary":"PRIVATE"}'
        if kind == 'oversize': encoded = b' ' * (bootstrap.LIMIT + 1)
        try:
            bootstrap.execute(encoded)
            raise AssertionError('unsafe envelope accepted: ' + kind)
        except (ValueError, TypeError, KeyError): pass
    old_memfd = bootstrap.os.memfd_create
    def unsupported(*args): raise OSError('PRIVATE')
    bootstrap.os.memfd_create = unsupported
    try:
        try:
            bootstrap.execute(json.dumps(good).encode())
            raise AssertionError('memfd fallback allowed')
        except OSError: pass
    finally:
        bootstrap.os.memfd_create = old_memfd

    context = root / 'context.json'
    context.write_text('{}')
    old_run = runner.subprocess.run
    old_args = sys.argv
    try:
        for failure in (False, True):
            def remote(args, **kwargs):
                assert args[:3] == ['ssh', '-o', 'BatchMode=yes']
                assert 'StrictHostKeyChecking=yes' in args and args[-2] == 'grace-deploy@195.133.195.190'
                assert args[-1].startswith('python3 -c ')
                assert kwargs['stderr'] == subprocess.DEVNULL
                assert json.loads(kwargs['input'])['sha256'] == good['sha256']
                return subprocess.CompletedProcess(args, 1 if failure else 0, b'PRIVATE' if failure else out)
            runner.subprocess.run = remote
            sys.argv = ['run', str(binary), str(context)]
            # main writes successful bytes only; replace stdout for exact observation.
            captured = io.BytesIO()
            class Output:
                buffer = captured
                def write(self, value): captured.write(value.encode())
                def flush(self): pass
            with contextlib.redirect_stdout(Output()):
                assert runner.main() == int(failure)
            assert b'PRIVATE' not in captured.getvalue()
    finally:
        runner.subprocess.run = old_run
        sys.argv = old_args

source = (HERE / 's112-diagnose/main.go').read_text()
assert 'O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK' in source
assert 'ValidateComparisonReport' in source and 'MergeProposals' in source
assert all(x not in source for x in ('os.WriteFile', 'os.Create(', 'exec.Command', 'http.Get', '.Reserve(', '.Settle('))
print('S1.12 real sealed memfd, fixed host, no disk fallback, closed failures and privacy PASS')
