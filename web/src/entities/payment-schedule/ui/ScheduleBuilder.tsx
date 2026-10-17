import { useState } from 'react'
import { Plus } from 'lucide-react'
import { parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { previewSchedule, type RejectedScheduleLine } from '../api/preview-schedule'
import { replaceUnlocked, rowKey, rowsFromPayments, type ScheduleRow } from '../model/schedule-rows'
import { EqualForm, PasteForm, RejectedLines } from './ScheduleForms'
import { useScheduleSummary } from './schedule-summary'
import { ScheduleLine } from './ScheduleLine'

type Mode = 'equal' | 'paste' | 'manual'

const modes: { id: Mode; label: string }[] = [
  { id: 'equal', label: 'Равные платежи' },
  { id: 'paste', label: 'Вставить из таблицы' },
  { id: 'manual', label: 'Вручную' },
]

const collapsedEdge = 3

// SPEC-0037: three ways to fill a schedule that all land in one compact,
// editable table. Dates, validation and the total come from Go previews.
export function ScheduleBuilder({
  rows,
  onChange,
  currency,
  idPrefix,
  initialMode = 'equal',
  withParts = false,
}: {
  rows: ScheduleRow[]
  onChange: (rows: ScheduleRow[]) => void
  currency: string
  idPrefix: string
  initialMode?: Mode
  // SPEC-0053: loan rows may carry principal, interest and other charges.
  withParts?: boolean
}) {
  const [mode, setMode] = useState<Mode>(initialMode)
  const [showParts, setShowParts] = useState(() => withParts && rows.some((row) => row.parts))
  const [expanded, setExpanded] = useState(false)
  const [message, setMessage] = useState('')
  const [rejected, setRejected] = useState<RejectedScheduleLine[]>([])
  const [busy, setBusy] = useState(false)
  const summary = useScheduleSummary(rows, currency)

  async function generate(run: () => ReturnType<typeof previewSchedule>) {
    setBusy(true)
    setMessage('')
    setRejected([])
    try {
      const result = await run()
      if (!result.ok) {
        setMessage(result.message)
        setRejected(result.rejectedLines)
        return
      }
      setRejected(result.preview.rejectedLines)
      onChange(replaceUnlocked(rows, rowsFromPayments(result.preview.payments)))
      setExpanded(false)
    } finally {
      setBusy(false)
    }
  }

  async function appendRow() {
    const last = rows.at(-1)
    const lastAmount = last ? parseMinor(last.amount) : null
    let dueOn = ''
    if (last?.dueOn) {
      const next = await previewSchedule({
        mode: 'equal',
        firstDueOn: last.dueOn,
        count: 2,
        interval: 'monthly',
        amountMinor: lastAmount && lastAmount > 0 ? lastAmount : 1,
      })
      if (next.ok) dueOn = next.preview.payments[1]?.dueOn ?? ''
    }
    onChange([...rows, { key: rowKey(), dueOn, amount: last?.amount ?? '' }])
  }

  function update(key: string, patch: Partial<ScheduleRow>) {
    onChange(rows.map((row) => (row.key === key ? { ...row, ...patch } : row)))
  }

  function fillDown(from: number) {
    const amount = rows[from].amount
    onChange(rows.map((row, index) => (index > from && !row.locked ? { ...row, amount } : row)))
  }

  const collapsed = !expanded && rows.length > collapsedEdge * 2
  const visible = collapsed
    ? [
        ...rows.slice(0, collapsedEdge).map((row, index) => ({ row, index })),
        null,
        ...rows
          .slice(-collapsedEdge)
          .map((row, offset) => ({ row, index: rows.length - collapsedEdge + offset })),
      ]
    : rows.map((row, index) => ({ row, index }))

  return (
    <div className="space-y-4">
      <div role="group" aria-label="Способ заполнения графика" className="flex flex-wrap gap-2">
        {modes.map((item) => (
          <Button
            key={item.id}
            type="button"
            variant={mode === item.id ? 'secondary' : 'ghost'}
            aria-pressed={mode === item.id}
            onClick={() => setMode(item.id)}
          >
            {item.label}
          </Button>
        ))}
      </div>

      {mode === 'equal' && (
        <EqualForm
          idPrefix={idPrefix}
          currency={currency}
          busy={busy}
          onBuild={(request) => void generate(() => previewSchedule(request))}
        />
      )}
      {mode === 'paste' && (
        <PasteForm
          idPrefix={idPrefix}
          busy={busy}
          withParts={withParts}
          onParse={(text, columns) => {
            if (columns.principalColumn) setShowParts(true)
            void generate(() => previewSchedule({ mode: 'paste', text, ...columns }))
          }}
        />
      )}

      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
      {rejected.length > 0 && <RejectedLines lines={rejected} />}

      <section aria-label="График платежей" className="space-y-2">
        {withParts && (
          <label className="flex min-h-11 items-center gap-3 text-sm">
            <Checkbox checked={showParts} onCheckedChange={(v) => setShowParts(v === true)} />
            <span>Указать тело и проценты</span>
          </label>
        )}
        {summary && (
          <p role="status" className="text-sm text-muted-foreground">
            {summary}
          </p>
        )}
        {rows.length > 0 && (
          <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto] items-center gap-x-2 gap-y-2">
            <span className="text-sm font-medium text-muted-foreground">Дата</span>
            <span className="text-sm font-medium text-muted-foreground">Сумма</span>
            {/* sr-only alone is absolutely positioned and takes no grid cell (SPEC-0047). */}
            <span className="col-span-2">
              <span className="sr-only">Действия</span>
            </span>
            {visible.map((item) => {
              if (item === null)
                return (
                  <div
                    key="collapsed"
                    className="col-span-4 flex items-center justify-between gap-3 rounded-xl bg-muted/60 px-3 py-2 text-sm"
                  >
                    <span>… ещё {rows.length - collapsedEdge * 2} платежей</span>
                    <Button type="button" variant="link" onClick={() => setExpanded(true)}>
                      Показать все {rows.length}
                    </Button>
                  </div>
                )
              const { row, index } = item
              const number = index + 1
              const last = index === rows.length - 1
              return (
                <ScheduleLine
                  key={row.key}
                  row={row}
                  number={number}
                  currency={currency}
                  idPrefix={idPrefix}
                  onDate={(dueOn) => update(row.key, { dueOn })}
                  onAmount={(amount) => update(row.key, { amount })}
                  onAmountKey={(event) => {
                    if (event.key === 'Enter' && last) {
                      event.preventDefault()
                      void appendRow()
                    }
                  }}
                  onFillDown={index < rows.length - 1 ? () => fillDown(index) : undefined}
                  onRemove={() => onChange(rows.filter((other) => other.key !== row.key))}
                  onParts={
                    showParts
                      ? (parts) => update(row.key, { parts, partsSource: 'bank' })
                      : undefined
                  }
                />
              )
            })}
          </div>
        )}
        {expanded && rows.length > collapsedEdge * 2 && (
          <Button type="button" variant="link" onClick={() => setExpanded(false)}>
            Свернуть график
          </Button>
        )}
        <Button type="button" variant="outline" onClick={() => void appendRow()}>
          <Plus aria-hidden="true" />
          Добавить строку
        </Button>
      </section>
    </div>
  )
}
