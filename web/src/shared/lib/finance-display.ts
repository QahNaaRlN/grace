import type { Account, FinancialEvent } from '@/shared/api/types'
export function accountLabel(account: Account) {
  const card = account.paymentInstruments?.find(item => !item.disabledAt)
  return card ? `${card.bank} *${card.last4}` : account.name
}
export function formatMoney(valueMinor: number | null, currency: string) {
  if (valueMinor === null) return "Не сверён"
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency }).format(
    valueMinor / 100,
  )
}
export function dateText(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Asia/Yekaterinburg',
    dateStyle: 'medium',
  }).format(new Date(value.length === 10 ? `${value}T12:00:00+05:00` : value))
}
export function eventDateText(event: Pick<FinancialEvent, 'occurredAt' | 'occurredDate'>) {
  if (event.occurredDate) return `${dateText(event.occurredDate)} · время неизвестно`
  return event.occurredAt ? dateText(event.occurredAt) : 'Дата неизвестна'
}
export function eventTimeText(event: Pick<FinancialEvent, 'occurredAt' | 'occurredDate'>) {
  if (event.occurredDate) return eventDateText(event)
  return event.occurredAt ? new Date(event.occurredAt).toLocaleString('ru-RU', { timeZone: 'Asia/Yekaterinburg' }) : 'Дата неизвестна'
}
export function today() {
  return new Intl.DateTimeFormat('sv-SE', {
    timeZone: 'Asia/Yekaterinburg',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date())
}
// Accepts grouped input as MoneyInput renders it ("20 000 000,50", NBSP, "−").
export function parseMinor(text: string) {
  const normalized = text.replace(/\s/g, '').replace(/\u2212/g, '-')
  const match = /^(-?)(\d+)(?:[.,](\d{1,2}))?$/.exec(normalized)
  if (!match) return null
  const value =
    BigInt(match[2]) * 100n + BigInt((match[3] ?? '').padEnd(2, '0'))
  const signed = match[1] ? -value : value
  return signed > BigInt(Number.MAX_SAFE_INTEGER) ||
    signed < BigInt(Number.MIN_SAFE_INTEGER)
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
// Grace moments are Asia/Yekaterinburg (+05:00, no DST). datetime-local text ↔ RFC 3339.
export function localDateTimeInput(iso: string) {
  return new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Yekaterinburg', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(iso)).replace(' ', 'T')
}
export function fromLocalDateTimeInput(value: string) {
  return `${value}:00+05:00`
}
export function momentText(iso: string) {
  return new Intl.DateTimeFormat('ru-RU', { timeZone: 'Asia/Yekaterinburg', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(iso))
}
