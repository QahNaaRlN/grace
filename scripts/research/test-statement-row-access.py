#!/usr/bin/env python3
"""S1.21 A1-A3: actual helper/remote file operations and workflow blocks, all SSH/gh/keygen fake."""
import base64
import importlib.util
import json
import os
from pathlib import Path
import shlex
import stat
import subprocess
import tempfile
import unittest
from unittest.mock import patch

REPO = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('row_access', REPO / 'scripts/research/setup-statement-row-ssh.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
SHA = 'a' * 40
BLOB = base64.b64encode(b'\0\0\0\x0bssh-ed25519\0\0\0\x20' + bytes(range(32))).decode()
PRIVATE = b'-----BEGIN OPENSSH PRIVATE KEY-----\nTEST_ONLY_NOT_A_REAL_KEY\n'


class FakeRun:
    def __init__(self, mode='success'):
        self.mode = mode
        self.calls = []
        self.keys = []
        self.secret_checks = 0
        self.ssh_calls = 0
        self.marker_calls = []

    def marker(self, path):
        self.marker_calls.append(path)
        if self.mode == 'repeat':
            raise FileExistsError('PRIVATE')

    def __call__(self, argv, data, timeout):
        self.calls.append((argv, data))
        if argv[0] == 'git':
            if 'rev-parse' in argv:
                return 0, ((SHA if self.mode != 'source' else 'b' * 40) + '\n').encode()
            if 'show' in argv:
                name = argv[-1].split(':', 1)[1]
                return 0, b'changed' if self.mode == 'file_mismatch' else (REPO / name).read_bytes()
            if 'status' in argv:
                return 0, b' M file\n' if self.mode == 'dirty' else b''
            return (1 if self.mode == 'ancestor' else 0), b''
        if argv[:3] == ['gh', 'run', 'list']:
            return 0, b'[{"databaseId":1}]'
        if argv[:3] == ['gh', 'run', 'view']:
            ci = dict(headSha=SHA, status='completed', conclusion='success', jobs=[dict(name=n, status='completed', conclusion='success') for n in helper.JOBS])
            if self.mode == 'ci': ci['jobs'][0]['conclusion'] = 'skipped'
            return 0, json.dumps(ci).encode()
        if argv[:3] == ['gh', 'secret', 'list']:
            self.secret_checks += 1
            present = self.mode == 'exists' or (self.mode == 'race' and self.secret_checks >= 2) or self.secret_checks >= 3
            if self.mode in ('verify', 'cleanup_failure') and self.secret_checks == 3:
                return 1, b'PRIVATE'
            return 0, json.dumps([{'name': helper.SECRET}] if present else []).encode()
        if argv[0] == 'ssh-keygen':
            assert argv[:5] == ['ssh-keygen', '-q', '-t', 'ed25519', '-N'] and argv[5] == ''
            key = Path(argv[-1]); self.keys.append(key)
            assert stat.S_IMODE(key.parent.stat().st_mode) == 0o700
            if self.mode == 'keygen': return 1, b'PRIVATE'
            key.write_bytes(PRIVATE); key.chmod(0o644 if self.mode == 'permissions' else 0o600)
            key.with_suffix('.pub').write_text('ssh-ed25519 ' + BLOB + ' grace-statement-s121-row-ci-' + SHA + '\n')
            if self.mode == 'public': key.with_suffix('.pub').write_text('PRIVATE invalid\n')
            return 0, b''
        if argv[0] == 'ssh':
            self.ssh_calls += 1
            assert helper.HOST == argv[-2]
            assert '-oStrictHostKeyChecking=yes' in argv and '-oConnectionAttempts=1' in argv
            assert '-oBatchMode=yes' in argv and data == helper.REMOTE.encode()
            remote = shlex.split(argv[-1])
            assert remote[:2] == ['python3', '-'] and remote[3:] == [SHA, BLOB]
            if remote[2] == 'install':
                assert self.ssh_calls == 1
                if self.mode == 'ssh': return 255, b'PRIVATE'
                if self.mode == 'reply': return 0, b'PRIVATE'
                if self.mode == 'interrupt': raise KeyboardInterrupt()
                return 0, b'{"status":"installed"}'
            assert remote[2] == 'rollback' and self.ssh_calls == 2
            return (1, b'PRIVATE') if self.mode == 'cleanup_failure' else (0, b'{"status":"removed"}')
        if argv[:3] == ['gh', 'secret', 'set']:
            assert argv[3:] == [helper.SECRET, '--repo', helper.REPOSITORY, '--env', 'production']
            assert data == PRIVATE and PRIVATE.decode() not in ' '.join(argv)
            return (1, b'PRIVATE') if self.mode == 'set' else (0, b'')
        if argv[:3] == ['gh', 'secret', 'delete']:
            assert argv[3] == helper.SECRET and self.mode in ('verify', 'cleanup_failure')
            return (1, b'PRIVATE') if self.mode == 'cleanup_failure' else (0, b'')
        raise AssertionError(argv)


class BootstrapTests(unittest.TestCase):
    def test_safe_cli_defaults_and_authorization(self):
        with tempfile.TemporaryDirectory() as tmp:
            tools = Path(tmp) / 'bin'; tools.mkdir()
            spy = Path(tmp) / 'called'
            for name in ('ssh', 'ssh-keygen', 'gh', 'git'):
                path = tools / name
                path.write_text('#!/bin/sh\nprintf called >> "$SPY"\nexit 99\n'); path.chmod(0o755)
            env = dict(os.environ, PATH=str(tools) + ':' + os.environ['PATH'], HOME='/does-not-exist', SSH_AUTH_SOCK='/invalid', GH_DEBUG='api', SPY=str(spy))
            for args, success in [([], True), (['--source', SHA], True), (['--mode', 'bootstrap'], False), (['--mode', 'bootstrap', '--source', SHA, '--manifest', '/missing'], False), (['--mode', 'bootstrap', '--source', "bad'PRIVATE", '--manifest', '/missing', '--execute-approved-setup'], False)]:
                r = subprocess.run(['python3', '-B', str(REPO / 'scripts/research/setup-statement-row-ssh.py'), *args], env=env, capture_output=True)
                self.assertEqual(r.returncode == 0, success)
                self.assertNotIn(b'PRIVATE', r.stdout + r.stderr)
                self.assertFalse(spy.exists())

    def test_bootstrap_caps_cleanup_and_uncertainty(self):
        for mode in ('success', 'source', 'dirty', 'ancestor', 'file_mismatch', 'ci', 'exists', 'repeat', 'keygen', 'permissions', 'public', 'race', 'ssh', 'reply', 'set', 'verify', 'cleanup_failure', 'interrupt'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as tmp:
                path = Path(tmp) / 'manifest.json'; path.write_text(json.dumps(helper.manifest(SHA)))
                runner = FakeRun(mode)
                result = helper.bootstrap(SHA, path, runner, runner.marker)
                self.assertEqual(result['status'] == 'CONFIGURED', mode == 'success')
                self.assertFalse(result['activationAuthorized']); self.assertFalse(result['pilotAuthorized'])
                self.assertLessEqual(result['installSSH'], 1); self.assertLessEqual(result['rollbackSSH'], 1)
                self.assertNotIn('PRIVATE', json.dumps(result))
                self.assertTrue(all(not key.parent.exists() for key in runner.keys))
                self.assertEqual(len(runner.marker_calls), 1)
                self.assertLessEqual(sum(call[0][:3] == ['gh', 'secret', 'set'] for call in runner.calls), 1)
                if mode in ('source', 'dirty', 'ancestor', 'file_mismatch', 'ci', 'exists', 'repeat', 'keygen', 'permissions', 'public'):
                    self.assertEqual(runner.ssh_calls, 0)
                if mode in ('race', 'ssh', 'reply', 'set', 'verify', 'cleanup_failure', 'interrupt'):
                    self.assertEqual(runner.ssh_calls, 2)
                if mode == 'race':
                    self.assertFalse(any(c[0][:3] == ['gh', 'secret', 'delete'] for c in runner.calls))
                if mode == 'set':
                    self.assertEqual(result['secretState'], 'unknown')
                    self.assertIsNone(result['secretRollback'])
                if mode == 'verify':
                    self.assertTrue(result['secretRollback']); self.assertTrue(result['rootRollback'])
                if mode == 'cleanup_failure':
                    self.assertFalse(result['secretRollback']); self.assertFalse(result['rootRollback'])

    def test_manifest_and_one_shot_before_commands(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'manifest'
            data = helper.manifest(SHA); data['maximumInstallSSH'] = 2
            path.write_text(json.dumps(data)); runner = FakeRun()
            self.assertEqual(helper.bootstrap(SHA, path, runner, runner.marker)['status'], 'REFUSED')
            self.assertFalse(runner.calls); self.assertEqual(len(runner.marker_calls), 1)
            data = helper.manifest(SHA); data['executionAuthorized'] = 0
            path.write_text(json.dumps(data))
            self.assertEqual(helper.bootstrap(SHA, path, runner, runner.marker)['status'], 'REFUSED')
            self.assertFalse(runner.calls)
            path.write_text('{"source":"' + SHA + '","source":"' + SHA + '"}')
            self.assertEqual(helper.bootstrap(SHA, path, runner, runner.marker)['status'], 'REFUSED')
            self.assertFalse(runner.calls)
            marker = Path(tmp) / 'attempt'
            helper.attempt_once(marker)
            self.assertEqual(stat.S_IMODE(marker.stat().st_mode), 0o600)
            with self.assertRaises(FileExistsError): helper.attempt_once(marker)


class RemoteTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name); self.root.chmod(0o700)
        (self.root / '.ssh').mkdir(mode=0o700)
        self.keys = self.root / '.ssh/authorized_keys'
        self.original = b'# foreign public key\nrestrict ssh-ed25519 OTHER foreign-owner\n'
        self.keys.write_bytes(self.original); self.keys.chmod(0o600)
        namespace = {'__name__': 'offline_test'}
        exec(helper.REMOTE, namespace)
        self.update = lambda operation, blob=BLOB: namespace['update'](self.root, SHA, blob, operation, owner=os.getuid())

    def tearDown(self): self.temp.cleanup()

    def test_install_and_own_tail_rollback_preserve_foreign_bytes(self):
        self.assertEqual(self.update('install'), 'installed')
        installed = self.keys.read_bytes()
        self.assertTrue(installed.startswith(self.original))
        self.assertIn(b'restrict ssh-ed25519 ' + BLOB.encode(), installed)
        with self.assertRaises(ValueError): self.update('install')
        self.assertEqual(self.update('rollback'), 'removed')
        self.assertEqual(self.keys.read_bytes(), self.original)
        self.assertEqual(self.update('rollback'), 'absent')

    def test_refuse_unsafe_files_and_ambiguous_rollback(self):
        self.keys.chmod(0o644)
        with self.assertRaises(ValueError): self.update('install')
        self.keys.chmod(0o600)
        self.keys.write_bytes(self.original.rstrip(b'\n'))
        with self.assertRaises(ValueError): self.update('install')
        self.keys.write_bytes(self.original)
        self.update('install')
        with self.keys.open('ab') as out: out.write(b'# concurrent foreign line\n')
        before = self.keys.read_bytes()
        with self.assertRaises(ValueError): self.update('rollback')
        self.assertEqual(self.keys.read_bytes(), before)

    def test_symlink_and_foreign_tag_refused(self):
        other = self.root / 'other'; other.write_bytes(self.original)
        self.keys.unlink(); self.keys.symlink_to(other)
        with self.assertRaises(OSError): self.update('install')
        self.assertEqual(other.read_bytes(), self.original)
        self.keys.unlink(); self.keys.write_bytes(('ssh-ed25519 OTHER grace-statement-s121-row-ci-' + SHA + '\n').encode()); self.keys.chmod(0o600)
        with self.assertRaises(ValueError): self.update('install')
        with self.assertRaises(ValueError): self.update('rollback')


def block(workflow, name):
    body = workflow.split('      - name: ' + name + '\n', 1)[1].split('        run: |\n', 1)[1]
    lines = []
    for line in body.splitlines():
        if line and not line.startswith('          '): break
        lines.append(line[10:])
    return '\n'.join(lines) + '\n'


class WorkflowTests(unittest.TestCase):
    def test_actual_shell_blocks_with_fake_ssh_only(self):
        workflow = (REPO / '.github/workflows/statement-row-access.yml').read_text()
        self.assertIn('default: false', workflow)
        self.assertIn('if: ${{ inputs.execute_approved_access }}', workflow)
        self.assertIn('workflow_dispatch:', workflow); self.assertNotIn('pull_request:', workflow)
        self.assertIn('group: production', workflow); self.assertIn('environment: production', workflow)
        self.assertIn('${{ secrets.PRODUCTION_STATEMENT_ROW_SSH_KEY }}', workflow)
        self.assertNotIn('${{ secrets.PRODUCTION_SSH_KEY }}', workflow)
        for forbidden in ('scp ', 'docker ', 'statement-row-s121.sh', 'OPERATOR_OWNER_ID'):
            self.assertNotIn(forbidden, workflow)
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); tools = root / 'bin'; tools.mkdir(); log = root / 'ssh.jsonl'; ci = root / 'ci.json'
            for name, body in {
                'git': 'test "${ANCESTOR:-true}" = true\n',
                'gh': 'case "$*" in run\\ list*) echo 123 ;; run\\ view*) cat "$CI_FIXTURE" ;; *) exit 9 ;; esac\n',
            }.items():
                path = tools / name; path.write_text('#!/bin/sh\nset -eu\n' + body); path.chmod(0o755)
            ssh = tools / 'ssh'
            ssh.write_text('#!/usr/bin/env python3\nimport json,os,sys\nwith open(os.environ["SSH_LOG"],"a") as f: f.write(json.dumps(sys.argv[1:])+"\\n")\nprint("PRIVATE", file=sys.stderr)\nprint("PRIVATE")\nsys.exit(int(os.environ["SSH_CODE"]))\n'); ssh.chmod(0o755)
            base = dict(os.environ, PATH=str(tools) + ':' + os.environ['PATH'], RUNNER_TEMP=tmp, GITHUB_SHA=SHA, AUTHORIZED_SHA=SHA, GITHUB_RUN_ATTEMPT='1', ROW_SSH_KEY='TEST_ONLY_FAKE_KEY', CI_FIXTURE=str(ci), SSH_LOG=str(log), SSH_CODE='0')
            for mode in ('success', 'failure', 'timeout', 'source', 'retry', 'ancestor', 'ci', 'ci_source', 'missing_key'):
                with self.subTest(mode=mode):
                    env = base.copy(); log.unlink(missing_ok=True)
                    resultpath = root / 's121-access-result.json'; resultpath.unlink(missing_ok=True)
                    data = dict(headSha=SHA, status='completed', conclusion='success', jobs=[dict(name=n, status='completed', conclusion='success') for n in helper.JOBS])
                    if mode == 'source': env['AUTHORIZED_SHA'] = 'b' * 40
                    if mode == 'retry': env['GITHUB_RUN_ATTEMPT'] = '2'
                    if mode == 'ancestor': env['ANCESTOR'] = 'false'
                    if mode == 'ci': data['jobs'][0]['conclusion'] = 'skipped'
                    if mode == 'ci_source': data['headSha'] = 'b' * 40
                    if mode == 'missing_key': env['ROW_SSH_KEY'] = ''
                    if mode == 'failure': env['SSH_CODE'] = '255'
                    if mode == 'timeout': env['SSH_CODE'] = '124'
                    ci.write_text(json.dumps(data))
                    final = None
                    try:
                        for name in ('Verify exact authorized source and CI 5/5', 'Configure dedicated pinned SSH', 'Verify root access once'):
                            final = subprocess.run(['bash', '-e', '-o', 'pipefail', '-c', block(workflow, name)], cwd=REPO, env=env, capture_output=True)
                            self.assertNotIn(b'PRIVATE', final.stdout + final.stderr)
                            if final.returncode: break
                        self.assertEqual(final.returncode == 0, mode == 'success')
                        if mode in ('success', 'failure', 'timeout'):
                            lines = log.read_text().splitlines(); self.assertEqual(len(lines), 1)
                            args = json.loads(lines[0])
                            self.assertEqual(args[-2:], [helper.HOST, 'test "$(id -u)" -eq 0'])
                            for flag in ('-oIdentityAgent=none', '-oIdentitiesOnly=yes', '-oConnectionAttempts=1', '-oBatchMode=yes', '-oStrictHostKeyChecking=yes'):
                                self.assertIn(flag, args)
                            result = json.loads(resultpath.read_text())
                            self.assertEqual(result['source'], SHA)
                            self.assertEqual(result['sshVerified'], mode == 'success')
                            self.assertFalse(result['pilotAuthorized']); self.assertFalse(result['activationAuthorized'])
                            key = root / 's121-row-access-ssh/key'
                            self.assertEqual(stat.S_IMODE(key.stat().st_mode), 0o600)
                        else:
                            self.assertFalse(log.exists()); self.assertFalse(resultpath.exists())
                    finally:
                        # Execute the actual cleanup body, not a substitute implementation.
                        cleanup = workflow.split('      - name: Remove ephemeral SSH key\n', 1)[1].split('        run: ', 1)[1].splitlines()[0]
                        subprocess.run(['bash', '-eu', '-c', cleanup], env=env, check=True)
                        self.assertFalse((root / 's121-row-access-ssh').exists())


if __name__ == '__main__':
    unittest.main()
