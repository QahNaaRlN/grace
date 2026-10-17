import { useState } from 'react'
import { toast } from 'sonner'
import {
  partsFromPayment,
  partsPayload,
  rowKey,
  ScheduleBuilder,
  type ScheduleRow,
} from '@/entities/payment-schedule'
import type { components } from '@/shared/api/generated/schema'
import { formatMoney, minorToInput, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Label } from '@/shared/ui/label'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Notice } from '@/shared/ui/status'
import {
  useCalculateScheduleParts,
  useSaveLoanDetails,
  useSaveScheduleDraft,
} from '../api/loan-mutations'

export type Draft = components['schemas']['PaymentScheduleDraft']

export type Loan = components['schemas']['LoanDetails']
type RepaymentType = NonNullable<Loan['repaymentType']>
export type Calculation = components['schemas']['PaymentSchedulePartsCalculation']

const warningText: Record<string, string> = {
  payment_below_interest: 'платёж меньше процентов — части не заполнены',
  principal_exhausted: 'основной долг уже погашен — части не заполнены',
  not_differentiated: 'сумма не похожа на дифференцированный платёж',
}

export function LoanForm({ initial, obligationId }: { initial: Loan; obligationId: string }) {
  const [reminders, setReminders] = useState(initial.reminders)
  const [repaymentType, setRepaymentType] = useState<RepaymentType>(
    initial.repaymentType ?? 'unknown',
  )
  const save = useSaveLoanDetails(obligationId, {
    onSuccess: () => toast.success('Сведения о кредите сохранены'),
    onError: (e) => toast.error(e.message),
  })
  function prepare(form: FormData) {
    return () => {
      const principal = String(form.get('principal'))
      const rate = String(form.get('rate'))
      const amount = principal ? parseMinor(principal) : null
      const basis = rate ? parseMinor(rate) : null
      if (
        (principal && (amount === null || amount <= 0)) ||
        (rate && (basis === null || basis < 0))
      )
        throw new Error('Проверьте сумму и ставку: не больше двух знаков после запятой.')
      return {
        lender: String(form.get('lender')),
        agreementNumber: String(form.get('number')),
        agreementOn: String(form.get('date')) || null,
        principalMinor: amount,
        annualRateBasisPoints: basis,
        repaymentType,
        reminders,
        revision: initial.revision,
      }
    }
  }
  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        save.mutate(prepare(new FormData(e.currentTarget)))
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        {[
          ['lender', 'Кредитор', initial.lender],
          ['number', 'Номер договора', initial.agreementNumber],
          ['date', 'Дата договора', initial.agreementOn ?? ''],
          ['principal', 'Сумма выдачи · необязательно', minorToInput(initial.principalMinor)],
          [
            'rate',
            'Годовая ставка, % · необязательно',
            minorToInput(initial.annualRateBasisPoints),
          ],
        ].map(([name, label, value]) => (
          <div className="space-y-2" key={name}>
            <Label htmlFor={`${obligationId}-${name}`}>{label}</Label>
            {name === 'principal' ? (
              <MoneyInput
                id={`${obligationId}-${name}`}
                name={name}
                currency="RUB"
                defaultValue={value}
              />
            ) : (
              <Input
                id={`${obligationId}-${name}`}
                name={name}
                type={name === 'date' ? 'date' : 'text'}
                inputMode={name === 'rate' ? 'decimal' : undefined}
                defaultValue={value}
                maxLength={160}
              />
            )}
          </div>
        ))}
      </div>
      <div className="grid gap-2 sm:max-w-xs">
        <Label htmlFor={`${obligationId}-repayment`}>Вид погашения</Label>
        <ChoiceSelect
          id={`${obligationId}-repayment`}
          value={repaymentType}
          onValueChange={(value) => setRepaymentType(value as RepaymentType)}
        >
          <option value="annuity">Аннуитетный — равные платежи</option>
          <option value="differentiated">Дифференцированный — равное тело</option>
          <option value="unknown">Не указан</option>
        </ChoiceSelect>
      </div>
      <p className="text-xs text-muted-foreground">
        График задаёт банк. Вид погашения, ставка, сумма выдачи и дата договора нужны только для
        расчёта тела и процентов в копии графика — результат сверьте с графиком банка.
      </p>
      <label className="flex min-h-11 items-start gap-3 text-sm">
        <Checkbox checked={reminders} onCheckedChange={(v) => setReminders(v === true)} />
        <span>Напоминать по подтверждённой почте за 3 и 1 день</span>
      </label>
      <Button variant="outline" disabled={save.isPending}>
        Сохранить сведения
      </Button>
    </form>
  )
}

