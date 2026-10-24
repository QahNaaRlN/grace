#!/usr/bin/env python3
"""S1.21 A1: offline manifest by default; live bootstrap needs separate B4."""
import argparse
import base64
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shlex
import signal
import stat
import subprocess
import sys
import tempfile

REPO = Path(__file__).resolve().parents[2]
HOST = 'root@195.133.195.190'
SECRET = 'PRODUCTION_STATEMENT_ROW_SSH_KEY'
REPOSITORY = 'QahNaaRlN/grace'
VERSION = 'statement.row-access.bootstrap.s121.v1'
JOBS = {'Quality gates', 'PostgreSQL integration gates', 'Build api image', 'Build web image', 'Build pdf-extractor image'}

# Sent over the separately approved agent SSH. No remote file is installed.
REMOTE = r'''
import base64, fcntl, json, os, re, stat, sys
from pathlib import Path

def update(root, source, blob, operation, owner=0):
    if not re.fullmatch(r'[0-9a-f]{40}', source) or operation not in ('install', 'rollback'):
        raise ValueError('arguments')
    decoded = base64.b64decode(blob, validate=True)
    if len(decoded) != 51 or decoded[:19] != b'\0\0\0\x0bssh-ed25519\0\0\0\x20':
        raise ValueError('key')
    tag = 'grace-statement-s121-row-ci-' + source
    line = ('restrict ssh-ed25519 ' + blob + ' ' + tag + '\n').encode('ascii')
    for directory in (root, root / '.ssh'):
        st = directory.lstat()
        if not stat.S_ISDIR(st.st_mode) or st.st_uid != owner or st.st_mode & 0o022:
            raise ValueError('directory')
    target = root / '.ssh/authorized_keys'
    fd = os.open(target, os.O_RDWR | os.O_NOFOLLOW)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        st = os.fstat(fd)
        if not stat.S_ISREG(st.st_mode) or st.st_uid != owner or stat.S_IMODE(st.st_mode) != 0o600 or st.st_nlink != 1 or st.st_size > 1048576:
            raise ValueError('file')
        raw = os.read(fd, 1048577)
        if len(raw) > 1048576 or (raw and not raw.endswith(b'\n')):
            raise ValueError('content')
        if operation == 'install':
            if tag.encode() in raw or blob.encode() in raw:
                raise ValueError('exists')
            os.lseek(fd, 0, os.SEEK_END)
            if os.write(fd, line) != len(line):
                raise ValueError('partial_write')
            os.fsync(fd)
            return 'installed'
        if tag.encode() not in raw and blob.encode() not in raw:
            return 'absent'
        if raw.splitlines(keepends=True).count(line) != 1 or not raw.endswith(line):
            raise ValueError('rollback_ambiguous')
        os.ftruncate(fd, len(raw) - len(line))
        os.fsync(fd)
        return 'removed'
    finally:
        os.close(fd)

if __name__ == '__main__':
    try:
        if os.geteuid() != 0 or len(sys.argv) != 4:
            raise ValueError('principal')
        status = update(Path('/root'), sys.argv[2], sys.argv[3], sys.argv[1])
        print(json.dumps({'status': status}))
    except Exception:
        print('{"status":"refused"}')
        sys.exit(1)
'''


class Refusal(Exception):
    pass


