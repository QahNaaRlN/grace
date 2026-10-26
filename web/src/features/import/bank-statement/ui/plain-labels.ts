export const plainFields = [
  ['operationDate', 'Дата и время операции'],
  ['bookingDate', 'Дата и время списания'],
  ['transactionAmount', 'Сумма операции со знаком'],
  ['transactionCurrency', 'Валюта операции'],
  ['accountAmount', 'Сумма по счёту со знаком'],
  ['accountCurrency', 'Валюта счёта'],
  ['description', 'Описание'],
  ['cardLast4', 'Карта'],
] as const
export const plainHeaders = [
  ['periodFrom', 'Начало периода'],
  ['periodTo', 'Конец периода'],
  ['availableBalanceDate', 'Дата доступного остатка'],
  ['availableBalanceObserved', 'Доступный остаток'],
  ['creditTurnover', 'Напечатанный итог пополнений'],
  ['debitTurnover', 'Напечатанный итог списаний'],
  ['currency', 'Валюта итогов'],
] as const
