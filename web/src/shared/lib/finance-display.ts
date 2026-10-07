import type { Account, FinancialEvent } from '@/shared/api/types'
export function accountLabel(account: Account) {
  const card = account.paymentInstruments?.[0]
  return card ? `${card.name} · ${card.bank} · •• ${card.last4}` : account.name
}
export function formatMoney(valueMinor: number, currency: string) {
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
export function parseMinor(text: string) {
  const match = /^(-?)(\d+)(?:[.,](\d{1,2}))?$/.exec(text.trim())
  if (!match) return null
  const value =
    BigInt(match[2]) * 100n + BigInt((match[3] ?? '').padEnd(2, '0'))
  const signed = match[1] ? -value : value
  return signed > BigInt(Number.MAX_SAFE_INTEGER) ||
    signed < BigInt(Number.MIN_SAFE_INTEGER)
    ? null
    : Number(signed)
}
