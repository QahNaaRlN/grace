// Test double for POST /api/v1/obligations/schedule-preview. The real rules
// live in Go (internal/obligations/schedulepreview_test.go); this fake only
// mirrors the response shapes the UI depends on.
type Payment = { dueOn: string; amountMinor: number }
type Body = {
  mode: 'equal' | 'paste' | 'rows'
  firstDueOn?: string
  count?: number
  amountMinor?: number
  firstAmountMinor?: number
  lastAmountMinor?: number
  text?: string
  payments?: Payment[]
}

function addMonths(date: string, months: number) {
  const [year, month, day] = date.split('-').map(Number)
  const first = new Date(Date.UTC(year, month - 1 + months, 1))
  const last = new Date(Date.UTC(first.getUTCFullYear(), first.getUTCMonth() + 1, 0)).getUTCDate()
  first.setUTCDate(Math.min(day, last))
  return first.toISOString().slice(0, 10)
}

function summary(payments: Payment[], rejectedLines: { line: number; text: string; reason: string }[] = []) {
  return {
    data: {
      payments,
      totalMinor: payments.reduce((sum, payment) => sum + payment.amountMinor, 0),
      firstDueOn: payments[0]?.dueOn,
      lastDueOn: payments.at(-1)?.dueOn,
      rejectedLines,
    },
  }
}

export function fakeSchedulePreview(body: Body) {
  if (body.mode === 'equal') {
    const count = body.count ?? 0
    return summary(Array.from({ length: count }, (_, index) => ({
      dueOn: addMonths(body.firstDueOn ?? '', index),
      amountMinor: index === 0 && body.firstAmountMinor ? body.firstAmountMinor
        : index === count - 1 && body.lastAmountMinor ? body.lastAmountMinor : body.amountMinor ?? 0,
    })))
  }
  if (body.mode === 'paste') {
    const payments: Payment[] = []
    const rejected: { line: number; text: string; reason: string }[] = []
    ;(body.text ?? '').split('\n').forEach((line, index) => {
      const match = /(\d{2})\.(\d{2})\.(\d{4})\s+([\d\s]+),(\d{2})/.exec(line)
      if (!match) rejected.push({ line: index + 1, text: line, reason: 'нет даты платежа' })
      else payments.push({ dueOn: `${match[3]}-${match[2]}-${match[1]}`, amountMinor: Number(match[4].replace(/\s/g, '') + match[5]) })
    })
    if (payments.length === 0) return { error: { code: 'no_payments', message: 'Ни одной строки графика не распознано.', rejectedLines: rejected } }
    return summary(payments, rejected)
  }
  const rows = body.payments ?? []
  const rejected = rows.flatMap((row, index) => (row.amountMinor > 0 && row.dueOn ? [] : [{ line: index + 1, text: '', reason: row.dueOn ? 'сумма должна быть больше нуля' : 'некорректная дата' }]))
  return summary(rows.filter(row => row.amountMinor > 0 && row.dueOn), rejected)
}
