// Import decisions and operation kinds as the review screens name them.
export const importLabels: { [key: string]: string } = {
  unresolved: 'Не проверено',
  create: 'Новая операция',
  link_existing: 'Связать с операцией',
  duplicate_of: 'Дубль другой строки',
  exclude: 'Исключить',
  published: 'Подтверждено',
  purchase: 'Покупка',
  income_received: 'Доход',
  debt_payment: 'Погашение',
  internal_transfer: 'Перевод между счетами',
  refund: 'Возврат',
}

export function importErrorText(error: unknown) {
  return error instanceof Error
    ? error.message
    : 'Не удалось выполнить действие. Обновите страницу и повторите.'
}

export function importSourceLabel(
  source: import('@/shared/api/generated/schema').components['schemas']['ImportSourceRecord'],
) {
  return source.pdf ? `PDF, строка ${source.pdf.ordinal}` : `${source.sheet}, строка ${source.row}`
}
