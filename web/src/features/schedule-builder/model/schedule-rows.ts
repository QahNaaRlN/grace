import { minorToInput, parseMinor } from '@/shared/lib/finance-display'

// One editable line of a schedule. `locked` rows (already paid in a loan
// draft) are shown but never replaced by a generator or edited.
export type ScheduleRow = {
  key: string
  dueOn: string
  amount: string
  locked?: boolean
  sourceId?: string
  paidMinor?: number
}

let sequence = 0
export function rowKey() {
  sequence += 1
  return `row-${sequence}`
}

export function rowsFromPayments(payments: { dueOn: string; amountMinor: number }[]): ScheduleRow[] {
  return payments.map(payment => ({ key: rowKey(), dueOn: payment.dueOn, amount: minorToInput(payment.amountMinor) }))
}

// Rows as sent to the preview: an unparseable amount goes as 0 so Go reports it.
export function rowsForPreview(rows: ScheduleRow[]) {
  return rows.map(row => ({ dueOn: row.dueOn, amountMinor: parseMinor(row.amount) ?? 0 }))
}

// Generated payments replace everything that is not locked.
export function replaceUnlocked(rows: ScheduleRow[], generated: ScheduleRow[]) {
  return [...rows.filter(row => row.locked), ...generated]
}
