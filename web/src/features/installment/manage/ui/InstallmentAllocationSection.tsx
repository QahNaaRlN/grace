import { useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  installmentQueries,
  type InstallmentAllocation,
  type InstallmentPlan,
} from '@/entities/installment'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { formatMoney, minorToInput, parseMinor } from '@/shared/lib/money'
import { dateText } from '@/shared/lib/date'
import { Button } from '@/shared/ui/button'
import { Disclosure } from '@/shared/ui/disclosure'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import { useAllocateInstallment } from '../api/use-installments'

export function InstallmentAllocationSection({ eventId }: { eventId: string }) {
  const allocation = useQuery(installmentQueries.allocation(eventId))
  const plans = useQuery(installmentQueries.list())
  if (allocation.isPending || plans.isPending) return <p>Загружаем распределение…</p>
  if (allocation.isError || plans.isError)
    return (
      <Notice tone="warning" title="Распределение недоступно">
        Обновите данные перед привязкой перевода.
      </Notice>
    )
  if (!plans.data.length) return null
  return (
    <AllocationEditor
      key={allocation.data.revision}
      data={allocation.data}
      plans={plans.data.filter((p) => p.accountId === allocation.data.accountId)}
    />
  )
}
function AllocationEditor({
  data,
  plans,
}: {
  data: InstallmentAllocation
  plans: InstallmentPlan[]
}) {
  const save = useAllocateInstallment(data.eventId)
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(data.items.map((i) => [i.paymentId, minorToInput(i.amountMinor)])),
  )
  const [reason, setReason] = useState('По данным банка')
  const [error, setError] = useState('')
  const [key, setKey] = useState(createIdempotencyKey)
  const currency = data.currency
  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      const items = Object.entries(values)
        .filter(([, value]) => value.trim() !== '')
        .map(([paymentId, value]) => {
          const amountMinor = parseMinor(value)
          if (amountMinor === null || amountMinor <= 0)
            throw new Error(
              'Для снятия связи очистите сумму. Для оплаты укажите положительную сумму.',
            )
          return { paymentId, amountMinor }
        })
      await save.mutateAsync({ body: { revision: data.revision, reason, items }, key })
      setKey(createIdempotencyKey())
    } catch (e) {
      setError((e as Error).message)
    }
  }
  return (
    <Disclosure title="Распределение по рассрочкам">
      <form onSubmit={submit} className="space-y-4 py-3">
        <p className="text-sm">
          Нераспределено: {formatMoney(data.unallocatedMinor, currency)}. Привязка не создаёт
          движения денег. Для карточной рассрочки используйте распределение банка.
        </p>
        {plans.map((plan) => (
          <fieldset key={plan.id} className="space-y-2">
            <legend className="font-medium">{plan.name}</legend>
            {plan.payments.map((row) => (
              <Field
                key={row.id}
                label={`${dateText(row.dueOn)} · осталось ${formatMoney(row.remainingMinor, plan.currency)}`}
              >
                <MoneyInput
                  currency={plan.currency}
                  value={values[row.id] ?? ''}
                  onChange={(e) => setValues((v) => ({ ...v, [row.id]: e.target.value }))}
                />
              </Field>
            ))}
          </fieldset>
        ))}
        <Field label="Причина распределения или исправления">
          <Input
            required
            maxLength={500}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        </Field>
        {error && (
          <Notice tone="danger" title="Не сохранено">
            {error}
          </Notice>
        )}
        <Button disabled={save.isPending}>Подтвердить распределение</Button>
        <Disclosure title={`История изменений: ${data.history.length}`}>
          <ul className="space-y-2 text-sm">
            {data.history.map((item, n) => (
              <li key={n}>
                {new Date(item.changedAt).toLocaleString('ru-RU')} · {item.reason}
              </li>
            ))}
          </ul>
        </Disclosure>
      </form>
    </Disclosure>
  )
}
