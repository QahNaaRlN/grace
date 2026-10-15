// Runs Steiger and compares its diagnostics with steiger.baseline.json (SPEC-0048).
//
// Steiger's own per-file "off" removes files from the rule's view of the project, which
// also hides new imports that target those files. The baseline is therefore matched per
// diagnostic (rule, file, message) after Steiger has checked the whole tree.
//
//   node scripts/lint-fsd.mjs          fail on new diagnostics or stale baseline entries
//   node scripts/lint-fsd.mjs --prune  remove fixed entries from the baseline

import { spawnSync } from 'node:child_process'
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const baselinePath = resolve(webRoot, 'steiger.baseline.json')
const prune = process.argv.includes('--prune')

const run = spawnSync('npx', ['steiger', './src', '--reporter', 'json'], {
  cwd: webRoot,
  encoding: 'utf8',
  shell: process.platform === 'win32',
  stdio: ['ignore', 'pipe', 'inherit'],
  maxBuffer: 64 * 1024 * 1024,
})
let diagnostics
try {
  diagnostics = JSON.parse(run.stdout || '[]')
} catch {
  console.error(run.stdout)
  console.error('lint:fsd: Steiger did not return a JSON report.')
  process.exit(2)
}

const key = (d) =>
  [d.ruleName, relative(webRoot, d.location.path).replace(/\\/g, '/'), d.message].join(' | ')

const baseline = existsSync(baselinePath) ? JSON.parse(readFileSync(baselinePath, 'utf8')) : []
const remaining = new Map()
for (const entry of baseline) remaining.set(entry, (remaining.get(entry) ?? 0) + 1)

const fresh = []
for (const d of diagnostics) {
  const k = key(d)
  const left = remaining.get(k) ?? 0
  if (left > 0) remaining.set(k, left - 1)
  else fresh.push(k)
}
const stale = [...remaining].flatMap(([k, n]) => Array(n).fill(k))

if (prune && stale.length) {
  const kept = [...baseline]
  for (const k of stale) kept.splice(kept.indexOf(k), 1)
  writeFileSync(baselinePath, JSON.stringify(kept, null, 2) + '\n')
  console.log(`lint:fsd: removed ${stale.length} fixed entries from steiger.baseline.json.`)
} else if (stale.length) {
  console.error('lint:fsd: fixed violations are still listed in steiger.baseline.json:')
  for (const k of stale) console.error(`  ${k}`)
  console.error('Run `npm run lint:fsd -- --prune` to remove them.')
}

if (fresh.length) {
  console.error('lint:fsd: new Feature-Sliced Design violations (not in the baseline):')
  for (const k of fresh) console.error(`  ${k}`)
}

if (fresh.length || (stale.length && !prune)) process.exit(1)
console.log(`lint:fsd: OK (${diagnostics.length} baseline violations remain).`)
