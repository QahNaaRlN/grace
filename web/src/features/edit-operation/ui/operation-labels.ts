export const kindLabels: Record<string, string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  debt_payment: 'Платёж по обязательству',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
}

export const convertible = ['purchase', 'income_received', 'debt_payment']

export const fieldLabels: Record<string, string> = {
  merchant: 'Продавец',
  description: 'Описание',
  occurred: 'Дата',
  kind: 'Вид',
  category: 'Категория',
  amountMinor: 'Сумма',
  account: 'Счёт',
  destinationAccount: 'Счёт зачисления',
  voided: 'Отменена',
}

// 409 linked carries a ready message naming what to unlink first.
export function failure(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}
