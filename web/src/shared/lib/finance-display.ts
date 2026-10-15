import type { Account } from '@/shared/api/types'
export function accountLabel(account: Account) {
  const card = account.paymentInstruments?.find((item) => !item.disabledAt)
  return card ? `${card.bank} *${card.last4}` : account.name
}
// SPEC-0046: Go decides a payment is «скоро» (0–3 days); this only words it.
export function dueSoonText(days: number) {
  if (days <= 0) return 'Сегодня'
  if (days === 1) return 'Завтра'
  return `Через ${days} дня`
}
