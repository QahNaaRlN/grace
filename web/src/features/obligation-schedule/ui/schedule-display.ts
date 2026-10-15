import type { FinancialEvent } from '@/shared/api'
import { type Tone } from '@/shared/ui/status'

export const statusLabels: Record<string, string> = {
  planned: 'Запланирован',
  overdue: 'Просрочен',
  partially_paid: 'Частично оплачен',
  paid: 'Оплачен',
}

// SPEC-0035: a payment state is a word, an icon and a tone; yellow means «скоро» only (SPEC-0046).
export const statusTones: Record<string, Tone> = {
  planned: 'info',
  overdue: 'danger',
  partially_paid: 'info',
  paid: 'success',
}

export const day = (iso: string) =>
  new Date(`${iso}T00:00:00Z`).toLocaleDateString('ru-RU', { timeZone: 'UTC' })

export function plural(n: number, one: string, few: string, many: string) {
  const d = n % 10,
    h = n % 100
  return d === 1 && h !== 11 ? one : d >= 2 && d <= 4 && (h < 12 || h > 14) ? few : many
}

export const payments = (n: number) => `${n} ${plural(n, 'платёж', 'платежа', 'платежей')}`

export type Linking = { events: FinancialEvent[]; usedEventIDs: Set<string> }
