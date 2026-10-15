import { useState, type FormEvent } from 'react'
import type { Account, Category, FinancialEvent } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { fromLocalDateTimeInput, localDateTimeInput as toLocalInput } from '@/shared/lib/date'
import { minorToInput, parseMinor } from '@/shared/lib/money'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
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
import { Textarea } from '@/shared/ui/textarea'
import { useCorrectOperation, type Correction } from '../api/operation-mutations'
import { kindLabels, convertible, failure } from './operation-labels'

export function EditDialog({
  event,
  accounts,
  categories,
  onClose,
  onSaved,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
  onClose: () => void
  onSaved: (next: FinancialEvent) => void
}) {
  const [key] = useState(createIdempotencyKey)
  const correction = useCorrectOperation(event, key)
  const isTransfer = event.kind === 'internal_transfer'
  const debit = event.postings.find((p) => p.direction === 'debit') ?? event.postings[0]
  const credit = event.postings.find((p) => p.direction === 'credit')
  const sourcePosting = isTransfer ? debit : event.postings[0]
  const [form, setForm] = useState({
    merchant: event.merchant,
    description: event.description,
    kind: event.kind as string,
    categoryId: event.categoryId ?? '',
    amount: minorToInput(sourcePosting?.amountMinor ?? null),
    accountId: sourcePosting?.accountId ?? '',
    destinationAccountId: credit?.accountId ?? '',
    occurredAt: event.occurredAt ? toLocalInput(event.occurredAt) : '',
    occurredDate: event.occurredDate ?? '',
  })
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const sameCurrency = accounts.filter((a) => a.currency === sourcePosting?.currency)
  const categoryKind = form.kind === 'income_received' ? 'income' : 'expense'
  const set = (patch: Partial<typeof form>) => setForm((current) => ({ ...current, ...patch }))

  async function save(submit: FormEvent) {
    submit.preventDefault()
    if (pending) return
    const body: Correction = { revision: event.revision }
    if (form.merchant.trim() !== event.merchant) body.merchant = form.merchant.trim()
    if (form.description.trim() !== event.description) body.description = form.description.trim()
    if (form.kind !== event.kind) body.kind = form.kind as Correction['kind']
    if ((form.categoryId || null) !== (event.categoryId ?? null))
      body.categoryId = form.categoryId || null
    const amount = parseMinor(form.amount)
    if (amount === null || amount <= 0)
      return setError('Сумма — больше нуля, не больше двух знаков после запятой.')
    if (amount !== sourcePosting?.amountMinor) body.amountMinor = amount
    if (form.accountId !== sourcePosting?.accountId) body.accountId = form.accountId
    if (isTransfer && form.destinationAccountId !== credit?.accountId)
      body.destinationAccountId = form.destinationAccountId
    if (event.occurredAt && form.occurredAt && form.occurredAt !== toLocalInput(event.occurredAt))
      body.occurredAt = fromLocalDateTimeInput(form.occurredAt)
    if (!event.occurredAt && form.occurredDate && form.occurredDate !== event.occurredDate)
      body.occurredDate = form.occurredDate
    if (Object.keys(body).length === 1) return setError('Ничего не изменилось.')
    setPending(true)
    setError('')
    const result = await correction.mutateAsync(body)
    setPending(false)
    if (!result.data) return setError(failure(result.error, 'Не удалось изменить операцию.'))
    onSaved(result.data)
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent className="max-h-[90dvh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Изменить операцию</DialogTitle>
          <DialogDescription>
            Изменения сохраняются в истории операции. Остатки и статистика пересчитаются.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="space-y-4">
          {convertible.includes(event.kind) && (
            <Field label="Вид">
              <ChoiceSelect
                value={form.kind}
                onValueChange={(kind) =>
                  set({ kind, categoryId: kind === 'purchase' ? form.categoryId : '' })
                }
              >
                {convertible.map((k) => (
                  <option key={k} value={k}>
                    {kindLabels[k]}
                  </option>
                ))}
              </ChoiceSelect>
            </Field>
          )}
          <Field label="Сумма">
            <MoneyInput
              value={form.amount}
              currency={sourcePosting?.currency}
              onChange={(e) => set({ amount: e.target.value })}
            />
          </Field>
          <Field label={isTransfer ? 'Со счёта' : 'Счёт / карта'}>
            <ChoiceSelect value={form.accountId} onValueChange={(accountId) => set({ accountId })}>
              {sameCurrency.map((a) => (
                <option key={a.id} value={a.id}>
                  {accountLabel(a)}
                </option>
              ))}
            </ChoiceSelect>
          </Field>
          {isTransfer && (
            <Field label="На счёт">
              <ChoiceSelect
                value={form.destinationAccountId}
                onValueChange={(destinationAccountId) => set({ destinationAccountId })}
              >
                {sameCurrency.map((a) => (
                  <option key={a.id} value={a.id}>
                    {accountLabel(a)}
                  </option>
                ))}
              </ChoiceSelect>
            </Field>
          )}
          {(form.kind === 'purchase' || form.kind === 'income_received') && (
            <Field label="Категория">
              <ChoiceSelect
                value={form.categoryId}
                onValueChange={(categoryId) => set({ categoryId })}
              >
                <option value="">
                  {form.kind === 'purchase' ? 'Выберите категорию' : 'Без категории'}
                </option>
                {categories
                  .filter((c) => c.kind === categoryKind && (!c.hidden || c.id === form.categoryId))
                  .map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
              </ChoiceSelect>
            </Field>
          )}
          {event.occurredAt ? (
            <Field label="Дата и время">
              <Input
                type="datetime-local"
                value={form.occurredAt}
                onChange={(e) => set({ occurredAt: e.target.value })}
              />
            </Field>
          ) : (
            <Field label="Дата (время неизвестно)">
              <Input
                type="date"
                value={form.occurredDate}
                onChange={(e) => set({ occurredDate: e.target.value })}
              />
            </Field>
          )}
          <Field label="Продавец">
            <Input
              value={form.merchant}
              maxLength={200}
              onChange={(e) => set({ merchant: e.target.value })}
            />
          </Field>
          <Field label="Описание">
            <Textarea
              value={form.description}
              maxLength={500}
              onChange={(e) => set({ description: e.target.value })}
            />
          </Field>
          {error && (
            <Notice tone="danger" title="Не сохранено">
              {error}
            </Notice>
          )}
          <div className="flex flex-wrap justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              Отмена
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? 'Сохраняем…' : 'Сохранить изменения'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
