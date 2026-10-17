import { useState, type FormEvent } from 'react'
import type { InstallmentPlan } from '@/entities/installment'
import type { Account } from '@/shared/api'
import { APP_UTC_OFFSET } from '@/shared/config'
import { today } from '@/shared/lib/date'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import { useInstallmentTransfer } from '../api/use-installments'

export function InstallmentPaymentForm({
  plan,
  accounts,
  onRecorded,
}: {
  plan: InstallmentPlan
  accounts: Account[]
  onRecorded: (eventId: string) => void
}) {
  const record = useInstallmentTransfer()
  const [source, setSource] = useState('')
  const [amount, setAmount] = useState('')
  const [on, setOn] = useState(today())
  const [key] = useState(createIdempotencyKey)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      const amountMinor = parseMinor(amount)
      if (amountMinor === null || amountMinor <= 0) throw new Error('Введите сумму перевода.')
      const transfer = await record.mutateAsync({
        sourceAccountId: source,
        destinationAccountId: plan.accountId,
        amountMinor,
        occurredAt: new Date(`${on}T12:00:00${APP_UTC_OFFSET}`).toISOString(),
        description: `Погашение: ${plan.name}`,
        key,
      })
      onRecorded(transfer.id)
    } catch (e) {
      setError((e as Error).message)
    }
  }
  return (
    <form onSubmit={submit} className="space-y-3">
      <p className="text-sm">
        Запишите перевод, который уже выполнили в банке. После записи подтвердите его распределение
        по графику.
      </p>
      <Field label="Счёт списания">
        <ChoiceSelect value={source} onValueChange={setSource}>
          <option value="">Выберите счёт</option>
          {accounts
            .filter(
              (a) =>
                a.id !== plan.accountId &&
                a.currency === plan.currency &&
                ['debit', 'cash'].includes(a.type),
            )
            .map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
        </ChoiceSelect>
      </Field>
      <Field label="Сумма перевода">
        <MoneyInput
          required
          currency={plan.currency}
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
      </Field>
      <Field label="Дата оплаты">
        <Input
          type="date"
          required
          max={today()}
          value={on}
          onChange={(e) => setOn(e.target.value)}
        />
      </Field>
      {error && (
        <Notice tone="danger" title="Перевод не записан">
          {error}
        </Notice>
      )}
      <Button disabled={record.isPending}>Записать состоявшийся перевод</Button>
    </form>
  )
}
