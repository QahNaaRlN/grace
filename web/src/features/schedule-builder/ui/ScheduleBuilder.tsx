import { useEffect, useState, type KeyboardEvent } from 'react'
import { ArrowDownToLine, Plus, Trash2 } from 'lucide-react'

import { dateText } from '@/shared/lib/date'
import { formatMoney, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Disclosure } from '@/shared/ui/disclosure'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { Textarea } from '@/shared/ui/textarea'
import {
  previewSchedule,
  type RejectedScheduleLine,
  type SchedulePreview,
  type SchedulePreviewRequest,
} from '../api/preview-schedule'
import {
  replaceUnlocked,
  rowKey,
  rowsForPreview,
  rowsFromPayments,
  type ScheduleRow,
} from '../model/schedule-rows'

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
}: {
  rows: ScheduleRow[]
  onChange: (rows: ScheduleRow[]) => void
  currency: string
  idPrefix: string
  initialMode?: Mode
}) {
  const [mode, setMode] = useState<Mode>(initialMode)
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
          onParse={(text, amountColumn) =>
            void generate(() => previewSchedule({ mode: 'paste', text, amountColumn }))
          }
        />
      )}

      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
      {rejected.length > 0 && <RejectedLines lines={rejected} />}

      <section aria-label="График платежей" className="space-y-2">
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

function ScheduleLine({
  row,
  number,
  currency,
  idPrefix,
  onDate,
  onAmount,
  onAmountKey,
  onFillDown,
  onRemove,
}: {
  row: ScheduleRow
  number: number
  currency: string
  idPrefix: string
  onDate: (value: string) => void
  onAmount: (value: string) => void
  onAmountKey: (event: KeyboardEvent<HTMLInputElement>) => void
  onFillDown?: () => void
  onRemove: () => void
}) {
  const paid = row.locked ? ' · есть оплата' : ''
  return (
    <>
      <Input
        id={`${idPrefix}-date-${number}`}
        type="date"
        aria-label={`Дата платежа ${number}${paid}`}
        value={row.dueOn}
        disabled={row.locked}
        onChange={(event) => onDate(event.target.value)}
      />
      <MoneyInput
        id={`${idPrefix}-amount-${number}`}
        aria-label={`Сумма платежа ${number}`}
        currency={currency}
        value={row.amount}
        disabled={row.locked}
        onChange={(event) => onAmount(event.target.value)}
        onKeyDown={onAmountKey}
      />
      {onFillDown && !row.locked ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={`Сумму платежа ${number} — во все следующие`}
          title="Заполнить сумму вниз"
          onClick={onFillDown}
        >
          <ArrowDownToLine aria-hidden="true" />
        </Button>
      ) : (
        <span />
      )}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={`Удалить платёж ${number}`}
        disabled={row.locked}
        onClick={onRemove}
      >
        <Trash2 aria-hidden="true" />
      </Button>
    </>
  )
}

