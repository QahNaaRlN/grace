import { z } from 'zod'

import { parseMinor } from '@/shared/lib/finance-display'

// SPEC-0036: amounts stay text until parseMinor turns them into integer minor units.
const positiveAmount = (message: string) =>
  z
    .string()
    .trim()
    .min(1, message)
    .refine(value => parseMinor(value) !== null, 'Не больше двух знаков после запятой')
    .refine(value => (parseMinor(value) ?? 0) > 0, 'Сумма должна быть больше нуля')

const PaymentSchema = z.object({
  dueOn: z.string().min(1, 'Укажите дату'),
  amount: positiveAmount('Укажите сумму'),
})

export const CreateObligationSchema = z
  .object({
    name: z.string().trim().min(1, 'Укажите название').max(200),
    kind: z.enum(['loan', 'installment', 'other_fixed']),
    currency: z.string().trim().length(3, 'Трёхбуквенный код валюты'),
    total: positiveAmount('Укажите общую сумму'),
    payments: z.array(PaymentSchema).min(1, 'Добавьте хотя бы один платёж').max(600),
  })
  .superRefine((value, context) => {
    const totalMinor = parseMinor(value.total)
    const amounts = value.payments.map(payment => parseMinor(payment.amount))
    if (totalMinor !== null && amounts.every(amount => amount !== null)) {
      const scheduleMinor = amounts.reduce((sum: bigint, amount) => sum + BigInt(amount ?? 0), 0n)
      if (BigInt(totalMinor) !== scheduleMinor) {
        context.addIssue({ code: 'custom', path: ['payments'], message: 'Сумма графика должна совпадать с общей суммой' })
      }
    }
    for (let index = 1; index < value.payments.length; index += 1) {
      if (value.payments[index].dueOn <= value.payments[index - 1].dueOn) {
        context.addIssue({ code: 'custom', path: ['payments', index, 'dueOn'], message: 'Даты должны строго возрастать' })
      }
    }
  })

export type CreateObligationValues = z.infer<typeof CreateObligationSchema>
