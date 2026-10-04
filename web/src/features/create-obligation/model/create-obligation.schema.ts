import { z } from 'zod'

const PaymentSchema = z.object({
  dueOn: z.string().min(1, 'Укажите дату'),
  amount: z
    .number({ message: 'Укажите сумму' })
    .positive('Сумма должна быть больше нуля')
    .multipleOf(0.01, 'Не больше двух знаков после запятой'),
})

export const CreateObligationSchema = z
  .object({
    name: z.string().trim().min(1, 'Укажите название').max(200),
    kind: z.enum(['loan', 'installment', 'other_fixed']),
    currency: z.string().trim().length(3, 'Трёхбуквенный код валюты'),
    total: z
      .number({ message: 'Укажите общую сумму' })
      .positive('Сумма должна быть больше нуля')
      .multipleOf(0.01, 'Не больше двух знаков после запятой'),
    payments: z.array(PaymentSchema).min(1, 'Добавьте хотя бы один платёж').max(600),
  })
  .superRefine((value, context) => {
    const totalMinor = Math.round(value.total * 100)
    const scheduleMinor = value.payments.reduce((sum, payment) => sum + Math.round(payment.amount * 100), 0)
    if (totalMinor !== scheduleMinor) {
      context.addIssue({ code: 'custom', path: ['payments'], message: 'Сумма графика должна совпадать с общей суммой' })
    }
    for (let index = 1; index < value.payments.length; index += 1) {
      if (value.payments[index].dueOn <= value.payments[index - 1].dueOn) {
        context.addIssue({ code: 'custom', path: ['payments', index, 'dueOn'], message: 'Даты должны строго возрастать' })
      }
    }
  })

export type CreateObligationValues = z.infer<typeof CreateObligationSchema>