function EqualForm({
  idPrefix,
  currency,
  busy,
  onBuild,
}: {
  idPrefix: string
  currency: string
  busy: boolean
  onBuild: (request: Extract<SchedulePreviewRequest, { mode: 'equal' | 'period' }>) => void
}) {
  // SPEC-0045: by count (first date + N) or by term (day of month, from–to month).
  const [basis, setBasis] = useState<'count' | 'period'>('count')
  const [day, setDay] = useState('15')
  const [fromMonth, setFromMonth] = useState('')
  const [toMonth, setToMonth] = useState('')
  const [firstDueOn, setFirstDueOn] = useState('')
  const [count, setCount] = useState('')
  const [amount, setAmount] = useState('')
  const [firstAmount, setFirstAmount] = useState('')
  const [lastAmount, setLastAmount] = useState('')
  const [error, setError] = useState('')
  function build() {
    const amountMinor = parseMinor(amount)
    const total = Number(count)
    const optional = (text: string) => (text.trim() ? parseMinor(text) : undefined)
    const first = optional(firstAmount)
    const final = optional(lastAmount)
    if (basis === 'count' && (!firstDueOn || !Number.isInteger(total) || total < 1 || total > 600))
      return setError('Укажите дату первого платежа и количество от 1 до 600.')
    if (basis === 'period' && (!fromMonth || !toMonth))
      return setError('Укажите месяц начала и месяц окончания.')
    if (amountMinor === null || amountMinor <= 0 || first === null || final === null)
      return setError('Проверьте суммы: больше нуля, не больше двух знаков после запятой.')
    setError('')
    const amounts = {
      amountMinor,
      ...(first !== undefined ? { firstAmountMinor: first } : {}),
      ...(final !== undefined ? { lastAmountMinor: final } : {}),
    }
    onBuild(
      basis === 'count'
        ? { mode: 'equal', firstDueOn, count: total, interval: 'monthly', ...amounts }
        : { mode: 'period', dayOfMonth: Number(day), fromMonth, toMonth, ...amounts },
    )
  }
  return (
    <div className="space-y-4 rounded-xl border p-4">
      <div role="radiogroup" aria-label="Как задать график" className="flex flex-wrap gap-2">
        {(
          [
            ['count', 'По количеству'],
            ['period', 'По сроку договора'],
          ] as const
        ).map(([id, label]) => (
          <Button
            key={id}
            type="button"
            role="radio"
            aria-checked={basis === id}
            variant={basis === id ? 'secondary' : 'ghost'}
            onClick={() => setBasis(id)}
          >
            {label}
          </Button>
        ))}
      </div>
      <div
        className={
          basis === 'count'
            ? 'grid gap-4 sm:grid-cols-3'
            : 'grid gap-4 sm:grid-cols-2 lg:grid-cols-4'
        }
      >
        {basis === 'count' ? (
          <>
            <div className="grid gap-2">
              <Label htmlFor={`${idPrefix}-first`}>Первый платёж</Label>
              <Input
                id={`${idPrefix}-first`}
                type="date"
                value={firstDueOn}
                onChange={(e) => setFirstDueOn(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor={`${idPrefix}-count`}>Количество, ежемесячно</Label>
              <Input
                id={`${idPrefix}-count`}
                inputMode="numeric"
                className="tabular-nums"
                value={count}
                onChange={(e) => setCount(e.target.value.replace(/\D/g, ''))}
                placeholder="120"
              />
            </div>
          </>
        ) : (
          <>
            <div className="grid gap-2">
              <Label htmlFor={`${idPrefix}-day`}>Число месяца</Label>
              <ChoiceSelect id={`${idPrefix}-day`} value={day} onValueChange={setDay}>
                {Array.from({ length: 30 }, (_, i) => (
                  <option key={i + 1} value={String(i + 1)}>
                    {i + 1}
                  </option>
                ))}
                <option value="31">Последний день месяца</option>
              </ChoiceSelect>
            </div>
            <div className="grid gap-2">
              <Label htmlFor={`${idPrefix}-from`}>С (месяц и год)</Label>
              <Input
                id={`${idPrefix}-from`}
                type="month"
                value={fromMonth}
                onChange={(e) => setFromMonth(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor={`${idPrefix}-to`}>По (включительно)</Label>
              <Input
                id={`${idPrefix}-to`}
                type="month"
                value={toMonth}
                onChange={(e) => setToMonth(e.target.value)}
              />
            </div>
          </>
        )}
        <div className="grid gap-2">
          <Label htmlFor={`${idPrefix}-amount`}>Сумма платежа</Label>
          <MoneyInput
            id={`${idPrefix}-amount`}
            currency={currency}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </div>
      </div>
      <Disclosure title="Первый или последний платёж отличается" className="rounded-xl bg-muted/40">
        <div className="grid gap-4 px-3 pb-3 sm:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor={`${idPrefix}-first-amount`}>Сумма первого платежа</Label>
            <MoneyInput
              id={`${idPrefix}-first-amount`}
              currency={currency}
              value={firstAmount}
              onChange={(e) => setFirstAmount(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor={`${idPrefix}-last-amount`}>Сумма последнего платежа</Label>
            <MoneyInput
              id={`${idPrefix}-last-amount`}
              currency={currency}
              value={lastAmount}
              onChange={(e) => setLastAmount(e.target.value)}
            />
          </div>
        </div>
      </Disclosure>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <Button type="button" variant="outline" disabled={busy} onClick={build}>
        {busy ? 'Строим…' : 'Построить график'}
      </Button>
    </div>
  )
}

function PasteForm({
  idPrefix,
  busy,
  onParse,
}: {
  idPrefix: string
  busy: boolean
  onParse: (text: string, amountColumn: number) => void
}) {
  const [text, setText] = useState('')
  const [column, setColumn] = useState('1')
  return (
    <div className="space-y-4 rounded-xl border p-4">
      <div className="grid gap-2">
        <Label htmlFor={`${idPrefix}-paste`}>Строки графика из банка</Label>
        <Textarea
          id={`${idPrefix}-paste`}
          value={text}
          onChange={(e) => setText(e.target.value)}
          aria-describedby={`${idPrefix}-paste-hint`}
          placeholder={
            '15.01.2027\t25 000,00\t18 000,00\t7 000,00\n15.02.2027\t25 000,00\t18 100,00\t6 900,00'
          }
        />
        <p id={`${idPrefix}-paste-hint`} className="text-sm text-muted-foreground">
          Скопируйте таблицу из выписки, Excel или PDF: в каждой строке дата и сумма. Заголовок и
          «Итого» будут пропущены с пояснением.
        </p>
      </div>
      <div className="grid gap-2 sm:max-w-xs">
        <Label htmlFor={`${idPrefix}-column`}>Сумма платежа — колонка после даты</Label>
        <ChoiceSelect id={`${idPrefix}-column`} value={column} onValueChange={setColumn}>
          {[1, 2, 3, 4, 5, 6].map((n) => (
            <option key={n} value={String(n)}>
              {n === 1 ? 'Первая (обычно «Сумма платежа»)' : `${n}-я`}
            </option>
          ))}
        </ChoiceSelect>
      </div>
      <Button
        type="button"
        variant="outline"
        disabled={busy || !text.trim()}
        onClick={() => onParse(text, Number(column))}
      >
        {busy ? 'Распознаём…' : 'Распознать график'}
      </Button>
    </div>
  )
}

function RejectedLines({ lines }: { lines: RejectedScheduleLine[] }) {
  const shown = lines.slice(0, 20)
  return (
    <div className="rounded-xl border border-dashed border-input p-3 text-sm">
      <p className="font-medium">Пропущено строк: {lines.length}</p>
      <ul className="mt-2 space-y-1">
        {shown.map((line) => (
          <li key={`${line.line}-${line.reason}`}>
            <span className="tabular-nums">Строка {line.line}</span> — {line.reason}
            {line.text && <span className="text-muted-foreground">: {line.text}</span>}
          </li>
        ))}
      </ul>
      {lines.length > shown.length && (
        <p className="mt-2 text-muted-foreground">и ещё {lines.length - shown.length}</p>
      )}
    </div>
  )
}

// The summary line ("120 платежей · 15 янв. 2027 г. — 15 дек. 2036 г. · итог …")
// comes from a debounced rows preview, so Go computes the total.
function useScheduleSummary(rows: ScheduleRow[], currency: string) {
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

function summaryText(preview: SchedulePreview, currency: string) {
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

function plural(count: number) {
  const tail = count % 100
  if (tail >= 11 && tail <= 14) return 'платежей'
  if (count % 10 === 1) return 'платёж'
  if (count % 10 >= 2 && count % 10 <= 4) return 'платежа'
  return 'платежей'
}
