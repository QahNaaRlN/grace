// SPEC-0036: presentation of amounts while they are typed. Only grouping is
// added; separators and digits are kept as typed, so an invalid amount
// ("1,2,3", three decimals, a minus where not allowed) still fails parseMinor
// and the form's validation instead of being silently repaired.

export const NBSP = String.fromCharCode(0xa0)
const kept = /[\d,.-]/

export function formatAmountText(raw: string) {
  const text = raw.replace(/−/g, '-').replace(/[^\d,.-]/g, '')
  const negative = text.startsWith('-')
  const unsigned = text.replace(/-/g, '')
  const separator = unsigned.search(/[.,]/)
  const whole = separator < 0 ? unsigned : unsigned.slice(0, separator)
  const rest = separator < 0 ? '' : `,${unsigned.slice(separator + 1)}`
  return `${negative ? '-' : ''}${whole.replace(/\B(?=(\d{3})+(?!\d))/g, NBSP)}${rest}`
}

// For APIs that take the amount as decimal text (receipt drafts): drop grouping only.
export function ungroupAmount(text: string) {
  return text.replace(/\s/g, '')
}

// Where the caret belongs in `next` so it stays after the same typed character.
export function caretAfterFormat(raw: string, caret: number, next: string) {
  let target = 0
  for (const char of raw.slice(0, caret).replace(/−/g, '-')) if (kept.test(char)) target += 1
  let position = 0
  for (let seen = 0; position < next.length && seen < target; position += 1)
    if (kept.test(next[position])) seen += 1
  return position
}
