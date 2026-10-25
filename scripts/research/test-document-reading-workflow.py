#!/usr/bin/env python3
# Actual Q shell blocks/CLI with fake git/gh/SSH/SCP/Docker; no live credentials.
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

REPO=Path(__file__).resolve().parents[2]
WORKFLOW=(REPO/'.github/workflows/document-reading-eval.yml').read_text()
WRAPPER=REPO/'deployments/production/document-reading-eval.sh'
assert 'default: false' in WORKFLOW and 'if: ${{ inputs.execute_approved_reading }}' in WORKFLOW
assert 'workflow_dispatch:' in WORKFLOW and 'pull_request:' not in WORKFLOW
assert 'PRODUCTION_STATEMENT_ROW_SSH_KEY' in WORKFLOW and 'PRODUCTION_STATEMENT_OPERATOR_OWNER_ID' in WORKFLOW
assert 'secrets.PRODUCTION_SSH_KEY' not in WORKFLOW and 'grace-deploy@' not in WORKFLOW

def block(name):
    section=WORKFLOW.split('      - name: '+name+'\n',1)[1]
    body=section.split('        run: |\n',1)[1]
    lines=[]
    for line in body.splitlines():
        if line and not line.startswith('          '):break
        lines.append(line[10:])
    return '\n'.join(lines)+'\n'

