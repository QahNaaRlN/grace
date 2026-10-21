#!/usr/bin/env python3
"""Offline sealed memfd, closed schema and fixed-host wrapper tests. Never SSH."""
import base64
import contextlib
import copy
import hashlib
import importlib.util
import io
import json
import os
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

bootstrap = module('s115bootstrap', 's115-bootstrap.py')
runner = module('s115runner', 'run-s115-diagnosis.py')
base = {'version':'statement.saved-proposal-diagnosis.v1', 'authorization':'s114-comparison-20261021',
        'source':'301afd6bedc4a19559e731471336bbc4e2f74075', 'contextVerified':True,
        'artifactsUnchanged':True, 'providerCalls':0, 'persistentWritesRequested':0, 'tuples':[]}
for i,kind in enumerate(('description_gap','unmapped','alias_relation','not_applicable','not_applicable','not_applicable')):
    base['tuples'].append({'ordinal':i+1,'kind':kind,'status':'UNVERIFIED' if i<3 else 'not_applicable',
                           'reason':'snapshot_unavailable' if i<3 else 'not_sent'})
base['tuples'][2] = {'ordinal':3,'kind':'alias_relation','status':'VERIFIED','relation':'outside_alias_syntax'}
good_projection = json.dumps(base).encode()
assert bootstrap.projection(good_projection)['tuples'][2]['status'] == 'VERIFIED'
verified = copy.deepcopy(base)
verified['tuples'][0] = {'ordinal':1,'kind':'description_gap','status':'VERIFIED','gap':{'rowOrdinal':2,'left':{'sourceOrdinal':1,'page':1,'block':1,'line':1,'word':1},'right':{'sourceOrdinal':3,'page':1,'block':1,'line':1,'word':3},'intermediateUnselected':1}}
verified['tuples'][1] = {'ordinal':2,'kind':'unmapped','status':'VERIFIED','totalUnmapped':1,'witnesses':[{'sourceOrdinal':1,'page':1,'block':1,'line':1,'word':1,'lexicalClass':'numeric'}]}
verified_projection = json.dumps(verified).encode()
bootstrap.projection(verified_projection)
invalid = []
for index, field, value in [(0,'rowOrdinal',True),(0,'intermediateUnselected',0)]:
    d=copy.deepcopy(verified);d['tuples'][index]['gap'][field]=value;invalid.append(json.dumps(d).encode())
for field,value in [('sourceOrdinal',True),('lexicalClass','PRIVATE'),('rawID','PRIVATE')]:
    d=copy.deepcopy(verified);d['tuples'][1]['witnesses'][0][field]=value;invalid.append(json.dumps(d).encode())

for field,value in [('private','PRIVATE'),('source','PRIVATE'),('providerCalls',True),('artifactsUnchanged',False),('providerCalls',1)]:
    d=copy.deepcopy(base);d[field]=value;invalid.append(json.dumps(d).encode())
for index,field,value in [(0,'reason','PRIVATE'),(2,'relation','PRIVATE'),(2,'rawAlias','PRIVATE'),(3,'status','VERIFIED'),(0,'gap',{'private':'PRIVATE'})]:
    d=copy.deepcopy(base);d['tuples'][index][field]=value;invalid.append(json.dumps(d).encode())
invalid += [good_projection.replace(b'"providerCalls": 0',b'"providerCalls": null'), good_projection.replace(b'"source":',b'"Source":'), b'{"source":"PRIVATE","source":"PRIVATE"}', good_projection + b' null', b' '*65537]
for raw in invalid:
    try: bootstrap.projection(raw)
    except (ValueError,TypeError,KeyError): pass
    else: raise AssertionError('open projection')

