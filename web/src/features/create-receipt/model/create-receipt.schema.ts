import { z } from 'zod'

import { parseMilli, parseMinor } from '@/shared/lib/finance-display'

// SPEC-0036: amounts and quantities stay text until parsed into integers.
const money = z
  .string()
  .trim()
  .min(1, 'Укажите сумму')
  .refine((value) => parseMinor(value) !== null, 'Не больше двух знаков после запятой')
  .refine((value) => (parseMinor(value) ?? 0) > 0, 'Сумма должна быть больше нуля')

export const CreateReceiptSchema = z.object({
  seller: z.string().trim().min(1, 'Укажите продавца').max(200),
  issuedAt: z.string().min(1, 'Укажите дату чека'),
  fiscalDocumentNumber: z.string().trim().max(160),
  items: z
    .array(
      z.object({
        name: z.string().trim().min(1, 'Укажите название').max(300),
        quantity: z
          .string()
          .trim()
          .min(1, 'Укажите количество')
          .refine(
            (value) => (parseMilli(value) ?? 0) > 0,
            'Количество больше нуля, до трёх знаков после запятой',
          ),
        unitPrice: money,
        lineTotal: money,
      }),
    )
    .min(1),
})

export type CreateReceiptValues = z.infer<typeof CreateReceiptSchema>
