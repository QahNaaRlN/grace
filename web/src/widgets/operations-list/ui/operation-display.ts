import { ArrowDownLeft, ArrowRightLeft, Landmark, Scale, Undo2 } from 'lucide-react'
import type { Account, FinancialEvent } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { calendarDay, clockText } from '@/shared/lib/date'

export const kindLabels: Record<FinancialEvent['kind'], string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
  debt_payment: 'Платёж по обязательству',
  balance_adjustment: 'Корректировка остатка',
}

export const kindIcons = {
  income_received: ArrowDownLeft,
  internal_transfer: ArrowRightLeft,
  refund: Undo2,
  debt_payment: Landmark,
  balance_adjustment: Scale,
}

export const sourceLabels: Record<string, string> = {
  manual: 'Ручной ввод',
  import: 'Импорт таблицы',
  bank_import: 'Импорт выписки',
  receipt: 'Фото чека',
}

export function localDay(event: FinancialEvent) {
  if (event.occurredDate) return event.occurredDate
  if (!event.occurredAt) return ''
  return calendarDay(event.occurredAt)
}

export function dayTitle(day: string) {
  const text = new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'UTC',
    day: 'numeric',
    month: 'long',
    weekday: 'long',
  }).format(new Date(`${day}T00:00:00Z`))
  // "понедельник, 13 октября" → "13 октября, понедельник"
  const [weekday, rest] = text.split(', ')
  return rest ? `${rest}, ${weekday}` : text
}

export function timeText(event: FinancialEvent) {
  if (!event.occurredAt) return 'время неизвестно'
  return clockText(event.occurredAt)
}

export function accountName(
  accounts: Account[],
  id: string | undefined,
  fallback: string | undefined,
) {
  const account = accounts.find((a) => a.id === id)
  return account ? accountLabel(account) : (fallback ?? 'Счёт не указан')
}

export function accountText(event: FinancialEvent, accounts: Account[]) {
  if (event.kind === 'internal_transfer') {
    const from = event.postings.find((p) => p.direction === 'debit')
    const to = event.postings.find((p) => p.direction === 'credit')
    return `${accountName(accounts, from?.accountId, from?.accountName)} → ${accountName(accounts, to?.accountId, to?.accountName)}`
  }
  const posting = event.postings[0]
  return accountName(accounts, posting?.accountId, posting?.accountName)
}

export function direction(event: FinancialEvent) {
  if (event.kind === 'internal_transfer') return undefined
  return event.postings[0]?.direction === 'credit' ? ('credit' as const) : ('debit' as const)
}