def checked_json(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise Refusal('duplicate_json_key')
            result[key] = value
        return result
    if len(raw) > 1048576:
        raise Refusal('oversize_result')
    return json.loads(raw, object_pairs_hook=pairs)


def manifest(source=None):
    if source is not None and not re.fullmatch(r'[0-9a-f]{40}', source):
        raise Refusal('source_invalid')
    paths = ['scripts/research/setup-statement-row-ssh.py', '.github/workflows/statement-row-access.yml', '.github/workflows/statement-row-pilot.yml', 'deployments/production/known_hosts']
    return dict(version=VERSION, source=source, executionAuthorized=False, activationAuthorized=False,
                repository=REPOSITORY, environment='production', secret=SECRET, host=HOST,
                authorizedKeys='/root/.ssh/authorized_keys',
                tag=None if source is None else 'grace-statement-s121-row-ci-' + source,
                attemptPath=None if source is None else '/tmp/grace-s121-row-access-' + source + '.attempted',
                maximumInstallSSH=1, maximumRollbackSSH=1,
                files={name: hashlib.sha256((REPO / name).read_bytes()).hexdigest() for name in paths})


def system_run(argv, data=None, timeout=30):
    # Raw command output is private and never propagated to logs/results.
    env = dict(os.environ, GH_DEBUG='', GH_FORCE_TTY='')
    with tempfile.TemporaryFile() as out, tempfile.TemporaryFile() as err:
        result = subprocess.run(argv, input=data, stdout=out, stderr=err, timeout=timeout, env=env)
        out.seek(0)
        raw = out.read(1048577)
        if len(raw) > 1048576:
            raise Refusal('oversize_result')
        return result.returncode, raw


def command(run, argv, data=None, timeout=30):
    code, raw = run(argv, data, timeout)
    if code:
        raise Refusal('command_failed')
    return raw


def secret_names(run):
    data = checked_json(command(run, ['gh', 'secret', 'list', '--repo', REPOSITORY, '--env', 'production', '--json', 'name']))
    if not isinstance(data, list) or any(not isinstance(row, dict) or set(row) != {'name'} or not isinstance(row['name'], str) for row in data):
        raise Refusal('secret_metadata_invalid')
    return {row['name'] for row in data}


def source_gate(source, run):
    if command(run, ['git', '-C', str(REPO), 'rev-parse', 'HEAD']).decode().strip() != source:
        raise Refusal('source_mismatch')
    if command(run, ['git', '-C', str(REPO), 'status', '--porcelain', '--untracked-files=no']).strip():
        raise Refusal('source_dirty')
    command(run, ['git', '-C', str(REPO), 'merge-base', '--is-ancestor', source, 'origin/main'])
    for name, digest in manifest(source)['files'].items():
        committed = command(run, ['git', '-C', str(REPO), 'show', source + ':' + name])
        if hashlib.sha256(committed).hexdigest() != digest:
            raise Refusal('source_file_mismatch')
    runs = checked_json(command(run, ['gh', 'run', 'list', '--repo', REPOSITORY, '--workflow', 'ci.yml', '--commit', source, '--status', 'success', '--limit', '1', '--json', 'databaseId']))
    if not isinstance(runs, list) or len(runs) != 1 or type(runs[0].get('databaseId')) is not int or runs[0]['databaseId'] <= 0:
        raise Refusal('ci_unverified')
    ci = checked_json(command(run, ['gh', 'run', 'view', str(runs[0]['databaseId']), '--repo', REPOSITORY, '--json', 'headSha,status,conclusion,jobs']))
    jobs = ci.get('jobs', [])
    if ci.get('headSha') != source or ci.get('status') != 'completed' or ci.get('conclusion') != 'success' or len(jobs) != 5 or {j.get('name') for j in jobs} != JOBS or not all(j.get('status') == 'completed' and j.get('conclusion') == 'success' for j in jobs):
        raise Refusal('ci_unverified')


def ssh(run, source, blob, operation):
    argv = ['ssh', '-F', '/dev/null', '-oBatchMode=yes', '-oConnectionAttempts=1', '-oConnectTimeout=10',
            '-oPreferredAuthentications=publickey', '-oPasswordAuthentication=no', '-oKbdInteractiveAuthentication=no',
            '-oStrictHostKeyChecking=yes', '-oUserKnownHostsFile=' + str(REPO / 'deployments/production/known_hosts'),
            HOST, shlex.join(['python3', '-', operation, source, blob])]
    return checked_json(command(run, argv, REMOTE.encode(), timeout=25))


def attempt_once(path):
    fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
    os.close(fd)  # Never remove: refusal also consumes the separately approved setup.


def bootstrap(source, supplied_manifest, run=system_run, marker=attempt_once):
    result = dict(version=VERSION, source=source, status='REFUSED', error='bootstrap_failed',
                  activationAuthorized=False, pilotAuthorized=False, fingerprint=None,
                  installSSH=0, rollbackSSH=0, rootRollback=None, secretState='not_created', secretRollback=None)
    attempted = False
    created = False
    blob = None
    try:
        if not isinstance(source, str) or not re.fullmatch(r'[0-9a-f]{40}', source):
            raise Refusal('source_invalid')
        marker('/tmp/grace-s121-row-access-' + source + '.attempted')
        expected = manifest(source)
        if source is None or json.dumps(checked_json(Path(supplied_manifest).read_bytes()), sort_keys=True) != json.dumps(expected, sort_keys=True):
            raise Refusal('manifest_mismatch')
        source_gate(source, run)
        if SECRET in secret_names(run):
            raise Refusal('secret_already_exists')
        with tempfile.TemporaryDirectory(prefix='grace-s121-row-key-', dir='/tmp') as temp:
            key = Path(temp) / 'key'
            try:
                command(run, ['ssh-keygen', '-q', '-t', 'ed25519', '-N', '', '-C', expected['tag'], '-f', str(key)])
                st = key.lstat()
                if not stat.S_ISREG(st.st_mode) or stat.S_IMODE(st.st_mode) != 0o600 or st.st_uid != os.getuid() or st.st_nlink != 1 or st.st_size > 16384:
                    raise Refusal('key_file_invalid')
                private = key.read_bytes()
                if not private.startswith(b'-----BEGIN OPENSSH PRIVATE KEY-----\n'):
                    raise Refusal('key_file_invalid')
                public = key.with_suffix('.pub').read_text().split()
                if len(public) != 3 or public[0] != 'ssh-ed25519' or public[2] != expected['tag']:
                    raise Refusal('public_key_invalid')
                blob = public[1]
                decoded = base64.b64decode(blob, validate=True)
                if len(decoded) != 51 or decoded[:19] != b'\0\0\0\x0bssh-ed25519\0\0\0\x20':
                    raise Refusal('public_key_invalid')
                result['fingerprint'] = 'SHA256:' + base64.b64encode(hashlib.sha256(decoded).digest()).decode().rstrip('=')
                attempted = True
                result['installSSH'] = 1
                if ssh(run, source, blob, 'install') != {'status': 'installed'}:
                    raise Refusal('root_install_unverified')
                # Recheck immediately before the only secret PUT; never read secret values.
                if SECRET in secret_names(run):
                    raise Refusal('secret_already_exists')
                result['secretState'] = 'unknown'
                command(run, ['gh', 'secret', 'set', SECRET, '--repo', REPOSITORY, '--env', 'production'], private)
                created = True
                result['secretState'] = 'created'
                if SECRET not in secret_names(run):
                    raise Refusal('secret_write_unverified')
                result.update(status='CONFIGURED', error=None)
            except (Exception, KeyboardInterrupt) as error:
                result['error'] = str(error) if isinstance(error, Refusal) else 'bootstrap_failed'
                if created:
                    try:
                        command(run, ['gh', 'secret', 'delete', SECRET, '--repo', REPOSITORY, '--env', 'production'])
                        result['secretRollback'] = True
                        result['secretState'] = 'removed'
                    except (Exception, KeyboardInterrupt):
                        result['secretRollback'] = False
                        result['secretState'] = 'unknown'
                if attempted:
                    result['rollbackSSH'] = 1
                    try:
                        reply = ssh(run, source, blob, 'rollback')
                        result['rootRollback'] = reply in ({'status': 'removed'}, {'status': 'absent'})
                    except (Exception, KeyboardInterrupt):
                        result['rootRollback'] = False
                # TemporaryDirectory removes both new key files on every handled exit.
    except (Exception, KeyboardInterrupt) as error:
        result['error'] = str(error) if isinstance(error, Refusal) else 'bootstrap_failed'
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--mode', choices=['check', 'bootstrap'], default='check')
    parser.add_argument('--source')
    parser.add_argument('--manifest')
    parser.add_argument('--execute-approved-setup', action='store_true')
    args = parser.parse_args(argv)
    if args.mode == 'check':
        if args.execute_approved_setup or args.manifest:
            parser.error('check mode does not accept execution arguments')
        try:
            print(json.dumps(manifest(args.source), separators=(',', ':')))
            return 0
        except Exception:
            print('{"status":"REFUSED","error":"source_invalid"}')
            return 1
    # No source/config/agent/manifest access before this explicit separate B4 gate.
    if not args.execute_approved_setup or not args.source or not re.fullmatch(r'[0-9a-f]{40}', args.source) or not args.manifest:
        print('{"status":"REFUSED","error":"setup_not_authorized"}')
        return 1
    def interrupted(*_):
        raise KeyboardInterrupt()
    signal.signal(signal.SIGTERM, interrupted)
    result = bootstrap(args.source, args.manifest)
    print(json.dumps(result, separators=(',', ':')))
    return 0 if result['status'] == 'CONFIGURED' else 1


if __name__ == '__main__':
    sys.exit(main())
