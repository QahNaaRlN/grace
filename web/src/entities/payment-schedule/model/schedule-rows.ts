import { minorToInput, parseMinor } from '@/shared/lib/money'

// SPEC-0053: principal, interest and other charges of a loan row as typed.
export type RowParts = { principal: string; interest: string; other: string }

// One editable line of a schedule. `locked` rows (already paid in a loan
// draft) are shown but never replaced by a generator; their date and amount
// stay, their parts may still be added or fixed.
export type ScheduleRow = {
  key: string
  dueOn: string
  amount: string
  locked?: boolean
  sourceId?: string
  paidMinor?: number
  parts?: RowParts
  partsSource?: 'bank' | 'calculated'
}

type PaymentWithParts = {
  dueOn: string
  amountMinor: number
  principalMinor?: number | null
  interestMinor?: number | null
  otherMinor?: number | null
  partsSource?: 'bank' | 'calculated' | null
}

let sequence = 0
export function rowKey() {
  sequence += 1
  return `row-${sequence}`
}

export function partsFromPayment(
  payment: PaymentWithParts,
): Pick<ScheduleRow, 'parts' | 'partsSource'> {
  if (payment.principalMinor == null || payment.interestMinor == null || payment.otherMinor == null)
    return {}
  return {
    parts: {
      principal: minorToInput(payment.principalMinor),
      interest: minorToInput(payment.interestMinor),
      other: minorToInput(payment.otherMinor),
    },
    partsSource: payment.partsSource ?? 'bank',
  }
}

export function rowsFromPayments(payments: PaymentWithParts[]): ScheduleRow[] {
  return payments.map((payment) => ({
    key: rowKey(),
    dueOn: payment.dueOn,
    amount: minorToInput(payment.amountMinor),
    ...partsFromPayment(payment),
  }))
}

// The parts of a row for the API: none when they are empty; Go checks that they
// add up to the amount.
export function partsPayload(row: ScheduleRow) {
  const parts = row.parts
  if (!parts || (!parts.principal.trim() && !parts.interest.trim() && !parts.other.trim()))
    return {}
  const value = (text: string) => (text.trim() ? parseMinor(text) : 0)
  const principalMinor = value(parts.principal)
  const interestMinor = value(parts.interest)
  const otherMinor = value(parts.other)
  if (
    principalMinor === null ||
    interestMinor === null ||
    otherMinor === null ||
    principalMinor < 0 ||
    interestMinor < 0 ||
    otherMinor < 0
  )
    throw new Error('Проверьте тело, проценты и прочее: не меньше нуля, два знака после запятой.')
  return { principalMinor, interestMinor, otherMinor }
}

// Rows as sent to the preview: an unparseable amount goes as 0 and unparseable
// parts as -1 so Go reports them. Parts go only for loan schedules.
export function rowsForPreview(rows: ScheduleRow[], withParts = false) {
  return rows.map((row) => {
    const payment = { dueOn: row.dueOn, amountMinor: parseMinor(row.amount) ?? 0 }
    if (!withParts) return payment
    try {
      return { ...payment, ...partsPayload(row) }
    } catch {
      return { ...payment, principalMinor: -1, interestMinor: 0, otherMinor: 0 }
    }
  })
}

// Generated payments replace everything that is not locked.
export function replaceUnlocked(rows: ScheduleRow[], generated: ScheduleRow[]) {
  return [...rows.filter((row) => row.locked), ...generated]
}
