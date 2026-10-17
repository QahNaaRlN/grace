import { Button } from '@/shared/ui/button'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'

import type { RowInput } from '../model/rows'
export function InstallmentRows({
  rows,
  onChange,
  currency,
}: {
  rows: RowInput[]
  onChange: (rows: RowInput[]) => void
  currency: string
}) {
  const update = (index: number, patch: Partial<RowInput>) =>
    onChange(rows.map((row, n) => (n === index ? { ...row, ...patch } : row)))
  return (
    <fieldset className="space-y-3">
      <legend className="mb-2 font-medium">График из банка</legend>
      {rows.map((row, index) => (
        <div key={index} className="grid gap-2 rounded-xl border p-3 sm:grid-cols-[1fr_1fr_auto]">
          <Field label={`Дата взноса ${index + 1}`}>
            <Input
              type="date"
              required
              value={row.dueOn}
              disabled={row.locked}
              onChange={(e) => update(index, { dueOn: e.target.value })}
            />
          </Field>
          <Field label={`Сумма взноса ${index + 1}`}>
            <MoneyInput
              required
              currency={currency}
              value={row.amount}
              disabled={row.locked}
              onChange={(e) => update(index, { amount: e.target.value })}
            />
          </Field>
          <Button
            type="button"
            variant="outline"
            disabled={row.locked}
            onClick={() => onChange(rows.filter((_, n) => n !== index))}
            aria-label={`Удалить взнос ${index + 1}`}
          >
            Удалить
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        disabled={rows.length >= 600}
        onClick={() => onChange([...rows, { dueOn: '', amount: '' }])}
      >
        Добавить взнос
      </Button>
    </fieldset>
  )
}
