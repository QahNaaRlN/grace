/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

// Vitest stubs CSS imports (even ?raw) and runs from web/, so read the file.
const css = readFileSync(join(process.cwd(), 'src/app/styles/index.css'), 'utf8').replace(/\r\n/g, '\n')

// SPEC-0035 AC1/AC4/AC6: the palette in index.css itself meets the contrast
// floors in both themes, so a later edit cannot quietly break them.
function block(selector: string) {
  const match = new RegExp(`(?:^|\\n)${selector.replace('.', '\\.')} \\{([^}]*)\\}`).exec(css)
  if (!match) throw new Error(`missing ${selector}`)
  return Object.fromEntries([...match[1].matchAll(/--([\w-]+):\s*([^;]+);/g)].map(m => [m[1], m[2].trim()]))
}
const light = block(':root')
const dark = { ...light, ...block('.dark') }
function resolve(theme: Record<string, string>, name: string, depth = 0): string {
  const value = theme[name]
  if (!value) throw new Error(`missing --${name}`)
  const alias = /^var\(--([\w-]+)\)$/.exec(value)
  return alias && depth < 8 ? resolve(theme, alias[1], depth + 1) : value
}
function luminance(hex: string) {
  const full = hex.length === 4 ? `#${[...hex.slice(1)].map(c => c + c).join('')}` : hex
  const [r, g, b] = [1, 3, 5].map(i => parseInt(full.slice(i, i + 2), 16) / 255)
    .map(c => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}
function contrast(theme: Record<string, string>, a: string, b: string) {
  const [x, y] = [luminance(resolve(theme, a)), luminance(resolve(theme, b))]
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05)
}

const text: [string, string][] = [
  ['foreground', 'background'], ['foreground', 'card'], ['muted-foreground', 'background'],
  ['muted-foreground', 'card'], ['muted-foreground', 'muted'], ['muted-foreground', 'secondary'],
  ['primary', 'card'], ['primary', 'background'], ['primary-foreground', 'primary'],
  ['selected-foreground', 'selected'], ['accent-foreground', 'accent'], ['secondary-foreground', 'secondary'],
  ['destructive', 'card'], ['destructive', 'destructive-subtle'], ['destructive-foreground', 'destructive'],
  ['success', 'card'], ['success', 'success-subtle'], ['warning', 'card'], ['warning', 'warning-subtle'],
  ['info', 'card'], ['info', 'info-subtle'], ['unknown', 'unknown-subtle'], ['foreground', 'popover'],
]
const graphics: [string, string][] = [
  ['input', 'card'], ['ring', 'card'], ['ring', 'background'],
  ...[1, 2, 3, 4, 5].map(n => [`chart-${n}`, 'card'] as [string, string]),
]

describe.each([['light', light], ['dark', dark]] as const)('SPEC-0035 palette, %s theme', (_, theme) => {
  it.each(text)('%s on %s is at least 4.5:1', (a, b) => {
    expect(contrast(theme, a, b)).toBeGreaterThanOrEqual(4.5)
  })
  it.each(graphics)('%s on %s is at least 3:1', (a, b) => {
    expect(contrast(theme, a, b)).toBeGreaterThanOrEqual(3)
  })
  it('separates the current item from hover', () => {
    expect(contrast(theme, 'selected', 'accent')).toBeGreaterThanOrEqual(1.15)
  })
  it('alternates chart lightness so neighbours differ without hue', () => {
    for (const n of [1, 2, 3, 4]) expect(contrast(theme, `chart-${n}`, `chart-${n + 1}`)).toBeGreaterThanOrEqual(1.5)
  })
})
