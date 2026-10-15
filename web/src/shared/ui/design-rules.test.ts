/// <reference types="node" />
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// SPEC-0035 AC2/AC3/AC8: rules the design system keeps across all of web/src.
const root = join(process.cwd(), 'src')
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return name === 'generated' ? [] : sources(path)
    return /\.tsx?$/.test(name) && !/\.test\.tsx?$/.test(name) ? [path] : []
  })
}
const files = sources(root).map((path) => ({
  path: relative(root, path),
  text: readFileSync(path, 'utf8'),
}))
const offenders = (pattern: RegExp) =>
  files.filter((file) => pattern.test(file.text)).map((file) => file.path)

describe('SPEC-0035 design rules in web/src', () => {
  it('has no focus halo rings — focus is the global outline', () => {
    expect(offenders(/focus-visible:ring|focus-visible:outline-|ring-\[3px\]/)).toEqual([])
  })
  it('has a single press effect (no translate-y-px)', () => {
    expect(offenders(/translate-y-px/)).toEqual([])
  })
  it('sets type only on the scale (no arbitrary text sizes or clamp)', () => {
    expect(offenders(/text-\[[\d.]+(rem|px|em)\]|text-\[clamp|clamp\(/)).toEqual([])
  })
  it('uses only the button sizes that differ', () => {
    expect(offenders(/<Button\b[^>]*\bsize=["'](xs|sm|icon-xs|icon-sm)["']/)).toEqual([])
    // The rule itself must catch a shrunken size (guards against a vacuous pattern).
    expect(
      /<Button\b[^>]*\bsize=["'](xs|sm|icon-xs|icon-sm)["']/.test(
        '<Button variant="outline" size="sm">',
      ),
    ).toBe(true)
  })
  it('draws card borders at full strength', () => {
    expect(offenders(/border-border\/70/)).toEqual([])
  })
})
