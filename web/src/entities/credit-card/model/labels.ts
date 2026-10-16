import type { components } from '@/shared/api/generated/schema'

export type CreditCard = components['schemas']['CreditCard']
export type CreditCardTerms = components['schemas']['CreditCardTerms']
export type CreditCardStatus = components['schemas']['CreditCardStatus']
export type GraceModel = CreditCardTerms['graceModel']

export const graceModelLabels: Record<GraceModel, string> = {
  statement: 'От выписки: расчётный период и дни на оплату',
  from_first_purchase: 'С первой покупки: N дней',
  none: 'Льготного периода нет',
  unknown: 'Не знаю',
}

/** What Grace cannot compute yet and why; shown instead of zeros. */
export const unknownLabels: Record<CreditCardStatus['unknown'][number], string> = {
  terms: 'Условия карты не заданы.',
  limit: 'Лимит неизвестен.',
  grace_model: 'Правило льготного периода неизвестно.',
  minimum_rule: 'Правило минимального платежа неизвестно.',
  interest: 'Начисленные проценты банк добавит к минимальному платежу; Grace их не считает.',
  history: 'Выписка закрылась до начала учёта карты: точную сумму покажет банк.',
  statement_day: 'День выписки неизвестен.',
  debt: 'Долг неизвестен: сверьте остаток счёта или введите данные из банка.',
}
