import { z } from 'zod'

const money = z
  .number({ message: 'Укажите сумму' })
  .positive('Сумма должна быть больше нуля')
  .multipleOf(0.01, 'Не больше двух знаков после запятой')

export const CreateReceiptSchema = z.object({
  seller: z.string().trim().min(1, 'Укажите продавца').max(200),
  issuedAt: z.string().min(1, 'Укажите дату чека'),
  fiscalDocumentNumber: z.string().trim().max(160),
  items: z.array(z.object({
    name: z.string().trim().min(1, 'Укажите название').max(300),
    quantity: z.number({ message: 'Укажите количество' }).positive().multipleOf(0.001),
    unitPrice: money,
    lineTotal: money,
  })).min(1),
})

export type CreateReceiptValues = z.infer<typeof CreateReceiptSchema>