with tempfile.TemporaryDirectory() as tmp:
    root = Path(tmp)
    verifier = root/'verifier'
    subprocess.run(['go','build','-o',str(verifier),'./scripts/research/s115-diagnose'], cwd=HERE.parents[1], check=True, stdout=subprocess.DEVNULL)
    for raw in [good_projection,verified_projection]+invalid:
        p = subprocess.run([str(verifier),'validate'], input=raw, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        assert p.returncode == (0 if raw in (good_projection,verified_projection) else 1)
        assert b'PRIVATE' not in p.stdout+p.stderr
    # Actual kernel memfd execution: all seals and empty environment, fixed closed stdout.
    code = root/'memfd.go'
    code.write_text('''package main
import("os";"strings";"syscall")
func main(){
p,e:=os.Readlink("/proc/self/exe");if e!=nil||!strings.Contains(p,"memfd:grace-s115"){os.Exit(2)}
if len(os.Environ())!=0{os.Exit(3)}
ok:=false;for fd:=uintptr(3);fd<32;fd++{v,_,err:=syscall.Syscall(syscall.SYS_FCNTL,fd,1034,0);if err==0&&v&15==15{ok=true}};if !ok{os.Exit(4)}
os.Stderr.WriteString("PRIVATE-STDERR")
os.Stdout.WriteString(`'''+good_projection.decode()+'''`)
}''')
    binary = root/'memfd'
    subprocess.run(['go','build','-o',str(binary),str(code)], check=True, stdout=subprocess.DEVNULL)
    content=binary.read_bytes()
    envelope={'binary':base64.b64encode(content).decode(), 'sha256':hashlib.sha256(content).hexdigest(), 'context':'[]'}
    before={p.name:(p.read_bytes(),p.stat().st_mtime_ns) for p in root.iterdir()}
    os.environ['GRACE_AI_PRIVATE_POISON']='PRIVATE'
    out=bootstrap.execute(json.dumps(envelope).encode())
    assert out==good_projection
    assert before=={p.name:(p.read_bytes(),p.stat().st_mtime_ns) for p in root.iterdir()}
    for kind in ('hash','base64','context','extra','duplicate','oversize'):
        d=dict(envelope)
        if kind=='hash': d['sha256']='a'*64
        if kind=='base64': d['binary']='PRIVATE'
        if kind=='context': d['context']='x'*524289
        if kind=='extra': d['private']='PRIVATE'
        raw=json.dumps(d).encode()
        if kind=='duplicate':raw=b'{"binary":"PRIVATE","binary":"PRIVATE"}'
        if kind=='oversize':raw=b' '*(bootstrap.LIMIT+1)
        try:bootstrap.execute(raw)
        except (ValueError,TypeError,KeyError):pass
        else:raise AssertionError('unsafe envelope '+kind)
    old_memfd=bootstrap.os.memfd_create
    try:
        def unsupported(*args): raise OSError('PRIVATE')
        bootstrap.os.memfd_create=unsupported
        try:bootstrap.execute(json.dumps(envelope).encode())
        except OSError:pass
        else:raise AssertionError('memfd fallback')
    finally:bootstrap.os.memfd_create=old_memfd
    old_run=bootstrap.subprocess.run
    try:
        for mode in ('failed','timeout','schema','oversize'):
            def fake_process(args,**kwargs):
                assert kwargs['env']=={} and kwargs['timeout']==45 and kwargs['stderr']==subprocess.DEVNULL
                if mode=='timeout':raise subprocess.TimeoutExpired(args,45,output=b'PRIVATE')
                return subprocess.CompletedProcess(args,1 if mode=='failed' else 0,b' '*65537 if mode=='oversize' else b'PRIVATE')
            bootstrap.subprocess.run=fake_process
            try:bootstrap.execute(json.dumps(envelope).encode())
            except (ValueError,subprocess.TimeoutExpired):pass
            else:raise AssertionError('process leak '+mode)
    finally:bootstrap.subprocess.run=old_run
    context=root/'context.json';context.write_text('[]')
    old_run=runner.subprocess.run;old_args=sys.argv
    try:
        for mode in ('success','failed','timeout','schema','oversize'):
            def remote(args,**kwargs):
                assert args[:3]==['ssh','-o','BatchMode=yes'] and 'StrictHostKeyChecking=yes' in args
                assert args[-2]=='grace-deploy@195.133.195.190' and args[-1].startswith('python3 -c ')
                assert kwargs['timeout']==75 and kwargs['stderr']==subprocess.DEVNULL
                if mode=='timeout':raise subprocess.TimeoutExpired(args,75,output=b'PRIVATE')
                return subprocess.CompletedProcess(args,1 if mode=='failed' else 0,out if mode=='success' else b' '*65537 if mode=='oversize' else b'PRIVATE')
            runner.subprocess.run=remote;sys.argv=['run',str(binary),str(context)]
            captured=io.BytesIO()
            class Output:
                buffer=captured
                def write(self,value):captured.write(value.encode())
                def flush(self):pass
            with contextlib.redirect_stdout(Output()):assert runner.main()==(0 if mode=='success' else 1)
            assert b'PRIVATE' not in captured.getvalue()
    finally:runner.subprocess.run=old_run;sys.argv=old_args
print('S1.15 sealed memfd/env={}, dual closed schema, privacy/errors/timeouts and fixed-host wrapper PASS; SSH/provider calls=0')
