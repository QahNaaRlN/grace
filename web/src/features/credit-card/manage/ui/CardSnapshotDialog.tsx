import { useState, type FormEvent } from 'react'

import { today } from '@/shared/lib/date'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import { useRecordCardSnapshot } from '../api/use-credit-card'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

/** What the bank app shows. It never changes operations; Grace compares and shows both. */
export function CardSnapshotDialog({
  accountId,
  currency,
  onClose,
}: {
  accountId: string
  currency: string
  onClose: () => void
}) {
  const [key] = useState(createIdempotencyKey)
  const record = useRecordCardSnapshot(accountId, key)
  const [form, setForm] = useState({
    installmentsIncluded: false,
    minimumInGrace: false,
    confirmAllOperations: false,
    periodFrom: '',
    takenOn: today(),
    debt: '',
    available: '',
    grace: '',
    graceDueOn: '',
    minimum: '',
    minimumDueOn: '',
  })
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const set = (patch: Partial<typeof form>) => setForm((current) => ({ ...current, ...patch }))

  async function submit(event: FormEvent) {
    event.preventDefault()
    const debtMinor = parseMinor(form.debt)
    const amounts = [form.available, form.grace, form.minimum]
    if (debtMinor === null) return setError('Введите долг по карте из приложения банка.')
    if (amounts.some((text) => text.trim() !== '' && parseMinor(text) === null))
      return setError('Суммы — числа, не больше двух знаков после запятой.')
    const optional = (text: string) => (text.trim() === '' ? null : parseMinor(text))
    if (pending) return
    setPending(true)
    setError('')
    const result = await record.mutateAsync({
      installmentsIncluded: form.installmentsIncluded,
      minimumInGrace: form.minimumInGrace,
      confirmAllOperations: form.confirmAllOperations,
      periodFrom: form.periodFrom || null,
      takenOn: form.takenOn,
      debtMinor,
      availableMinor: optional(form.available),
      graceAmountMinor: optional(form.grace),
      graceDueOn: form.graceDueOn || null,
      minimumMinor: optional(form.minimum),
      minimumDueOn: form.minimumDueOn || null,
      note: '',
    })
    setPending(false)
    if (!result.data) return setError(errorText(result.error, 'Не удалось сохранить данные банка.'))
    onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !pending && onClose()}>
      <DialogContent className="max-h-dvh overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Данные из банка</DialogTitle>
          <DialogDescription>
            Перенесите то, что показывает приложение банка. Операции в Grace не меняются; если суммы
            разойдутся, Grace покажет обе.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <Field label="На дату">
            <Input
              type="date"
              value={form.takenOn}
              onChange={(e) => set({ takenOn: e.target.value })}
            />
          </Field>
          <Field label="Долг по карте">
            <MoneyInput
              value={form.debt}
              currency={currency}
              onChange={(e) => set({ debt: e.target.value })}
            />
          </Field>
          <Field label="Доступно" hint="Необязательно">
            <MoneyInput
              value={form.available}
              currency={currency}
              onChange={(e) => set({ available: e.target.value })}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Платёж для льготного периода" hint="Необязательно">
              <MoneyInput
                value={form.grace}
                currency={currency}
                onChange={(e) => set({ grace: e.target.value })}
              />
            </Field>
            <Field label="До даты">
              <Input
                type="date"
                value={form.graceDueOn}
                onChange={(e) => set({ graceDueOn: e.target.value })}
              />
            </Field>
            <Field label="Минимальный платёж" hint="Необязательно">
              <MoneyInput
                value={form.minimum}
                currency={currency}
                onChange={(e) => set({ minimum: e.target.value })}
              />
            </Field>
            <Field label="До даты">
              <Input
                type="date"
                value={form.minimumDueOn}
                onChange={(e) => set({ minimumDueOn: e.target.value })}
              />
            </Field>
          </div>
          <label className="flex min-h-11 items-center gap-2">
            <input
              type="checkbox"
              checked={form.installmentsIncluded}
              onChange={(e) => set({ installmentsIncluded: e.target.checked })}
            />{' '}
            Общая сумма для грейса уже включает рассрочки
          </label>
          {form.installmentsIncluded && (
            <div className="space-y-3">
              <Field label="Начало расчётного периода">
                <Input
                  required
                  type="date"
                  value={form.periodFrom}
                  onChange={(e) => set({ periodFrom: e.target.value })}
                />
              </Field>
              <label className="flex min-h-11 items-center gap-2">
                <input
                  type="checkbox"
                  checked={form.minimumInGrace}
                  onChange={(e) => set({ minimumInGrace: e.target.checked })}
                />{' '}
                Минимальный платёж входит в сумму для грейса
              </label>
              <label className="flex min-h-11 items-center gap-2">
                <input
                  type="checkbox"
                  required
                  checked={form.confirmAllOperations}
                  onChange={(e) => set({ confirmAllOperations: e.target.checked })}
                />{' '}
                Подтверждаю: сумма на конец даты учитывает все внесённые операции
              </label>
              <p className="text-sm text-muted-foreground">
                Укажите оставшуюся общую сумму из банка. После новых операций потребуется обновить
                подтверждение.
              </p>
            </div>
          )}
          {error && (
            <Notice tone="danger" title="Данные не сохранены">
              {error}
            </Notice>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              Отмена
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? 'Сохраняем…' : 'Сохранить'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
