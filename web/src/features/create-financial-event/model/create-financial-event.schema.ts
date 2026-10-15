import { z } from 'zod'

export const CreateFinancialEventSchema = z
  .object({
    kind: z.enum(['purchase', 'income', 'transfer', 'debt_payment']),
    accountId: z.string().min(1, 'Выберите счёт'),
    destinationAccountId: z.string(),
    categoryId: z.string(),
    amount: z
      .number({ message: 'Укажите сумму' })
      .positive('Сумма должна быть больше нуля')
      .multipleOf(0.01, 'Не больше двух знаков после запятой'),
    merchant: z.string().trim().max(160),
    description: z.string().trim().max(500),
  })
  .superRefine((value, context) => {
    if (value.kind === 'purchase' && !value.categoryId) {
      context.addIssue({ code: 'custom', path: ['categoryId'], message: 'Выберите категорию' })
    }
    if (value.kind === 'purchase' && !value.merchant && !value.description) {
      context.addIssue({
        code: 'custom',
        path: ['merchant'],
        message: 'Укажите получателя или комментарий',
      })
    }
    if (value.kind === 'transfer' && !value.destinationAccountId) {
      context.addIssue({
        code: 'custom',
        path: ['destinationAccountId'],
        message: 'Выберите счёт назначения',
      })
    }
    if (value.kind === 'transfer' && value.accountId === value.destinationAccountId) {
      context.addIssue({
        code: 'custom',
        path: ['destinationAccountId'],
        message: 'Счета должны различаться',
      })
    }
  })

export type CreateFinancialEventValues = z.infer<typeof CreateFinancialEventSchema>
