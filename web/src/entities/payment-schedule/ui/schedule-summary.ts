import { useEffect, useState } from 'react'
import { dateText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { previewSchedule, type SchedulePreview } from '../api/preview-schedule'
import { rowsForPreview, type ScheduleRow } from '../model/schedule-rows'

// The summary line ("120 платежей · 15 янв. 2027 г. — 15 дек. 2036 г. · итог …")
// comes from a debounced rows preview, so Go computes the total.
export function useScheduleSummary(rows: ScheduleRow[], currency: string) {
  const [summary, setSummary] = useState<{ text: string; key: string } | null>(null)
  const payload = JSON.stringify(rowsForPreview(rows))
  const key = `${currency}|${payload}`
  useEffect(() => {
    if (payload === '[]') return
    let cancelled = false
    const timer = setTimeout(() => {
      void previewSchedule({ mode: 'rows', payments: JSON.parse(payload) }).then((result) => {
        if (!cancelled)
          setSummary({
            text: result.ok ? summaryText(result.preview, currency) : result.message,
            key,
          })
      })
    }, 300)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [payload, currency, key])
  if (rows.length === 0) return ''
  return summary?.key === key ? summary.text : 'Считаем итог…'
}

export function summaryText(preview: SchedulePreview, currency: string) {
  const count = preview.payments.length
  const range =
    preview.firstDueOn && preview.lastDueOn
      ? ` · ${dateText(preview.firstDueOn)} — ${dateText(preview.lastDueOn)}`
      : ''
  const problems = preview.rejectedLines.length
    ? ` · требуют исправления: ${preview.rejectedLines.map((line) => `строка ${line.line} (${line.reason})`).join(', ')}`
    : ''
  const code = currency.trim().toUpperCase()
  const total = /^[A-Z]{3}$/.test(code)
    ? formatMoney(preview.totalMinor, code)
    : formatMoney(preview.totalMinor, 'XXX')
        .replace(/\s*XXX\s*/, ' ')
        .trim()
  return `${count} ${plural(count)}${range} · итог ${total}${problems}`
}

export function plural(count: number) {
  const tail = count % 100
  if (tail >= 11 && tail <= 14) return 'платежей'
  if (count % 10 === 1) return 'платёж'
  if (count % 10 >= 2 && count % 10 <= 4) return 'платежа'
  return 'платежей'
}
