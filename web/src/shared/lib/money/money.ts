export function formatMoney(valueMinor: number | null, currency: string) {
  if (valueMinor === null) return 'Не сверён'
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency }).format(valueMinor / 100)
}
// Decimal-string minor units ("-123456") → "−1 234,56 ₽" without float, for values
// that may exceed Number.MAX_SAFE_INTEGER.
export function formatMinorText(valueMinor: string, currency: string) {
  const n = BigInt(valueMinor)
  const positive = n < 0n ? -n : n
  return `${n < 0n ? '−' : ''}${(positive / 100n).toLocaleString('ru-RU')},${(positive % 100n).toString().padStart(2, '0')} ${currency}`
}
// Decimal-string US dollar micros ("1500000") → "$1.50".
export function formatUsdMicros(valueMicros: string) {
  const n = BigInt(valueMicros)
  const fraction = (n % 1000000n).toString().padStart(6, '0').replace(/0+$/, '').padEnd(2, '0')
  return `$${n / 1000000n}.${fraction}`
}
// Accepts grouped input as MoneyInput renders it ("20 000 000,50", NBSP, "−").
export function parseMinor(text: string) {
  const normalized = text.replace(/\s/g, '').replace(/−/g, '-')
  const match = /^(-?)(\d+)(?:[.,](\d{1,2}))?$/.exec(normalized)
  if (!match) return null
  const value = BigInt(match[2]) * 100n + BigInt((match[3] ?? '').padEnd(2, '0'))
  const signed = match[1] ? -value : value
  return signed > BigInt(Number.MAX_SAFE_INTEGER) || signed < BigInt(Number.MIN_SAFE_INTEGER)
    ? null
    : Number(signed)
}
// Receipt quantity → integer thousandths ("1,5" → 1500), without float.
export function parseMilli(text: string) {
  const match = /^(\d+)(?:[.,](\d{1,3}))?$/.exec(text.replace(/\s/g, ''))
  if (!match) return null
  const value = BigInt(match[1]) * 1000n + BigInt((match[2] ?? '').padEnd(3, '0'))
  return value > BigInt(Number.MAX_SAFE_INTEGER) ? null : Number(value)
}
// Integer minor units → editable text ("1897" → "18,97") without float division.
export function minorToInput(valueMinor: number | null) {
  if (valueMinor === null) return ''
  const negative = valueMinor < 0
  const digits = String(Math.abs(valueMinor)).padStart(3, '0')
  return `${negative ? '-' : ''}${digits.slice(0, -2)},${digits.slice(-2)}`
}
