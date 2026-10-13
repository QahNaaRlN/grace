import { z } from 'zod'

// SPEC-0037: the schedule (and its total) is built and validated by Go through
// the schedule preview; the form validates only what the person types here.
export const CreateObligationSchema = z.object({
  name: z.string().trim().min(1, 'Укажите название').max(200),
  kind: z.enum(['loan', 'installment', 'other_fixed']),
  currency: z.string().trim().regex(/^[A-Za-z]{3}$/, 'Трёхбуквенный код валюты'),
})

export type CreateObligationValues = z.infer<typeof CreateObligationSchema>
