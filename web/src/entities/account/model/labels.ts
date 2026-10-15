import type { Account } from '@/shared/api'

export type PaymentCard = NonNullable<Account['paymentInstruments']>[number]

export const accountTypeLabels: Record<Account['type'], string> = {
  debit: 'Дебетовый счёт',
  credit: 'Кредитный счёт',
  installment: 'Рассрочка',
  cash: 'Наличные',
}

export const reconciliationOutcomeLabels = {
  matched: 'Остаток совпал',
  opening_set: 'Начальный остаток установлен',
  adjusted: 'Разница записана корректировкой',
}
