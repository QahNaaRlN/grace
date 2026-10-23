#!/usr/bin/env python3
"""Q4 closed Go/Python parity, actual sealed memfd, wrapper privacy. Never SSH."""
import base64
import contextlib
import copy
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
sys.dont_write_bytecode=True
HERE=Path(__file__).resolve().parent
SOURCE='1'*40

def module(name,file):
    spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
bootstrap=module('s118bootstrap','s118-bootstrap.py');runner=module('s118runner','run-s118-diagnosis.py')
base={'version':'statement.saved-quality-diagnosis.s118.v1','authorization':'s117-comparison-20261023','artifactSource':'24f73223b9b3370ca78c41d34e89edca9a276488','verifierSource':SOURCE,'contextVerified':True,'artifactsUnchanged':True,'providerCalls':0,'persistentWritesRequested':0,'tuples':[{'ordinal':i+1,'status':'UNVERIFIED','reason':'snapshot_unavailable'} for i in range(6)]}
position={'sourceOrdinal':1,'page':1,'block':1,'line':1,'word':1,'lexicalClass':'other'}
verified=copy.deepcopy(base)
verified['tuples'][0]={'ordinal':1,'status':'VERIFIED','coverage':{'status':'VERIFIED','totalUnmapped':1,'positions':[position]},'reasons':[{'ordinal':1,'scope':'coverage','status':'VERIFIED','code':'unmapped_spans','relation':'mapped','publicCode':'unknown','count':1}],'checks':[{'ordinal':17,'status':'VERIFIED','code':'coverage','state':'incomplete','links':[1]}]}
good=[json.dumps(x).encode() for x in (base,verified)]
invalid=[]
for key,value in [('private','PRIVATE'),('verifierSource','f'*40),('artifactSource','f'*40),('providerCalls',True),('artifactsUnchanged',False)]:
    d=copy.deepcopy(verified);d[key]=value;invalid.append(json.dumps(d).encode())
for path,key,value in [('coverage','positions',[dict(position,rawID='PRIVATE')]),('coverage','totalUnmapped',True)]:
    d=copy.deepcopy(verified);d['tuples'][0][path][key]=value;invalid.append(json.dumps(d).encode())
for key,value in [('code','PRIVATE'),('publicCode','PRIVATE'),('rowOrdinal',True),('reason','PRIVATE'),('field','PRIVATE')]:
    d=copy.deepcopy(verified);d['tuples'][0]['reasons'][0][key]=value;invalid.append(json.dumps(d).encode())
for key,value in [('positions',None),('positions',[]),('rowOrdinal',0),('clock',{'date':'PRIVATE'}),('links',[True]),('state','PRIVATE'),('code','PRIVATE')]:
    d=copy.deepcopy(verified);d['tuples'][0]['checks'][0][key]=value;invalid.append(json.dumps(d).encode())
invalid += [good[0]+b' null',good[0].replace(b'"providerCalls": 0',b'"providerCalls": 0,"ProviderCalls":0'), b' '*65537]
with tempfile.TemporaryDirectory() as tmp:
    root=Path(tmp);verifier=root/'verifier'
    subprocess.run(['go','build','-ldflags','-X main.verifierSource='+SOURCE,'-o',str(verifier),'./scripts/research/s118-diagnose'],cwd=HERE.parents[1],check=True,stdout=subprocess.DEVNULL)
    for raw in good+invalid:
        accepted=True
        try:bootstrap.projection(raw,SOURCE)
        except (ValueError,TypeError,KeyError):accepted=False
        expected=raw in good
        assert accepted==expected
        p=subprocess.run([str(verifier),'validate'],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        assert (p.returncode==0)==expected,(raw,p.stdout)
        assert b'PRIVATE' not in p.stdout+p.stderr
    code=root/'memfd.go'
    code.write_text('''package main
import("os";"strings";"syscall")
func main(){p,e:=os.Readlink("/proc/self/exe");if e!=nil||!strings.Contains(p,"memfd:grace-s118")||len(os.Environ())!=0{os.Exit(2)}
ok:=false;for fd:=uintptr(3);fd<32;fd++{v,_,err:=syscall.Syscall(syscall.SYS_FCNTL,fd,1034,0);if err==0&&v&15==15{ok=true}};if !ok{os.Exit(4)}
os.Stderr.WriteString("PRIVATE-STDERR");os.Stdout.WriteString(`'''+good[1].decode()+'''`)}''')
    binary=root/'memfd';subprocess.run(['go','build','-o',str(binary),str(code)],check=True,stdout=subprocess.DEVNULL)
    content=binary.read_bytes();envelope=json.dumps({'binary':base64.b64encode(content).decode(),'sha256':hashlib.sha256(content).hexdigest(),'context':'[]','verifierSource':SOURCE}).encode()
    assert bootstrap.execute(envelope)==good[1]
    real_run=runner.subprocess.run
    class Result:
        returncode=0
        stdout=good[1]
    def fake_run(args,**kwargs):
        assert args[:2]==['ssh','-o'] and 'grace-deploy@195.133.195.190' in args
        assert kwargs['stderr']==subprocess.DEVNULL and kwargs['timeout']==75
        return Result()
    runner.subprocess.run=fake_run
    binary_path=root/'binary';binary_path.write_bytes(content);ctx=root/'context';ctx.write_text('[]')
    oldargv=sys.argv
    class Output:
        def __init__(self):self.buffer=io.BytesIO()
        def write(self,value):return self.buffer.write(value.encode())
    try:
        sys.argv=['run',str(binary_path),str(ctx),SOURCE]
        for raw,success in [(good[1],True),(b'PRIVATE',False),(good[1].replace(SOURCE.encode(),b'f'*40),False)]:
            Result.stdout=raw;capture=Output()
            with contextlib.redirect_stdout(capture):result=runner.main()
            assert (result==0)==success and b'PRIVATE' not in capture.buffer.getvalue()
        def timeout(*args,**kwargs):raise subprocess.TimeoutExpired('PRIVATE',75)
        runner.subprocess.run=timeout;capture=Output()
        with contextlib.redirect_stdout(capture):assert runner.main()==1
        assert b'PRIVATE' not in capture.buffer.getvalue()
    finally:sys.argv=oldargv;runner.subprocess.run=real_run
print('S1.18 strict Go/Python parity, actual memfd/empty env/seals, fixed-host wrapper/privacy PASS; SSH0')