with tempfile.TemporaryDirectory(prefix='grace-q-controls-') as temporary:
    root=Path(temporary)
    sha=hashlib.sha1(temporary.encode()).hexdigest()
    bins=root/'bin';bins.mkdir()
    log=root/'commands'
    env=dict(os.environ,TEST_Q_OUTPUT=temporary,TEST_Q_SOURCE=sha)
    for key in ['OPENROUTER_API_KEY','OPENROUTER_API_KEY_FILE','TYPESAFE_API_KEY','GRACE_STATEMENT_PAGE_OPENAI_KEY']:env.pop(key,None)
    subprocess.run(['go','test','./internal/documentqualification','-run','^TestQActualGatewayEightCasesAndReplay$','-count=1'],cwd=REPO,env=env,check=True,stdout=subprocess.DEVNULL)
    prepared=root/'bundle';manifest=hashlib.sha256((prepared/'MANIFEST.json').read_bytes()).hexdigest()
    good=root/'good-summary.json';assert json.loads(good.read_text())['manifestSHA256']==manifest
    binary=root/'reader'
    subprocess.run(['go','build','-trimpath','-ldflags','-X main.sourceSHA='+sha,'-o',str(binary),'./cmd/document-reading-eval'],cwd=REPO,env=env,check=True)
    for name,body in {
        'git':'case "$1" in merge-base|cat-file) test "${ANCESTOR:-true}" = true ;; status|log|rev-parse|-c|--version) exec "$REAL_GIT" "$@" ;; *)exit 9;;esac\n',
        'gh':r'''case "$*" in api*) echo "${RUN_COUNT:-1}" ;; run\ list*) echo 123 ;; run\ view*) cat "$CI_FIXTURE" ;; *) exit 9;;esac
''',
        'scp':'echo scp >> "$COMMANDS"\ntest "${FAKE_SSH:-good}" != scp_fail\n',
        'ssh':'''echo ssh >> "$COMMANDS"
case "$*" in *'mkdir '*) test "$FAKE_SSH" != auth_fail;exit $?;;esac
case "$FAKE_SSH" in good)cat "$GOOD_SUMMARY";;private)echo '{"private":"PRIVATE-MODEL-TEXT"}';;overflow)head -c 70000 /dev/zero | tr '\\0' X;;loss)echo PRIVATE-STDERR >&2;exit 255;;*)exit 9;;esac
''',
        'docker':'''echo "$*" >> "$DOCKER_COMMANDS"
case "$1" in
 ps)echo abc123;;
 inspect)if [ -e "$INSPECT_SEEN" ] && [ "$FAKE_WRAPPER" = changed ];then echo changed;else echo 'container image';fi;touch "$INSPECT_SEEN";;
 cp)exit 0;;
 exec)case "$*" in
  *-mode=run*) if [ "$FAKE_WRAPPER" = run_fail ];then echo PRIVATE-STDERR >&2;exit 1;fi;echo '{"private":"PRIVATE-MODEL-TEXT"}';;
  *-mode=summary*) cat >/dev/null;if [ "$FAKE_WRAPPER" = run_fail ];then exit 2;fi;cat "$GOOD_SUMMARY";;
  *)exit 0;;esac;;
 *)exit 9;;esac
'''
    }.items():
        path=bins/name;path.write_text('#!/bin/sh\nset -eu\n'+body);path.chmod(0o755)
    jobs=['Quality gates','PostgreSQL integration gates','Build api image','Build web image','Build pdf-extractor image']
    ci=root/'ci.json'
    base=dict(env,REAL_GIT=shutil.which('git'),PATH=str(bins)+':'+env['PATH'],RUNNER_TEMP=temporary,GITHUB_SHA=sha,AUTHORIZED_SHA=sha,APPROVAL_ID='b'*40,APPROVED_MANIFEST=manifest,GITHUB_RUN_ATTEMPT='1',GITHUB_REPOSITORY='QahNaaRlN/grace',OPERATOR_OWNER_ID='00000000-0000-0000-0000-000000000001',COMMANDS=str(log),CI_FIXTURE=str(ci),GOOD_SUMMARY=str(good),FAKE_SSH='good')
    guard=block('Проверить source, единственную попытку и CI5/5')
    for scenario in ['good','source','retry','ancestor','duplicate_dispatch','skipped_ci','ci_source']:
        cx=base.copy();data=dict(headSha=sha,status='completed',conclusion='success',jobs=[dict(name=n,status='completed',conclusion='success') for n in jobs])
        if scenario=='source':cx['AUTHORIZED_SHA']='0'*40
        if scenario=='retry':cx['GITHUB_RUN_ATTEMPT']='2'
        if scenario=='ancestor':cx['ANCESTOR']='false'
        if scenario=='duplicate_dispatch':cx['RUN_COUNT']='2'
        if scenario=='skipped_ci':data['jobs'][0]['conclusion']='skipped'
        if scenario=='ci_source':data['headSha']='0'*40
        ci.write_text(json.dumps(data))
        result=subprocess.run(['bash','-c',guard],cwd=REPO,env=cx,capture_output=True)
        assert (result.returncode==0)==(scenario=='good'),(scenario,result.stdout,result.stderr)
    # Exercise the real build/check/export block with the current operator proposal.
    checked=subprocess.run([str(binary),'-mode=check'],cwd=REPO,env=env,check=True,capture_output=True).stdout.rstrip(b'\n')
    build_manifest=hashlib.sha256(checked).hexdigest()
    build_root=root/'build-control';build_root.mkdir()
    build_env=dict(base,RUNNER_TEMP=str(build_root),APPROVED_MANIFEST=build_manifest)
    build=subprocess.run(['bash','-c',block('Собрать bundle и сверить одобренный manifest')],cwd=REPO,env=build_env,capture_output=True)
    assert build.returncode==0,(build.stdout,build.stderr)
    built_manifest=json.loads((build_root/('grace-document-reading-q-'+sha)/'MANIFEST.json').read_text())
    assert built_manifest['candidate']==json.loads(checked)['candidate']
    assert built_manifest['executionAuthorized'] is False
    assert built_manifest['activationAuthorized'] is False
    bundle=root/('grace-document-reading-q-'+sha);shutil.copytree(prepared,bundle);shutil.copy2(binary,bundle/'grace-document-reading-eval')
    transport=block('Выполнить один разрешённый Q4')
    for scenario in ['good','private','overflow','loss','auth_fail','scp_fail']:
        log.write_text('');cx=dict(base,FAKE_SSH=scenario)
        result=subprocess.run(['bash','-c',transport],cwd=REPO,env=cx,capture_output=True)
        assert (result.returncode==0)==(scenario=='good'),(scenario,result.stdout,result.stderr)
        saved=(root/'document-reading-summary.json').read_bytes()
        assert b'PRIVATE' not in saved+result.stdout+result.stderr
        assert len(log.read_text().splitlines())<=3
        assert not (root/'document-reading-stdout.private').exists() and not (root/'document-reading-stderr.private').exists()
        if scenario!='good':assert json.loads(saved)['status']=='report_unavailable'
    # Real wrapper, lock, checksum, no-clobber marker and cleanup; Docker is fake.
    remote=Path('/tmp/grace-document-reading-q-'+sha)
    assert not remote.exists(),'retain uncertain preexisting bundle'
    try:
        for scenario in ['good','manifest','source','tamper','symlink','spent','lock','run_fail','changed']:
            shutil.copytree(prepared,remote);shutil.copy2(binary,remote/'grace-document-reading-eval');shutil.copy2(WRAPPER,remote/'run.sh');(remote/'SOURCE_SHA').write_text(sha+'\n')
            names=['grace-document-reading-eval','corpus.json','MANIFEST.json','SOURCE_SHA','run.sh']
            (remote/'SHA256SUMS').write_text(''.join(hashlib.sha256((remote/n).read_bytes()).hexdigest()+'  '+n+'\n' for n in names))
            runtime=root/scenario;shared=runtime/'shared';shared.mkdir(parents=True)
            dockerlog=runtime/'docker';seen=runtime/'seen'
            cx=dict(base,GRACE_ROOT=str(runtime),GRACE_STATEMENT_OPERATOR_OWNER_ID=base['OPERATOR_OWNER_ID'],DOCKER_COMMANDS=str(dockerlog),INSPECT_SEEN=str(seen),FAKE_WRAPPER=scenario)
            approved=manifest;authorized=sha
            if scenario=='manifest':approved='0'*64
            if scenario=='source':authorized='0'*40
            if scenario=='tamper':(remote/'corpus.json').write_text('PRIVATE')
            if scenario=='symlink':(remote/'corpus.json').unlink();(remote/'corpus.json').symlink_to(prepared/'corpus.json')
            if scenario=='spent':(shared/'document-reading-spec0056-q.started').write_text('consumed')
            held=None
            if scenario=='lock':
                import fcntl
                held=open(shared/'statement-screening.lock','w');fcntl.flock(held,fcntl.LOCK_EX|fcntl.LOCK_NB)
            result=subprocess.run(['sh',str(WRAPPER),str(remote),sha,authorized,approved,'b'*40],cwd=REPO,env=cx,capture_output=True)
            if held:held.close()
            assert (result.returncode==0)==(scenario=='good'),(scenario,result.stdout,result.stderr)
            assert b'PRIVATE' not in result.stdout+result.stderr
            if scenario in ['good','run_fail','changed']:
                assert (shared/'document-reading-spec0056-q.started').exists()
                assert (shared/'document-reading-spec0056-q.private.json').exists()
                assert 'rm -rf '+str(remote) in dockerlog.read_text()
                assert not remote.exists()
            else:
                assert not dockerlog.exists()
            if remote.exists():shutil.rmtree(remote)
    finally:
        if remote.exists():shutil.rmtree(remote)
print('SPEC-0056 Q workflow/wrapper/CLI: source, one-shot, budget projection, privacy and cleanup PASS; actual SSH/provider calls=0')
