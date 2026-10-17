import { parseMinor } from '@/shared/lib/money'
import type { components } from '@/shared/api/generated/schema'
export type RowInput = { dueOn: string; amount: string; locked?: boolean }
export function parseRows(rows: RowInput[]): components['schemas']['InstallmentRow'][] {
  return rows.map((row) => {
    const amountMinor = parseMinor(row.amount)
    if (amountMinor === null) throw new Error('Введите суммы всех строк графика.')
    return { dueOn: row.dueOn, amountMinor }
  })
}
