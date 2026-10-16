import { useState, type FormEvent } from 'react'

import type { Account } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { fromLocalDateTimeInput, localDateTimeInput } from '@/shared/lib/date'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { formatMoney, minorToInput, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
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
import { useRepayCreditCard } from '../api/use-repay-credit-card'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

export function RepayCardDialog({
  card,
  payingAccounts,
  suggestedMinor,
  debtMinor,
  onClose,
}: {
  card: Account
  /** Own accounts in the card's currency that can pay it. */
  payingAccounts: Account[]
  suggestedMinor: number | null
  debtMinor: number | null
  onClose: () => void
}) {
  const [key] = useState(createIdempotencyKey)
  const repay = useRepayCreditCard()
  const [from, setFrom] = useState(payingAccounts[0]?.id ?? '')
  const [amount, setAmount] = useState(suggestedMinor ? minorToInput(suggestedMinor) : '')
  const [moment, setMoment] = useState(() => localDateTimeInput(new Date().toISOString()))
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const amountMinor = parseMinor(amount)
  const overpay = amountMinor !== null && debtMinor !== null && amountMinor > debtMinor

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!from) return setError('Выберите счёт, с которого платите.')
    if (amountMinor === null || amountMinor <= 0) return setError('Введите сумму погашения.')
    if (!moment) return setError('Укажите дату и время платежа.')
    if (pending) return
    setPending(true)
    setError('')
    const result = await repay.mutateAsync({
      fromAccountId: from,
      cardAccountId: card.id,
      amountMinor,
      occurredAt: fromLocalDateTimeInput(moment),
      key,
    })
    setPending(false)
    if (result.error) return setError(errorText(result.error, 'Не удалось записать погашение.'))
    onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !pending && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Погасить карту</DialogTitle>
          <DialogDescription>
            Grace запишет перевод на кредитный счёт: долг карты уменьшится, а расходом это не станет
            — покупки уже учтены.
          </DialogDescription>
        </DialogHeader>
        {payingAccounts.length === 0 ? (
          <Notice tone="warning" title="Нет счёта для оплаты">
            Добавьте дебетовый счёт или наличные в валюте {card.currency}.
          </Notice>
        ) : (
          <form onSubmit={submit} className="space-y-4">
            <Field label="Списать со счёта">
              <ChoiceSelect value={from} onValueChange={setFrom}>
                {payingAccounts.map((account) => (
                  <option key={account.id} value={account.id}>
                    {accountLabel(account)}
                  </option>
                ))}
              </ChoiceSelect>
            </Field>
            <Field
              label="Сумма"
              hint={
                overpay
                  ? `Больше долга (${formatMoney(debtMinor, card.currency)}): остаток станет переплатой.`
                  : undefined
              }
            >
              <MoneyInput
                value={amount}
                currency={card.currency}
                onChange={(e) => setAmount(e.target.value)}
              />
            </Field>
            <Field label="Дата и время">
              <Input
                type="datetime-local"
                value={moment}
                onChange={(e) => setMoment(e.target.value)}
              />
            </Field>
            {error && (
              <Notice tone="danger" title="Погашение не записано">
                {error}
              </Notice>
            )}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
                Отмена
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Записываем…' : 'Погасить'}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
