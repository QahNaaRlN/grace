import { useState } from 'react'
import { toast } from 'sonner'
import type { FinancialEvent } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'
import { dateText } from '@/shared/lib/date'
import { formatMoney, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'
import { Notice } from '@/shared/ui/status'
import { useCreateLoanPrepayment, useCreateLoanSnapshot } from '../api/loan-mutations'

type Debt = components['schemas']['LoanDebtStatus']
type Totals = components['schemas']['LoanTotals']

// Unknown values are words, never zero.
const money = (value: number | null | undefined, currency: string) =>
  value === null || value === undefined ? 'неизвестно' : formatMoney(value, currency)

const basisText: Record<Debt['basis'], string> = {
  snapshot: 'от данных банка',
  issued: 'от суммы выдачи',
  unknown: '',
}

function TotalsList({
  title,
  totals,
  currency,
}: {
  title: string
  totals: Totals
  currency: string
}) {
  const rows: [string, number][] = [
    ['Основной долг', totals.principalMinor],
    ['Проценты', totals.interestMinor],
    ['Прочее', totals.otherMinor],
    ['Не распределено', totals.unknownMinor],
  ]
  return (
    <div className="space-y-1">
      <p className="text-sm font-medium">{title}</p>
      <dl className="grid grid-cols-2 gap-x-3 gap-y-1 text-sm">
        {rows
          .filter(([label, value]) => label !== 'Не распределено' || value > 0)
          .map(([label, value]) => (
            <div key={label} className="contents">
              <dt className="text-muted-foreground">{label}</dt>
              <dd className="text-right tabular-nums">{formatMoney(value, currency)}</dd>
            </div>
          ))}
      </dl>
    </div>
  )
}

// SPEC-0053: the principal left by Grace and by the bank, what was paid and what
// the schedule still expects. Every figure comes from Go.
export function LoanDebt({
  obligationId,
  currency,
  debt,
  candidates,
}: {
  obligationId: string
  currency: string
  debt: Debt
  candidates: FinancialEvent[]
}) {
  const [dialog, setDialog] = useState<'snapshot' | 'prepayment' | null>(null)
  const snapshot = debt.latestSnapshot
  return (
    <section aria-label="Долг по кредиту" className="space-y-4 rounded-xl border p-4">
      <div className="space-y-1">
        <p className="text-sm text-muted-foreground">Остаток основного долга по данным Grace</p>
        <p className="text-xl font-semibold tabular-nums">
          {money(debt.principalOutstandingMinor, currency)}
        </p>
        {debt.principalOutstandingMinor !== null ? (
          <p className="text-xs text-muted-foreground">
            {basisText[debt.basis]}
            {debt.basisOn ? ` на ${dateText(debt.basisOn)}` : ''}, минус тело платежей после неё
          </p>
        ) : debt.basis === 'unknown' ? (
          <p className="text-xs text-muted-foreground">
            Укажите сумму выдачи в сведениях договора или введите данные из банка.
          </p>
        ) : (
          <p className="text-xs text-muted-foreground">
            Не распределено платежей: {debt.unallocatedPayments}. Укажите для них тело и проценты.
          </p>
        )}
      </div>
      {snapshot && (
        <div className="space-y-1 text-sm">
          <p>
            Банк на {dateText(snapshot.takenOn)}:{' '}
            <span className="font-medium tabular-nums">
              {formatMoney(snapshot.principalMinor, currency)}
            </span>
            {snapshot.payoffMinor !== null &&
              ` · полное погашение ${formatMoney(snapshot.payoffMinor, currency)}`}
          </p>
          {debt.discrepancyMinor !== null && debt.discrepancyMinor !== 0 && (
            <p className="text-muted-foreground">
              Grace на эту дату: {money(debt.gracePrincipalOnSnapshotMinor, currency)} · расхождение{' '}
              {formatMoney(debt.discrepancyMinor, currency)}. Grace не выбирает, какая сумма верна.
            </p>
          )}
        </div>
      )}
      {debt.scheduleOutdated && (
        <Notice tone="warning" title="График устарел">
          Основной долг в будущих платежах не совпадает с остатком. Скопируйте новый график из банка
          в разделе «Копии графика».
        </Notice>
      )}
      <div className="grid gap-4 sm:grid-cols-2">
        <TotalsList title="Уплачено" totals={debt.paid} currency={currency} />
        <TotalsList title="Предстоит по графику" totals={debt.upcoming} currency={currency} />
      </div>
      {debt.prepayments.length > 0 && (
        <div className="space-y-1 text-sm">
          <p className="font-medium">Досрочные погашения</p>
          <ul className="space-y-1">
            {debt.prepayments.map((p) => (
              <li key={p.id} className="flex justify-between gap-3">
                <span>{dateText(p.paidOn)}</span>
                <span className="tabular-nums">
                  {formatMoney(p.amountMinor, currency)} · тело{' '}
                  {formatMoney(p.principalMinor, currency)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" onClick={() => setDialog('snapshot')}>
          Ввести данные из банка
        </Button>
        <Button
          variant="outline"
          disabled={candidates.length === 0}
          onClick={() => setDialog('prepayment')}
        >
          Досрочное погашение
        </Button>
      </div>
      {candidates.length === 0 && (
        <p className="text-xs text-muted-foreground">
          Для досрочного погашения сначала создайте платёж по обязательству.
        </p>
      )}
      <Dialog open={dialog !== null} onOpenChange={(open) => !open && setDialog(null)}>
        <DialogContent>
          {dialog === 'snapshot' && (
            <SnapshotForm
              obligationId={obligationId}
              currency={currency}
              onDone={() => setDialog(null)}
            />
          )}
          {dialog === 'prepayment' && (
            <PrepaymentForm
              obligationId={obligationId}
              currency={currency}
              candidates={candidates}
              onDone={() => setDialog(null)}
            />
          )}
        </DialogContent>
      </Dialog>
    </section>
  )
}

function optionalMinor(form: FormData, name: string) {
  const text = String(form.get(name) ?? '').trim()
  if (!text) return null
  const value = parseMinor(text)
  if (value === null || value < 0)
    throw new Error('Проверьте суммы: не больше двух знаков после запятой.')
  return value
}

function SnapshotForm({
  obligationId,
  currency,
  onDone,
}: {
  obligationId: string
  currency: string
  onDone: () => void
}) {
  const save = useCreateLoanSnapshot(obligationId, {
    onSuccess: () => {
      toast.success('Данные из банка сохранены')
      onDone()
    },
    onError: (e) => toast.error(e.message),
  })
  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        const form = new FormData(e.currentTarget)
        save.mutate(() => {
          const principalMinor = optionalMinor(form, 'principal')
          if (principalMinor === null) throw new Error('Укажите остаток основного долга.')
          return {
            takenOn: String(form.get('takenOn')),
            principalMinor,
            accruedInterestMinor: optionalMinor(form, 'interest'),
            payoffMinor: optionalMinor(form, 'payoff'),
            note: String(form.get('note') ?? ''),
          }
        })
      }}
    >
      <DialogHeader>
        <DialogTitle>Данные из банка</DialogTitle>
        <DialogDescription>
          Перепишите то, что показывает приложение банка. Операции и график не изменятся.
        </DialogDescription>
      </DialogHeader>
      <div className="grid gap-2">
        <Label htmlFor={`${obligationId}-snap-date`}>Дата</Label>
        <Input id={`${obligationId}-snap-date`} name="takenOn" type="date" required />
      </div>
      {(
        [
          ['principal', 'Остаток основного долга'],
          ['interest', 'Начисленные проценты · необязательно'],
          ['payoff', 'Сумма полного погашения · необязательно'],
        ] as const
      ).map(([name, label]) => (
        <div className="grid gap-2" key={name}>
          <Label htmlFor={`${obligationId}-snap-${name}`}>{label}</Label>
          <MoneyInput id={`${obligationId}-snap-${name}`} name={name} currency={currency} />
        </div>
      ))}
      <div className="grid gap-2">
        <Label htmlFor={`${obligationId}-snap-note`}>Примечание · необязательно</Label>
        <Input id={`${obligationId}-snap-note`} name="note" maxLength={500} />
      </div>
      <Button disabled={save.isPending}>Сохранить</Button>
    </form>
  )
}

function PrepaymentForm({
  obligationId,
  currency,
  candidates,
  onDone,
}: {
  obligationId: string
  currency: string
  candidates: FinancialEvent[]
  onDone: () => void
}) {
  const [eventId, setEventId] = useState('')
  const [split, setSplit] = useState(false)
  const save = useCreateLoanPrepayment(obligationId, {
    onSuccess: () => {
      toast.success('Досрочное погашение связано с кредитом')
      onDone()
    },
    onError: (e) => toast.error(e.message),
  })
  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        const form = new FormData(e.currentTarget)
        save.mutate(() => {
          if (!eventId) throw new Error('Выберите платёж.')
          if (!split) return { financialEventId: eventId }
          return {
            financialEventId: eventId,
            principalMinor: optionalMinor(form, 'principal') ?? 0,
            interestMinor: optionalMinor(form, 'interest') ?? 0,
            otherMinor: optionalMinor(form, 'other') ?? 0,
          }
        })
      }}
    >
      <DialogHeader>
        <DialogTitle>Досрочное погашение</DialogTitle>
        <DialogDescription>
          По умолчанию вся сумма уменьшает основной долг. Новый график после погашения задаёт банк —
          скопируйте его в «Копии графика».
        </DialogDescription>
      </DialogHeader>
      <Select value={eventId} onValueChange={setEventId}>
        <SelectTrigger aria-label="Платёж для досрочного погашения">
          <SelectValue placeholder="Выберите платёж" />
        </SelectTrigger>
        <SelectContent>
          {candidates.map((event) => (
            <SelectItem key={event.id} value={event.id}>
              {event.merchant || event.description || 'Платёж'} ·{' '}
              {formatMoney(event.postings[0].amountMinor, event.postings[0].currency)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <label className="flex min-h-11 items-center gap-3 text-sm">
        <Checkbox checked={split} onCheckedChange={(v) => setSplit(v === true)} />
        <span>Банк списал часть на проценты или комиссии</span>
      </label>
      {split && (
        <div className="grid gap-2 sm:grid-cols-3">
          {(
            [
              ['principal', 'Тело'],
              ['interest', 'Проценты'],
              ['other', 'Прочее'],
            ] as const
          ).map(([name, label]) => (
            <div className="grid gap-2" key={name}>
              <Label htmlFor={`${obligationId}-pre-${name}`}>{label}</Label>
              <MoneyInput id={`${obligationId}-pre-${name}`} name={name} currency={currency} />
            </div>
          ))}
        </div>
      )}
      <Button disabled={save.isPending}>Связать</Button>
    </form>
  )
}
