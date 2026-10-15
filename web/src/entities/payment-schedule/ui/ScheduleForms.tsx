import { useState } from 'react'
import { parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Disclosure } from '@/shared/ui/disclosure'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { Textarea } from '@/shared/ui/textarea'
import { type RejectedScheduleLine, type SchedulePreviewRequest } from '../api/preview-schedule'

export function EqualForm({
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

export function PasteForm({
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

export function RejectedLines({ lines }: { lines: RejectedScheduleLine[] }) {
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
