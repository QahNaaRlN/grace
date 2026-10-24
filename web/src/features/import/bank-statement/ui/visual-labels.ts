import type { components } from '@/shared/api/generated/schema'
type Reading = components['schemas']['VisualReading']
export const rowLabels: Record<keyof components['schemas']['VisualFields'], string> = {
  operationDate: 'Дата/время операции',
  bookingDate: 'Дата проведения',
  accountAmount: 'Сумма по счёту',
  accountCurrency: 'Валюта счёта',
  direction: 'Направление: debit или credit',
  transactionAmount: 'Исходная сумма',
  transactionCurrency: 'Исходная валюта',
  description: 'Полное описание',
}
export const headerLabels: Record<keyof components['schemas']['VisualHeader'], string> = {
  openingBookDate: 'Начало периода',
  closingBookDate: 'Конец периода',
  openingBookBalance: 'Начальный учётный остаток',
  closingBookBalance: 'Конечный учётный остаток',
  availableBalanceObserved: 'Доступный остаток',
  debitTurnover: 'Напечатанный оборот списаний',
  creditTurnover: 'Напечатанный оборот поступлений',
  currency: 'Валюта итогов',
}
export const emptyHeader: Reading['header'] = {
  openingBookDate: null,
  closingBookDate: null,
  openingBookBalance: null,
  closingBookBalance: null,
  availableBalanceObserved: null,
  debitTurnover: null,
  creditTurnover: null,
  currency: null,
}
