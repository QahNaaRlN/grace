import { z } from 'zod'

export const CreateTransactionFormSchema = z.object({
  accountId: z.uuid('Выберите счёт'),
  categoryId: z.uuid('Выберите категорию'),
  amount: z
    .number({ message: 'Укажите сумму' })
    .positive('Сумма должна быть больше нуля')
    .multipleOf(0.01, 'Не больше двух знаков после запятой'),
  merchant: z.string().trim().min(1, 'Укажите получателя').max(160),
  description: z.string().trim().max(500),
})

export type CreateTransactionFormValues = z.infer<typeof CreateTransactionFormSchema>

