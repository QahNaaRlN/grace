import { useState, type FormEvent } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { apiClient } from '@/shared/api/client'
import type { components } from '@/shared/api/generated/schema'
import type { Account, Category, FinancialEvent } from '@/shared/api/types'
import { accountLabel } from '@/shared/lib/finance-display'
import {
  dateTimeText,
  fromLocalDateTimeInput,
  localDateTimeInput as toLocalInput,
} from '@/shared/lib/date'
import { minorToInput, parseMinor, formatMoney } from '@/shared/lib/money'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Disclosure } from '@/shared/ui/disclosure'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import { Textarea } from '@/shared/ui/textarea'

type Correction = components['schemas']['FinancialEventCorrection']
const kindLabels: Record<string, string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  debt_payment: 'Платёж по обязательству',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
}
const convertible = ['purchase', 'income_received', 'debt_payment']
const fieldLabels: Record<string, string> = {
  merchant: 'Продавец',
  description: 'Описание',
  occurred: 'Дата',
  kind: 'Вид',
  category: 'Категория',
  amountMinor: 'Сумма',
  account: 'Счёт',
  destinationAccount: 'Счёт зачисления',
  voided: 'Отменена',
}

const refreshKeys = [
  ['financial-events'],
  ['accounts'],
  ['cash-position'],
  ['statistics'],
  ['period-summary'],
  ['categories'],
]

// 409 linked carries a ready message naming what to unlink first.
function failure(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

export function OperationActions({
  event,
  accounts,
  categories,
  onChanged,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
  onChanged: (next: FinancialEvent) => void
}) {
  const [mode, setMode] = useState<'edit' | 'void' | null>(null)
  if (event.voidedAt) return null
  return (
    <>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" onClick={() => setMode('edit')}>
          Изменить
        </Button>
        <Button variant="destructive" onClick={() => setMode('void')}>
          Отменить операцию
        </Button>
      </div>
      {mode === 'edit' && (
        <EditDialog
          event={event}
          accounts={accounts}
          categories={categories}
          onClose={() => setMode(null)}
          onSaved={(next) => {
            setMode(null)
            onChanged(next)
          }}
        />
      )}
      {mode === 'void' && (
        <VoidDialog
          event={event}
          onClose={() => setMode(null)}
          onVoided={(next) => {
            setMode(null)
            onChanged(next)
          }}
        />
      )}
    </>
  )
}

function EditDialog({
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
  const cache = useQueryClient()
  const [key] = useState(createIdempotencyKey)
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
    const result = await apiClient.PATCH('/api/v1/financial-events/{eventId}', {
      params: {
        path: { eventId: event.id },
        header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': key },
      },
      body,
    })
    setPending(false)
    if (!result.data) return setError(failure(result.error, 'Не удалось изменить операцию.'))
    await Promise.all(refreshKeys.map((queryKey) => cache.invalidateQueries({ queryKey })))
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

function VoidDialog({
  event,
  onClose,
  onVoided,
}: {
  event: FinancialEvent
  onClose: () => void
  onVoided: (next: FinancialEvent) => void
}) {
  const cache = useQueryClient()
  const [key] = useState(createIdempotencyKey)
  const [reason, setReason] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function confirm() {
    if (pending || !reason.trim()) return
    setPending(true)
    setError('')
    const result = await apiClient.POST('/api/v1/financial-events/{eventId}/void', {
      params: {
        path: { eventId: event.id },
        header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': key },
      },
      body: { revision: event.revision, reason: reason.trim() },
    })
    setPending(false)
    if (!result.data) return setError(failure(result.error, 'Не удалось отменить операцию.'))
    await Promise.all(refreshKeys.map((queryKey) => cache.invalidateQueries({ queryKey })))
    onVoided(result.data)
  }
  const posting = event.postings[0]
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Отменить операцию?</DialogTitle>
          <DialogDescription>
            {posting
              ? `${formatMoney(posting.amountMinor, posting.currency)} перестанут учитываться в остатках и статистике.`
              : 'Операция перестанет учитываться.'}{' '}
            Запись и история сохранятся; вернуть отменённую операцию нельзя.
          </DialogDescription>
        </DialogHeader>
        <Field label="Причина">
          <Textarea
            value={reason}
            maxLength={500}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Например, дубль из выписки"
          />
        </Field>
        {error && (
          <Notice tone="danger" title="Не отменена">
            {error}
          </Notice>
        )}
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={pending}>
            Не отменять
          </Button>
          <Button
            variant="destructive-solid"
            disabled={pending || !reason.trim()}
            onClick={() => void confirm()}
          >
            {pending ? 'Отменяем…' : 'Отменить операцию'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function OperationHistory({
  event,
  accounts,
  categories,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
}) {
  const history = useQuery({
    queryKey: ['financial-events', 'history', event.id, event.revision],
    enabled: event.revision > 1,
    queryFn: async () => {
      const { data } = await apiClient.GET('/api/v1/financial-events/{eventId}/history', {
        params: { path: { eventId: event.id } },
      })
      if (!data) throw new Error('История недоступна.')
      return data.items
    },
  })
  if (event.revision <= 1) return null
  const currency = event.postings[0]?.currency ?? 'RUB'
  const show = (field: string, value: unknown) => {
    if (value === null || value === undefined || value === '') return '—'
    const text = String(value)
    if (field === 'amountMinor') return formatMoney(Number(value), currency)
    if (field === 'kind') return kindLabels[text] ?? text
    if (field === 'category')
      return categories.find((c) => c.id === text)?.name ?? 'категория удалена'
    if (field === 'account' || field === 'destinationAccount') {
      const a = accounts.find((item) => item.id === text)
      return a ? accountLabel(a) : 'счёт недоступен'
    }
    if (field === 'occurred' && text.includes('T')) return dateTimeText(text)
    return text
  }
  return (
    <div>
      <Disclosure title={`История изменений · ${event.revision - 1}`}>
        {history.isPending ? (
          <p className="px-3 text-sm text-muted-foreground">Загружаем…</p>
        ) : history.isError ? (
          <p className="px-3 text-sm text-destructive">История недоступна.</p>
        ) : (
          <ol className="space-y-3 px-3 text-sm">
            {history.data?.map((item) => (
              <li key={item.revision}>
                <p className="text-muted-foreground">{dateTimeText(item.createdAt)}</p>
                <ul>
                  {item.changes.map((change, i) => (
                    <li key={i}>
                      {fieldLabels[change.field] ?? change.field}:{' '}
                      {change.field === 'voided'
                        ? show(change.field, change.after)
                        : `${show(change.field, change.before)} → ${show(change.field, change.after)}`}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        )}
      </Disclosure>
    </div>
  )
}
