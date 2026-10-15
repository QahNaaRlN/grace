import { useState } from 'react'
import { toast } from 'sonner'

import { subscriptionPeriodLabels } from '@/entities/subscription'
import type { Account, Category } from '@/shared/api'
import { DEFAULT_CURRENCY } from '@/shared/lib/currencies'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { parseMinor, minorToInput } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { CurrencySelect } from '@/shared/ui/currency-select'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import {
  useSaveSubscription,
  type Subscription,
  type SubscriptionInput,
} from '../api/subscription-mutations'

export function SubscriptionForm({
  initial,
  accounts,
  categories,
  onSaved,
}: {
  initial?: Subscription
  accounts: Account[]
  categories: Category[]
  onSaved: () => void
}) {
  const [creationKey] = useState(createIdempotencyKey)
  const [period, setPeriod] = useState<SubscriptionInput['period']>(initial?.period ?? 'monthly')
  const [account, setAccount] = useState(initial?.paymentInstrumentId ?? initial?.accountId ?? '')
  const [reminders, setReminders] = useState(initial?.reminders ?? false)
  const [currency, setCurrency] = useState(initial?.currency ?? DEFAULT_CURRENCY)
  const [category, setCategory] = useState(initial?.categoryId ?? '')
  // SPEC-0047: expense categories, as for a purchase; hidden ones only if already chosen.
  const choices = categories.filter((c) => c.kind === 'expense' && (!c.hidden || c.id === category))
  const save = useSaveSubscription(initial, creationKey, {
    onSuccess: () => {
      toast.success('Подписка сохранена')
      onSaved()
    },
    onError: (e) => toast.error(e.message),
  })
  function prepare(form: FormData) {
    return (): SubscriptionInput => {
      const amount = parseMinor(String(form.get('amount')))
      if (amount === null || amount <= 0)
        throw new Error('Укажите положительную сумму с точностью до копейки.')
      if (!category) throw new Error('Выберите категорию расхода.')
      return {
        name: String(form.get('name')),
        amountMinor: amount,
        currency,
        period,
        startsOn: String(form.get('start')),
        endsOn: String(form.get('end')) || null,
        accountId:
          accounts.find(
            (a) => a.id === account || a.paymentInstruments?.some((c) => c.id === account),
          )?.id ?? null,
        paymentInstrumentId:
          accounts.flatMap((a) => a.paymentInstruments ?? []).find((c) => c.id === account)?.id ??
          null,
        categoryId: category,
        paused: initial?.paused ?? false,
        reminders,
        revision: initial?.revision ?? 0,
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
      <div className="space-y-2">
        <Label htmlFor="sub-name">Название</Label>
        <Input id="sub-name" name="name" defaultValue={initial?.name} required maxLength={160} />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="sub-amount">Сумма</Label>
          <MoneyInput
            id="sub-amount"
            name="amount"
            defaultValue={initial ? minorToInput(initial.amountMinor) : ''}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="sub-currency">Валюта</Label>
          <CurrencySelect id="sub-currency" value={currency} onValueChange={setCurrency} />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="sub-category">Категория</Label>
        <ChoiceSelect id="sub-category" value={category} onValueChange={setCategory}>
          <option value="">Выберите категорию</option>
          {choices.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </ChoiceSelect>
      </div>
      <div className="space-y-2">
        <Label htmlFor="sub-period">Период</Label>
        <ChoiceSelect
          id="sub-period"
          value={period}
          onValueChange={(v) => setPeriod(v as SubscriptionInput['period'])}
        >
          {Object.entries(subscriptionPeriodLabels).map(([v, label]) => (
            <option key={v} value={v}>
              {label}
            </option>
          ))}
        </ChoiceSelect>
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="sub-start">Первое списание</Label>
          <Input
            id="sub-start"
            name="start"
            type="date"
            defaultValue={initial?.startsOn}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="sub-end">До даты · необязательно</Label>
          <Input id="sub-end" name="end" type="date" defaultValue={initial?.endsOn ?? ''} />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="sub-account">С какой карты или счёта</Label>
        <ChoiceSelect id="sub-account" value={account} onValueChange={setAccount}>
          <option value="">Не указывать</option>
          {accounts.flatMap((a) => [
            <option key={a.id} value={a.id}>
              {a.name} · счёт
            </option>,
            ...(a.paymentInstruments ?? [])
              .filter((c) => !c.disabledAt || c.id === account)
              .map((c) => (
                <option key={c.id} value={c.id}>
                  {c.bank} *{c.last4}
                </option>
              )),
          ])}
        </ChoiceSelect>
      </div>
      <label className="flex min-h-11 items-start gap-3 text-sm">
        <Checkbox checked={reminders} onCheckedChange={(v) => setReminders(v === true)} />
        <span>Напоминать по почте за 3 и 1 день. Нужна подтверждённая почта в профиле.</span>
      </label>
      <Button className="w-full" disabled={save.isPending}>
        {save.isPending ? 'Сохраняю…' : 'Сохранить'}
      </Button>
    </form>
  )
}
