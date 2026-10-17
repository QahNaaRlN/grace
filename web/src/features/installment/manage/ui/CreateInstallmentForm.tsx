import { initialInstallmentForm } from '../model/initial-form'
import { InstallmentReview } from './InstallmentReview'
import { useState, type FormEvent } from 'react'
import type { Account, Category, FinancialEvent } from '@/shared/api'
import { today } from '@/shared/lib/date'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { formatMoney, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { CurrencySelect } from '@/shared/ui/currency-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import { useCreateInstallment, type InstallmentInput } from '../api/use-installments'
import { InstallmentRows } from './InstallmentRows'
import { parseRows, type RowInput } from '../model/rows'
export function CreateInstallmentForm({
  accounts,
  categories,
  events,
  onDone,
}: {
  accounts: Account[]
  categories: Category[]
  events: FinancialEvent[]
  onDone: () => void
}) {
  const create = useCreateInstallment()
  const [key] = useState(createIdempotencyKey)
  const [form, setForm] = useState(initialInstallmentForm)
  const [rows, setRows] = useState<RowInput[]>([{ dueOn: today(), amount: '' }])
  const [review, setReview] = useState<InstallmentInput | null>(null)
  const [error, setError] = useState('')
  const set = (patch: Partial<typeof form>) => {
    setForm((v) => ({ ...v, ...patch }))
    setReview(null)
  }
  const purchases = events.filter(
    (event) =>
      event.kind === 'purchase' &&
      !event.voidedAt &&
      event.postings.some(
        (p) =>
          p.currency === form.currency &&
          accounts.some(
            (a) =>
              a.id === p.accountId &&
              a.type === (form.mode === 'standalone' ? 'installment' : 'credit'),
          ),
      ),
  )
  function prepare(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      const principalMinor = parseMinor(form.principal)
      const remaining = form.historical ? parseMinor(form.remaining) : principalMinor
      if (principalMinor === null || remaining === null)
        throw new Error('Введите сумму покупки и остаток.')
      const purchase = purchases.find((p) => p.id === form.purchaseEventId)
      const accountId =
        purchase?.postings.find((p) => p.direction === 'debit')?.accountId ?? form.accountId
      const body: InstallmentInput = {
        name: form.name,
        provider: form.provider,
        mode: form.mode,
        currency: form.currency,
        openedOn: form.openedOn,
        observedOn: form.historical ? form.observedOn : form.openedOn,
        principalMinor,
        initialRemainingMinor: remaining,
        historical: form.historical,
        reminders: form.reminders,
        payments: parseRows(rows),
        ...(accountId ? { accountId } : {}),
      }
      if (purchase) body.purchaseEventId = purchase.id
      else if (!form.historical) body.purchase = { categoryId: form.categoryId }
      if (form.firstAmount) {
        const amountMinor = parseMinor(form.firstAmount)
        if (amountMinor === null) throw new Error('Проверьте первый взнос.')
        body.firstPayment = { amountMinor, sourceAccountId: form.firstAccount, on: form.firstOn }
      }
      setReview(body)
    } catch (e) {
      setError((e as Error).message)
    }
  }
  async function confirm() {
    if (!review) return
    try {
      await create.mutateAsync({ body: review, key })
      onDone()
    } catch (e) {
      setError((e as Error).message)
    }
  }
  return (
    <form onSubmit={prepare} className="space-y-4">
      <Notice tone="info" title="Покупка учитывается один раз">
        Погашения записываются переводами. Поддерживаются рассрочки без процентов; возврат связанной
        покупки пока недоступен.
      </Notice>
      <Field label="Название">
        <Input required value={form.name} onChange={(e) => set({ name: e.target.value })} />
      </Field>
      <Field label="Провайдер или банк">
        <Input required value={form.provider} onChange={(e) => set({ provider: e.target.value })} />
      </Field>
      <Field label="Вид рассрочки">
        <ChoiceSelect
          value={form.mode}
          onValueChange={(value) =>
            set({ mode: value as typeof form.mode, accountId: '', purchaseEventId: '' })
          }
        >
          <option value="standalone">Отдельная рассрочка / BNPL</option>
          <option value="credit_card">Внутри кредитной карты</option>
        </ChoiceSelect>
      </Field>
      <Field label="Валюта">
        <CurrencySelect
          value={form.currency}
          onValueChange={(currency) => set({ currency, accountId: '', purchaseEventId: '' })}
        />
      </Field>
      {form.mode === 'credit_card' && (
        <Field label="Кредитный счёт">
          <ChoiceSelect value={form.accountId} onValueChange={(accountId) => set({ accountId })}>
            <option value="">Выберите карту</option>
            {accounts
              .filter((a) => a.type === 'credit' && a.currency === form.currency)
              .map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
          </ChoiceSelect>
        </Field>
      )}
      <Field label="Начало договора">
        <Input
          type="date"
          required
          value={form.openedOn}
          onChange={(e) => set({ openedOn: e.target.value })}
        />
      </Field>
      <Field label="Полная сумма покупки">
        <MoneyInput
          required
          currency={form.currency}
          value={form.principal}
          onChange={(e) => set({ principal: e.target.value })}
        />
      </Field>
      <label className="flex min-h-11 items-center gap-2">
        <input
          type="checkbox"
          checked={form.historical}
          onChange={(e) => set({ historical: e.target.checked })}
        />{' '}
        Договор начался до начала учёта
      </label>
      {form.historical && (
        <>
          <Field label="Начало наблюдения">
            <Input
              type="date"
              required
              value={form.observedOn}
              onChange={(e) => set({ observedOn: e.target.value })}
            />
          </Field>
          <Field label="Остаток тела на эту дату">
            <MoneyInput
              required
              currency={form.currency}
              value={form.remaining}
              onChange={(e) => set({ remaining: e.target.value })}
            />
          </Field>
          <p className="text-sm text-muted-foreground">
            Прошлые взносы и покупка не создаются. В графике укажите только оставшиеся платежи.
          </p>
        </>
      )}
      <Field label="Связь с покупкой">
        <ChoiceSelect
          value={form.purchaseEventId}
          onValueChange={(purchaseEventId) => set({ purchaseEventId })}
        >
          <option value="">{form.historical ? 'Покупка не учтена' : 'Создать покупку'}</option>
          {purchases.map((p) => (
            <option key={p.id} value={p.id}>
              {p.merchant || p.description || 'Покупка'} ·{' '}
              {formatMoney(p.postings[0].amountMinor, form.currency)}
            </option>
          ))}
        </ChoiceSelect>
      </Field>
      {!form.historical && !form.purchaseEventId && (
        <Field label="Категория покупки">
          <ChoiceSelect value={form.categoryId} onValueChange={(categoryId) => set({ categoryId })}>
            <option value="">Выберите категорию</option>
            {categories
              .filter((c) => c.kind === 'expense')
              .map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
          </ChoiceSelect>
        </Field>
      )}
      <InstallmentRows
        rows={rows}
        onChange={(value) => {
          setRows(value)
          setReview(null)
        }}
        currency={form.currency}
      />
      <Field label="Первый взнос" hint="Необязательно; только уже оплаченный">
        <MoneyInput
          currency={form.currency}
          value={form.firstAmount}
          onChange={(e) => set({ firstAmount: e.target.value })}
        />
      </Field>
      {form.firstAmount && (
        <>
          <Field label="Счёт первого взноса">
            <ChoiceSelect
              value={form.firstAccount}
              onValueChange={(firstAccount) => set({ firstAccount })}
            >
              <option value="">Выберите счёт</option>
              {accounts
                .filter((a) => a.currency === form.currency && ['debit', 'cash'].includes(a.type))
                .map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.name}
                  </option>
                ))}
            </ChoiceSelect>
          </Field>
          <Field label="Дата первого взноса">
            <Input
              type="date"
              required
              value={form.firstOn}
              onChange={(e) => set({ firstOn: e.target.value })}
            />
          </Field>
        </>
      )}
      {form.mode === 'standalone' && (
        <label className="flex min-h-11 items-center gap-2">
          <input
            type="checkbox"
            checked={form.reminders}
            onChange={(e) => set({ reminders: e.target.checked })}
          />{' '}
          Напоминать за 3 и 1 день
        </label>
      )}
      {review && <InstallmentReview review={review} />}
      {error && (
        <Notice tone="danger" title="Не сохранено">
          {error}
        </Notice>
      )}
      <div className="flex flex-wrap gap-2">
        <Button type="submit" variant="outline" disabled={create.isPending}>
          Проверить ввод
        </Button>
        {review && (
          <Button type="button" disabled={create.isPending} onClick={() => void confirm()}>
            {create.isPending ? 'Сохраняем…' : 'Подтвердить рассрочку'}
          </Button>
        )}
      </div>
    </form>
  )
}
