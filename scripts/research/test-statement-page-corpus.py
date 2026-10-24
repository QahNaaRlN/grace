#!/usr/bin/env python3
"""No Docker/model calls. Check pinned corpus and actual sandbox command guards."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('page_corpus', Path(__file__).with_name('prepare-statement-page-corpus.py'))
corpus = importlib.util.module_from_spec(spec)
spec.loader.exec_module(corpus)

class CorpusTests(unittest.TestCase):
    def test_frozen_source_data_and_hashes(self):
        corpus.check()
        data = corpus.full32()
        self.assertEqual(32, len(data['rows']))
        self.assertEqual(32, len({r['anchor'][0]['quote'] for r in data['rows']}))
        self.assertEqual({1, 2, 3}, {r['anchor'][0]['page'] for r in data['rows']})
        self.assertEqual([1, 2], [e['page'] for e in data['rows'][10]['fields']['description']['evidence']])
        self.assertEqual('-100.00', data['rows'][1]['fields']['accountAmount']['value'])
        self.assertEqual('-100.00', data['rows'][2]['fields']['accountAmount']['value'])
        self.assertEqual('-2.00', data['rows'][8]['fields']['transactionAmount']['value'])
        manifest = json.loads((corpus.OUT / 'manifest.json').read_text())
        self.assertEqual(manifest['fontSHA256'], corpus.FONT_SHA256)
        self.assertEqual('sha256:5874a467b6446d165c933d99f65259c77c6db3dc9b2f9890ea96a1f37c7d97c3', manifest['rendererImage'])
        for path in manifest['files']:
            self.assertTrue(path.startswith('internal/'))
        # The visual control is the same document and pixels, without text.
        self.assertIsNone(manifest['cases'][2]['text'])
        self.assertEqual(manifest['cases'][0]['pages'], manifest['cases'][2]['pages'])

    def test_renderer_sandbox_is_the_actual_invocation(self):
        calls = []
        with patch.object(corpus.os, 'getuid', return_value=10001), patch.object(corpus.subprocess, 'run', side_effect=lambda *a, **kw: calls.append((a, kw))):
            corpus.sandbox(Path('/public/full32.pdf'), Path('/public/output'))
        self.assertEqual(1, len(calls))
        (argv,), options = calls[0]
        for flag in ('--pull=never', '--network=none', '--read-only', '--cap-drop=ALL', '--security-opt=no-new-privileges:true', '--pids-limit=32', '--memory=512m', '--cpus=1'):
            self.assertIn(flag, argv)
        self.assertNotEqual('0:0', argv[argv.index('--user') + 1])
        self.assertIn('/public/full32.pdf:/input.pdf:ro', argv)
        self.assertNotIn('--privileged', argv)
        self.assertNotIn('--env-file', argv)
        self.assertEqual({'PATH': '/usr/bin:/bin'}, options['env'])
        self.assertEqual(90, options['timeout'])
        self.assertTrue(options['check'])
        self.assertIn('-scale-to 1800', argv[-1])
        self.assertNotIn('-crop', argv[-1])
        self.assertNotIn('curl', argv[-1])

if __name__ == '__main__':
    unittest.main()
