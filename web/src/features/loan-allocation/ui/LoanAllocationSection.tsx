import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { loanAllocationQueries } from '@/entities/obligation'
import type { components } from '@/shared/api/generated/schema'
import { dateText } from '@/shared/lib/date'
import { formatMoney, minorToInput, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import {
  useResetLoanAllocation,
  useSaveLoanAllocation,
  type LoanAllocation,
} from '../api/loan-allocation-mutations'

type Parts = components['schemas']['LoanParts']

const sourceText = {
  schedule: 'по графику',
  owner: 'указано вами',
  unknown: 'неизвестно — у строки графика нет тела и процентов',
} as const

function PartsLine({ parts, currency }: { parts: Parts; currency: string }) {
  return (
    <span className="tabular-nums">
      тело {formatMoney(parts.principalMinor, currency)} · проценты{' '}
      {formatMoney(parts.interestMinor, currency)}
      {parts.otherMinor > 0 ? ` · прочее ${formatMoney(parts.otherMinor, currency)}` : ''}
    </span>
  )
}

// SPEC-0053: how a debt payment splits for its loan. Shown only when the
// operation is linked to a loan; the split itself comes from Go.
export function LoanAllocationSection({ eventId }: { eventId: string }) {
  const query = useQuery(loanAllocationQueries.event(eventId))
  const [editing, setEditing] = useState(false)
  const allocation = query.data
  if (!allocation) return null
  return (
    <section aria-label="Распределение по кредиту" className="space-y-2 rounded-xl bg-muted/50 p-3">
      <p className="text-sm font-medium">Распределение по кредиту · {allocation.obligationName}</p>
      {allocation.schedule && (
        <div className="space-y-1 text-sm">
          <p>
            Платёж графика {dateText(allocation.schedule.dueOn)}:{' '}
            {formatMoney(allocation.schedule.amountMinor, allocation.currency)}
          </p>
          <p className="text-muted-foreground">
            {allocation.schedule.parts && (
              <>
                <PartsLine parts={allocation.schedule.parts} currency={allocation.currency} />{' '}
                ·{' '}
              </>
            )}
            {sourceText[allocation.schedule.source]}
          </p>
          {editing ? (
            <AllocationForm allocation={allocation} onDone={() => setEditing(false)} />
          ) : (
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" onClick={() => setEditing(true)}>
                Указать как в выписке
              </Button>
              {allocation.schedule.source === 'owner' && <ResetButton allocation={allocation} />}
            </div>
          )}
        </div>
      )}
      {allocation.prepayment && (
        <p className="text-sm">
          Досрочное погашение {formatMoney(allocation.prepayment.amountMinor, allocation.currency)}
          : <PartsLine parts={allocation.prepayment} currency={allocation.currency} />
        </p>
      )}
    </section>
  )
}

function ResetButton({ allocation }: { allocation: LoanAllocation }) {
  const reset = useResetLoanAllocation(allocation.financialEventId, {
    onSuccess: () => toast.success('Распределение снова по графику'),
    onError: (e) => toast.error(e.message),
  })
  return (
    <Button
      variant="ghost"
      disabled={reset.isPending}
      onClick={() => reset.mutate(allocation.schedule!.revision)}
    >
      Вернуть по графику
    </Button>
  )
}

function AllocationForm({
  allocation,
  onDone,
}: {
  allocation: LoanAllocation
  onDone: () => void
}) {
  const schedule = allocation.schedule!
  const save = useSaveLoanAllocation(allocation.financialEventId, {
    onSuccess: () => {
      toast.success('Распределение сохранено')
      onDone()
    },
    onError: (e) => toast.error(e.message),
  })
  const id = (name: string) => `${allocation.financialEventId}-allocation-${name}`
  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault()
        const form = new FormData(e.currentTarget)
        save.mutate(() => {
          const value = (name: string) => {
            const text = String(form.get(name) ?? '').trim()
            const parsed = text ? parseMinor(text) : 0
            if (parsed === null || parsed < 0)
              throw new Error('Проверьте суммы: не меньше нуля, два знака после запятой.')
            return parsed
          }
          return {
            principalMinor: value('principal'),
            interestMinor: value('interest'),
            otherMinor: value('other'),
            revision: schedule.revision,
          }
        })
      }}
    >
      <div className="grid gap-2 sm:grid-cols-3">
        {(
          [
            ['principal', 'Тело', schedule.parts?.principalMinor],
            ['interest', 'Проценты', schedule.parts?.interestMinor],
            ['other', 'Прочее', schedule.parts?.otherMinor],
          ] as const
        ).map(([name, label, value]) => (
          <div className="grid gap-2" key={name}>
            <Label htmlFor={id(name)}>{label}</Label>
            <MoneyInput
              id={id(name)}
              name={name}
              currency={allocation.currency}
              defaultValue={minorToInput(value ?? null)}
            />
          </div>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        Сумма частей должна равняться {formatMoney(schedule.amountMinor, allocation.currency)}.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button disabled={save.isPending}>Сохранить</Button>
        <Button type="button" variant="ghost" onClick={onDone}>
          Отмена
        </Button>
      </div>
    </form>
  )
}
