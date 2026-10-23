#!/usr/bin/env python3
"""S1.20 actual pipeline fixtures, Go/Python parity, real memfd, SSH stub only."""
import base64
import contextlib
import copy
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time
sys.dont_write_bytecode=True
HERE=Path(__file__).resolve().parent
REPO=HERE.parents[1]
SOURCE='1'*40


def module(name,filename):
    spec=importlib.util.spec_from_file_location(name,HERE/filename)
    m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m


bootstrap=module('s120testbootstrap','s120-bootstrap.py')
runner=module('s120testrunner','run-s120-diagnosis.py')


class Output:
    def __init__(self):self.buffer=io.BytesIO()
    def write(self,value):return self.buffer.write(value.encode())


with tempfile.TemporaryDirectory(prefix='grace-s120-transport-') as tmp:
    root=Path(tmp)
    subprocess.run(['go','test','./internal/statementqualification','-run','^TestClockDiagnosisActualPipeline$','-count=1'],cwd=REPO,env=dict(os.environ,TEST_S120_OUTPUT=tmp),check=True,stdout=subprocess.DEVNULL)
    context=(root/'context.json').read_bytes();good=(root/'projection.json').read_bytes()
    verified=json.loads(good);failure=bootstrap.failure(SOURCE,'summary_mismatch')
    binary=root/'reader'
    subprocess.run(['go','build','-trimpath','-ldflags','-X main.readerSource='+SOURCE,'-o',str(binary),'./scripts/research/s120-diagnose'],cwd=REPO,check=True,stdout=subprocess.DEVNULL)
    assert bootstrap.projection(good,SOURCE)['status']=='VERIFIED'
    cases=[(good,True),(failure,True)]
    for path,key,value in [('', 'providerCalls', True),('', 'sqlCalls',1),('accounting','actualMicros',61),('accounting','callCostsMicros',[-1,10,10,10,10,10]),('', 'rejectReasons',[]),('', 'reportDigest','PRIVATE'),('', 'readerSource','f'*40),('', 'private','PRIVATE')]:
        d=copy.deepcopy(verified);(d[path] if path else d)[key]=value;cases.append((json.dumps(d).encode(),False))
    for key,value in [('clockCount',True),('correct',999),('precision',[True]*5),('checks',verified['attempts'][0]['checks'][:19])]:
        d=copy.deepcopy(verified);d['attempts'][0][key]=value;cases.append((json.dumps(d).encode(),False))
    for key,value in [('nonClockRegressions',[]),('baseOrdinal',True),('precisionFailures',[{'rowOrdinal':1,'field':'PRIVATE'}])]:
        d=copy.deepcopy(verified);d['pairs'][0][key]=value;cases.append((json.dumps(d).encode(),False))
    for key,value in [('code','PRIVATE'),('ordinal',True),('rowOrdinal',True),('passed',0)]:
        d=copy.deepcopy(verified);d['attempts'][0]['checks'][0][key]=value;cases.append((json.dumps(d).encode(),False))
    for key,value in [('reason','PRIVATE'),('private','PRIVATE'),('activationAuthorized',True)]:
        d=json.loads(failure);d[key]=value;cases.append((json.dumps(d).encode(),False))
    cases.extend([(good+b' null',False),(good.replace(b'"providerCalls":0',b'"providerCalls":0,"ProviderCalls":0'),False),(good.replace(b'"providerCalls":0',b'"providerCalls":null'),False),(b' '*65537,False),(b'"\xff"',False),(b'\xef\xbb\xbf'+good,False),(good.decode().encode('utf-16'),False)])
    for raw,expected in cases:
        accepted=True
        try:bootstrap.projection(raw,SOURCE)
        except Exception:accepted=False
        p=subprocess.run([str(binary),'-mode=validate','-authorized-source='+SOURCE],input=raw,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        assert accepted==expected and (p.returncode==0)==expected,(expected,raw[:100],p.stdout)
        assert b'PRIVATE' not in p.stdout+p.stderr

    # A real Linux memfd binary checks seals, empty env, immutable source args,
    # consumes only the public fixture, and deliberately emits private stderr.
    code=root/'memfd.go'
    code.write_text('''package main
import("os";"io";"strings";"syscall")
var scenario="verified"
func main(){
if len(os.Args)==2&&os.Args[1]=="-mode=check"{os.Stdout.WriteString('''+json.dumps(json.dumps(bootstrap.manifest(SOURCE)))+''');return}
p,e:=os.Readlink("/proc/self/exe");if e!=nil||!strings.Contains(p,"memfd:grace-s120")||len(os.Environ())!=0||len(os.Args)!=3||os.Args[1]!="-mode=read"||os.Args[2]!="-authorized-source='''+SOURCE+'''"{os.Exit(2)}
ok:=false;for fd:=uintptr(3);fd<32;fd++{v,_,err:=syscall.Syscall(syscall.SYS_FCNTL,fd,1034,0);if err==0&&v&15==15{ok=true}};if !ok{os.Exit(4)}
_,e=io.ReadAll(os.Stdin);if e!=nil{os.Exit(5)}
os.Stderr.WriteString("PRIVATE-STDERR")
if scenario=="failed" {os.Stdout.WriteString('''+json.dumps(failure.decode())+''');os.Exit(7)}
os.Stdout.WriteString('''+json.dumps(good.decode())+''')}
''')
    memfd=root/'memfd'
    def build(mode):
        subprocess.run(['go','build','-ldflags','-X main.scenario='+mode,'-o',str(memfd),str(code)],check=True,stdout=subprocess.DEVNULL)
        content=memfd.read_bytes()
        return json.dumps(dict(binary=base64.b64encode(content).decode(),sha256=hashlib.sha256(content).hexdigest(),context=context.decode(),readerSource=SOURCE)).encode()
    envelope=build('verified');assert bootstrap.execute(envelope)==good
    failed_envelope=build('failed');assert bootstrap.execute(failed_envelope)==failure
    for change in [('context','PRIVATE'),('sha256','f'*64),('readerSource','PRIVATE'),('binary','!')]:
        d=json.loads(envelope);d[change[0]]=change[1]
        try:bootstrap.execute(json.dumps(d).encode())
        except Exception:pass
        else:raise AssertionError('unsafe memfd envelope')
    try:bootstrap.capture([sys.executable,'-c','import sys;sys.stdout.write("PRIVATE"*10000)'],b'',2)
    except ValueError:pass
    else:raise AssertionError('stdout overflow accepted')
    before=time.monotonic()
    try:bootstrap.capture([sys.executable,'-c','import time;time.sleep(10)'],b'',0.1)
    except ValueError:pass
    else:raise AssertionError('timeout accepted')
    assert time.monotonic()-before<3

    jobs=[dict(name=n,status='completed',conclusion='success') for n in runner.ci.JOBS]
    ci=root/'ci.json';ci.write_text(json.dumps(dict(headSha=SOURCE,status='completed',conclusion='success',jobs=jobs)))
    real_capture=runner.schema.capture;ssh_calls=[];result_code=0;result_raw=good
    def fake_capture(args,payload,timeout,**kw):
        if args[0]!='ssh':return real_capture(args,payload,timeout,**kw)
        ssh_calls.append(args)
        assert args[:3]==['ssh','-o','BatchMode=yes'] and args.count('grace-deploy@195.133.195.190')==1
        assert 'StrictHostKeyChecking=yes' in args and timeout==75 and 'python3 -B -c ' in args[-1]
        data=json.loads(payload);assert hashlib.sha256(data['context'].encode()).hexdigest()==bootstrap.CONTEXT_SHA
        return result_code,result_raw
    runner.schema.capture=fake_capture
    oldargv=sys.argv
    args=['run',str(binary),str(root/'context.json'),SOURCE,SOURCE,'true',str(ci)]
    try:
        sys.argv=args
        for result_code,result_raw,expected in [(0,good,0),(7,failure,1),(0,b'PRIVATE',1),(7,good,1),(0,good.replace(SOURCE.encode(),b'f'*40),1),(0,b'x'*65537,1)]:
            count=len(ssh_calls);capture=Output()
            with contextlib.redirect_stdout(capture):status=runner.main()
            out=capture.buffer.getvalue();assert status==expected and len(ssh_calls)==count+1 and b'PRIVATE' not in out
            bootstrap.projection(out,SOURCE)
            if result_raw==failure:assert json.loads(out)['reason']=='summary_mismatch'
        for index,value in [(4,'f'*40),(5,'false'),(3,'PRIVATE')]:
            sys.argv=args.copy();sys.argv[index]=value;count=len(ssh_calls);capture=Output()
            with contextlib.redirect_stdout(capture):assert runner.main()==1
            assert len(ssh_calls)==count and b'PRIVATE' not in capture.buffer.getvalue()
        sys.argv=args
        for mutation in ['ci','context','binary','compiled_source']:
            target={'ci':ci,'context':root/'context.json','binary':binary,'compiled_source':binary}[mutation];saved=target.read_bytes()
            if mutation=='binary':target.write_bytes(b'x'*(16*1024*1024+1))
            elif mutation=='compiled_source':target.write_bytes(memfd.read_bytes()) # test binary has same manifest; change source to force preflight below
            else:target.write_bytes(b'PRIVATE')
            if mutation=='compiled_source':
                sys.argv=args.copy();sys.argv[3]=sys.argv[4]='f'*40
                ci.write_text(json.dumps(dict(headSha='f'*40,status='completed',conclusion='success',jobs=jobs)))
            count=len(ssh_calls);capture=Output()
            with contextlib.redirect_stdout(capture):assert runner.main()==1
            assert len(ssh_calls)==count and b'PRIVATE' not in capture.buffer.getvalue()
            target.write_bytes(saved);sys.argv=args
    finally:sys.argv=oldargv;runner.schema.capture=real_capture

# Check actual workflow guard blocks through shell stubs, not a duplicate model.
workflow=(REPO/'.github/workflows/statement-clock-diagnosis.yml').read_text()
assert workflow.count('\n  clock-diagnosis:')==1 and 'if: ${{ inputs.execute_approved_diagnosis }}' in workflow
assert 'default: false' in workflow and 'group: production' in workflow and 'environment: production' in workflow
assert 'deploy-production.yml' not in workflow and all(x not in workflow for x in ('scp ', 'docker ', 'openrouter', 'GRACE_AI_'))
assert workflow.count('run-s120-diagnosis.py" "$bundle/reader"')==1
assert workflow.count('test "$AUTHORIZED_SHA" = "$GITHUB_SHA"')==2
assert workflow.count('git merge-base --is-ancestor "$GITHUB_SHA" origin/main')==2
# Execute the first actual workflow guard with hermetic gh/git, including
# mismatched source, non-ancestor and skipped required job. No network/SSH.
guard=workflow.split('      - name: Verify exact authorized source and CI 5/5',1)[1].split('        run: |\n',1)[1].split('      - name:',1)[0]
guard='\n'.join(line[10:] for line in guard.splitlines())
with tempfile.TemporaryDirectory(prefix='grace-s120-workflow-') as td:
    d=Path(td);bin_dir=d/'bin';bin_dir.mkdir()
    git=bin_dir/'git';git.write_text('#!/bin/sh\nexit "$TEST_ANCESTOR_STATUS"\n');git.chmod(0o755)
    gh=bin_dir/'gh';gh.write_text('#!/bin/sh\ncase "$2" in list) echo 123 ;; view) cat "$TEST_CI_JSON" ;; *) exit 4 ;; esac\n');gh.chmod(0o755)
    ci_file=d/'source-ci.json'
    jobs=[dict(name=n,status='completed',conclusion='success') for n in runner.ci.JOBS]
    for scenario in ('valid','source','ancestor','ci_sha','skipped','rerun'):
        data=dict(headSha=SOURCE,status='completed',conclusion='success',jobs=copy.deepcopy(jobs))
        if scenario=='ci_sha':data['headSha']='f'*40
        if scenario=='skipped':data['jobs'][0]['conclusion']='skipped'
        ci_file.write_text(json.dumps(data))
        env=dict(os.environ,PATH=str(bin_dir)+os.pathsep+os.environ['PATH'],GITHUB_SHA=SOURCE,GITHUB_RUN_ATTEMPT=('2' if scenario=='rerun' else '1'),AUTHORIZED_SHA=('f'*40 if scenario=='source' else SOURCE),RUNNER_TEMP=td,TEST_CI_JSON=str(ci_file),TEST_ANCESTOR_STATUS=('1' if scenario=='ancestor' else '0'))
        p=subprocess.run(['bash','-c',guard],cwd=REPO,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        assert (p.returncode==0)==(scenario=='valid'),(scenario,p.stdout,p.stderr)
assert 'sha256sum -c SHA256SUMS' in workflow and 'verify-screening-source-ci.py' in workflow
artifact=workflow.split('uses: actions/upload-artifact@v7',1)[1].split('      - name:',1)[0]
assert all(x not in artifact for x in ('s120-closed','report.json','projection.json','reader\n','context.json'))
print(f'S1.20 PASS: actual six-response fixture; {len(cases)} Go/Python cases; real memfd seals/empty env/nonzero cause; bounded output/timeout; one SSH stub, preflight refusals; SSH0 live')