export function ScheduleEditor({
  draft,
  obligationId,
  onSaved,
  onApply,
  calculation,
  onCalculated,
}: {
  draft: Draft
  obligationId: string
  onSaved: (d: Draft) => void
  onApply: () => void
  // Kept by the parent: a saved calculation bumps the revision and remounts the editor.
  calculation: Calculation | null
  onCalculated: (c: Calculation | null) => void
}) {
  // SPEC-0037: the same three ways to fill a schedule as a new obligation; paid rows stay locked.
  const [rows, setRows] = useState<ScheduleRow[]>(() =>
    draft.payments.map((p) => ({
      key: rowKey(),
      dueOn: p.dueOn,
      amount: minorToInput(p.amountMinor),
      locked: p.paidMinor > 0,
      sourceId: p.sourceId,
      paidMinor: p.paidMinor,
      ...partsFromPayment(p),
    })),
  )
  const calculate = useCalculateScheduleParts(obligationId, {
    onSuccess: (result) => {
      onCalculated(result)
      onSaved(result.draft)
      toast.success('Тело и проценты рассчитаны. Сверьте их с графиком банка.')
    },
    onError: (e) => toast.error(e.message),
  })
  const [dirty, setDirty] = useState(false)
  const save = useSaveScheduleDraft(obligationId, draft.id, {
    onSuccess: (d) => {
      onCalculated(null)
      onSaved(d)
      toast.success('Черновик сохранён')
    },
    onError: (e) => toast.error(e.message),
  })
  function prepare() {
    const payments = rows.map((p) => {
      const amountMinor = parseMinor(p.amount)
      if (amountMinor === null || amountMinor <= 0)
        throw new Error('Укажите положительную сумму каждого платежа.')
      const parts = partsPayload(p)
      return {
        sourceId: p.sourceId ?? '',
        dueOn: p.dueOn,
        paidMinor: p.paidMinor ?? 0,
        amountMinor,
        ...parts,
        ...('principalMinor' in parts ? { partsSource: p.partsSource ?? 'bank' } : {}),
      }
    })
    return { revision: draft.revision, payments }
  }
  return (
    <div className="space-y-4">
      <ScheduleBuilder
        rows={rows}
        onChange={(next) => {
          setDirty(true)
          setRows(next)
        }}
        currency="RUB"
        idPrefix={`draft-${draft.id}`}
        initialMode="manual"
        withParts
      />
      {calculation && <CalculationNotes calculation={calculation} />}
      <div className="flex flex-wrap gap-2">
        <Button disabled={!dirty || save.isPending} onClick={() => save.mutate(prepare)}>
          Сохранить черновик
        </Button>
        <Button
          variant="outline"
          className="h-auto min-h-11 max-w-full whitespace-normal py-2"
          disabled={dirty || save.isPending || calculate.isPending}
          onClick={() => calculate.mutate(draft)}
        >
          Рассчитать тело и проценты
        </Button>
        <Button
          variant="outline"
          className="h-auto min-h-11 max-w-full whitespace-normal py-2"
          disabled={dirty || save.isPending}
          onClick={onApply}
        >
          Применить сохранённый график
        </Button>
      </div>
      {dirty && (
        <p className="text-sm text-muted-foreground">
          Сначала сохраните изменения. Даты должны идти по порядку.
        </p>
      )}
    </div>
  )
}

// SPEC-0053: what Grace could not split and how far the calculated principal is
// from the issued amount; the owner decides before applying the copy.
function CalculationNotes({ calculation }: { calculation: Calculation }) {
  const warnings = calculation.rows
    .map((row, index) => ({ number: index + 1, warning: row.warning }))
    .filter((row) => row.warning)
  if (warnings.length === 0 && calculation.principalDifferenceMinor === 0) return null
  return (
    <Notice tone="warning" title="Проверьте расчёт">
      <ul className="space-y-1">
        {calculation.principalDifferenceMinor !== 0 && (
          <li>
            Тело по расчёту отличается от суммы выдачи на{' '}
            {formatMoney(calculation.principalDifferenceMinor, 'RUB')}.
          </li>
        )}
        {warnings.map((row) => (
          <li key={row.number}>
            Платёж {row.number}: {warningText[row.warning!] ?? row.warning}
          </li>
        ))}
      </ul>
    </Notice>
  )
}
