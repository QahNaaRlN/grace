import type { components } from '@/shared/api/generated/schema'
type Proposal = components['schemas']['StatementProposal']
type Row = components['schemas']['StatementRowMapping']

export const emptyRow = (): Row => ({
  operationDate: [],
  bookingDate: [],
  accountAmount: [],
  accountCurrency: [],
  transactionAmount: [],
  transactionCurrency: [],
  description: [],
  direction: '',
  directionEvidence: [],
})
export const rowFields = [
  ['operationDate', 'Дата и время операции'],
  ['bookingDate', 'Дата и время списания'],
  ['accountAmount', 'Сумма по счёту'],
  ['accountCurrency', 'Валюта счёта'],
  ['transactionAmount', 'Сумма в валюте операции'],
  ['transactionCurrency', 'Валюта операции'],
  ['description', 'Описание'],
  ['directionEvidence', 'Заголовок направления для суммы без знака'],
] as const
export const headerFields = [
  ['openingBookBalance', 'Начальный бухгалтерский остаток'],
  ['closingBookBalance', 'Конечный бухгалтерский остаток'],
  ['availableBalanceObserved', 'Доступный остаток банка'],
  ['debitTurnover', 'Оборот списаний'],
  ['creditTurnover', 'Оборот зачислений'],
] as const
export const initial = (): Proposal => ({
  rows: [emptyRow()],
  header: {
    openingBookDate: [],
    closingBookDate: [],
    openingBookBalance: [],
    closingBookBalance: [],
    availableBalanceObserved: [],
    debitTurnover: [],
    creditTurnover: [],
    currency: [],
    labels: {},
  },
  service: [],
})
