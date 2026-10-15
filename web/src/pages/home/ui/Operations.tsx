import { useState, type FormEvent } from 'react'
import { useInfiniteQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { ArrowDownLeft, ArrowRightLeft, Landmark, Scale, Search, Undo2 } from 'lucide-react'

import { CreateReceiptForm } from '@/features/create-receipt'
import { financialEventQueries } from '@/entities/financial-event'
import type { Account, Category, FinancialEvent, Receipt } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { calendarDay, clockText, eventTimeText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { CategoryIcon } from '@/shared/ui/category-icon'
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
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { Money } from '@/shared/ui/money'
import { Notice, StatusBadge } from '@/shared/ui/status'

import { OperationActions, OperationHistory } from './OperationEditor'

const kindLabels: Record<FinancialEvent['kind'], string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
  debt_payment: 'Платёж по обязательству',
  balance_adjustment: 'Корректировка остатка',
}
const kindIcons = {
  income_received: ArrowDownLeft,
  internal_transfer: ArrowRightLeft,
  refund: Undo2,
  debt_payment: Landmark,
  balance_adjustment: Scale,
}
const sourceLabels: Record<string, string> = {
  manual: 'Ручной ввод',
  import: 'Импорт таблицы',
  bank_import: 'Импорт выписки',
  receipt: 'Фото чека',
}
const filterKeys = ['q', 'category', 'account', 'kind', 'from', 'to', 'voided'] as const
type DayTotal = { date: string; currency: string; inflowMinor: number; outflowMinor: number }

function localDay(event: FinancialEvent) {
  if (event.occurredDate) return event.occurredDate
  if (!event.occurredAt) return ''
  return calendarDay(event.occurredAt)
}

function dayTitle(day: string) {
  const text = new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'UTC',
    day: 'numeric',
    month: 'long',
    weekday: 'long',
  }).format(new Date(`${day}T00:00:00Z`))
  // "понедельник, 13 октября" → "13 октября, понедельник"
  const [weekday, rest] = text.split(', ')
  return rest ? `${rest}, ${weekday}` : text
}

function timeText(event: FinancialEvent) {
  if (!event.occurredAt) return 'время неизвестно'
  return clockText(event.occurredAt)
}

function accountName(accounts: Account[], id: string | undefined, fallback: string | undefined) {
  const account = accounts.find((a) => a.id === id)
  return account ? accountLabel(account) : (fallback ?? 'Счёт не указан')
}

function accountText(event: FinancialEvent, accounts: Account[]) {
  if (event.kind === 'internal_transfer') {
    const from = event.postings.find((p) => p.direction === 'debit')
    const to = event.postings.find((p) => p.direction === 'credit')
    return `${accountName(accounts, from?.accountId, from?.accountName)} → ${accountName(accounts, to?.accountId, to?.accountName)}`
  }
  const posting = event.postings[0]
  return accountName(accounts, posting?.accountId, posting?.accountName)
}

function EventIcon({ event }: { event: FinancialEvent }) {
  const Icon = event.kind === 'purchase' ? null : kindIcons[event.kind]
  return (
    <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-secondary text-secondary-foreground">
      {Icon ? (
        <Icon aria-hidden="true" className="size-5" />
      ) : (
        <CategoryIcon icon={event.categoryIcon} />
      )}
    </span>
  )
}

function direction(event: FinancialEvent) {
  if (event.kind === 'internal_transfer') return undefined
  return event.postings[0]?.direction === 'credit' ? ('credit' as const) : ('debit' as const)
}

// SPEC-0043: a compact list grouped by day; details and actions live in a sheet.
export function Operations({
  categories,
  accounts,
  receipts,
}: {
  categories: Category[]
  accounts: Account[]
  receipts: Receipt[]
}) {
  const [params, setParams] = useSearchParams()
  const filter = Object.fromEntries(
    filterKeys.map((key) => [key, params.get(key) ?? '']),
  ) as Record<(typeof filterKeys)[number], string>
  const [selected, setSelected] = useState<FinancialEvent | null>(null)
  const list = useInfiniteQuery(financialEventQueries.page(filter))
  const items = list.data?.pages.flatMap((page) => page.items) ?? []
  const totals = new Map<string, DayTotal[]>()
  for (const total of list.data?.pages.flatMap((page) => page.dayTotals) ?? []) {
    const day = totals.get(total.date) ?? []
    totals.set(total.date, [...day.filter((t) => t.currency !== total.currency), total])
  }
  const days: { day: string; events: FinancialEvent[] }[] = []
  for (const event of items) {
    const day = localDay(event)
    const last = days.at(-1)
    if (last && last.day === day) last.events.push(event)
    else days.push({ day, events: [event] })
  }
  const receiptsByPurchase = new Map(receipts.map((receipt) => [receipt.purchaseEventId, receipt]))
  const active = filterKeys.some((key) => filter[key])

  function apply(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = new FormData(event.currentTarget)
    const next = new URLSearchParams()
    for (const key of filterKeys) {
      const value = String(form.get(key) ?? (filter[key] || '')).trim()
      if (value) next.set(key, value)
    }
    setParams(next)
  }
  function setFilter(key: (typeof filterKeys)[number], value: string) {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next)
  }

  return (
    <div className="space-y-4">
      <Card size="sm">
        <CardContent>
          <form
            key={params.toString()}
            onSubmit={apply}
            className="grid gap-3 sm:grid-cols-2 lg:grid-cols-[2fr_repeat(5,minmax(0,1fr))]"
          >
            <Field label="Поиск">
              <Input
                name="q"
                type="search"
                defaultValue={filter.q}
                placeholder="Магазин или описание"
                maxLength={100}
              />
            </Field>
            <Field label="Категория">
              <ChoiceSelect
                value={filter.category}
                onValueChange={(value) => setFilter('category', value)}
              >
                <option value="">Все</option>
                <option value="uncategorized">Покупки без категории</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </ChoiceSelect>
            </Field>
            <Field label="Счёт">
              <ChoiceSelect
                value={filter.account}
                onValueChange={(value) => setFilter('account', value)}
              >
                <option value="">Все</option>
                {accounts.map((a) => (
                  <option key={a.id} value={a.id}>
                    {accountLabel(a)}
                  </option>
                ))}
              </ChoiceSelect>
            </Field>
            <Field label="Вид">
              <ChoiceSelect value={filter.kind} onValueChange={(value) => setFilter('kind', value)}>
                <option value="">Все</option>
                {Object.entries(kindLabels).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </ChoiceSelect>
            </Field>
            <Field label="С даты">
              <Input name="from" type="date" defaultValue={filter.from} />
            </Field>
            <Field label="До даты, не включая">
              <Input name="to" type="date" defaultValue={filter.to} />
            </Field>
            <div className="flex flex-wrap items-center gap-2 sm:col-span-2 lg:col-span-6">
              <Button type="submit">
                <Search />
                Найти
              </Button>
              <label className="flex min-h-11 items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="size-4 accent-primary"
                  checked={filter.voided === 'true'}
                  onChange={(e) => setFilter('voided', e.target.checked ? 'true' : '')}
                />
                Показать отменённые
              </label>
              {active && (
                <Button type="button" variant="ghost" onClick={() => setParams({})}>
                  Сбросить фильтры
                </Button>
              )}
              <Link
                className="ml-auto text-sm text-primary underline-offset-4 hover:underline"
                to="/categories"
              >
                Управлять категориями
              </Link>
            </div>
          </form>
        </CardContent>
      </Card>

      {list.isPending ? (
        <LoadingSkeleton label="Загружаем операции…" />
      ) : list.isError ? (
        <Notice
          tone="danger"
          title="Операции не загрузились"
          action={
            <Button variant="outline" onClick={() => void list.refetch()}>
              Повторить
            </Button>
          }
        >
          {list.error.message}
        </Notice>
      ) : items.length === 0 ? (
        <Card>
          <CardContent>
            <p className="py-6 text-center text-muted-foreground">
              {active ? 'По этим условиям операций нет.' : 'Операций пока нет.'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card size="sm">
          <CardContent className="space-y-5">
            {days.map(({ day, events }) => (
              <section key={day || 'unknown'} aria-labelledby={`day-${day}`}>
                <header className="flex flex-wrap items-baseline justify-between gap-2 border-b border-border pb-2">
                  <h3 id={`day-${day}`} className="text-sm font-semibold first-letter:uppercase">
                    {day ? dayTitle(day) : 'Дата неизвестна'}
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    {(totals.get(day) ?? [])
                      .map((t) =>
                        [
                          t.outflowMinor > 0 ? `−${formatMoney(t.outflowMinor, t.currency)}` : '',
                          t.inflowMinor > 0 ? `+${formatMoney(t.inflowMinor, t.currency)}` : '',
                        ]
                          .filter(Boolean)
                          .join(' · '),
                      )
                      .filter(Boolean)
                      .join(' · ')}
                  </p>
                </header>
                <ul className="divide-y divide-border">
                  {events.map((event) => (
                    <li key={event.id}>
                      <button
                        type="button"
                        onClick={() => setSelected(event)}
                        className="flex min-h-14 w-full items-center gap-3 rounded-xl px-2 py-2.5 text-left hover:bg-accent"
                      >
                        <EventIcon event={event} />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate font-medium">
                            {event.merchant || event.description || kindLabels[event.kind]}
                          </span>
                          <span className="block truncate text-sm text-muted-foreground">
                            {event.kind === 'purchase'
                              ? (event.categoryName ?? 'Без категории')
                              : kindLabels[event.kind]}{' '}
                            · {accountText(event, accounts)}
                            {!event.occurredAt && ' · время неизвестно'}
                          </span>
                        </span>
                        {event.postings[0] && (
                          <Money
                            valueMinor={event.postings[0].amountMinor}
                            currency={event.postings[0].currency}
                            direction={direction(event)}
                            className="shrink-0 text-right"
                          />
                        )}
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
            <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
              <p className="text-sm text-muted-foreground">Показано операций: {items.length}</p>
              {list.hasNextPage && (
                <Button
                  variant="outline"
                  disabled={list.isFetchingNextPage}
                  onClick={() => void list.fetchNextPage()}
                >
                  {list.isFetchingNextPage ? 'Загружаем…' : 'Показать ещё'}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {selected && (
        <OperationSheet
          event={selected}
          accounts={accounts}
          categories={categories}
          onChanged={setSelected}
          receipt={receiptsByPurchase.get(selected.id)}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  )
}

function OperationSheet({
  event,
  accounts,
  categories,
  receipt,
  onClose,
  onChanged,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
  receipt?: Receipt
  onClose: () => void
  onChanged: (next: FinancialEvent) => void
}) {
  const posting = event.postings[0]
  // Only facts that exist are shown; nothing reads «Не указано».
  const facts: [string, string][] = [
    ['Вид', kindLabels[event.kind]],
    ['Дата', event.occurredAt ? eventTimeText(event) : `${eventTimeText(event)}`],
    [event.kind === 'internal_transfer' ? 'Счета' : 'Счёт / карта', accountText(event, accounts)],
    ...(event.kind === 'purchase'
      ? [['Категория', event.categoryName ?? 'Без категории'] as [string, string]]
      : []),
    ...(event.merchant ? [['Продавец', event.merchant] as [string, string]] : []),
    ...(event.description ? [['Описание', event.description] as [string, string]] : []),
    ['Источник', sourceLabels[event.source] ?? event.source],
  ]
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent
        data-motion="menu"
        className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(420px,calc(100%-1rem))] translate-x-0 translate-y-0 rounded-r-none sm:max-w-[420px]"
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <EventIcon event={event} />
            <span className="min-w-0 break-words">
              {event.merchant || event.description || kindLabels[event.kind]}
            </span>
          </DialogTitle>
          <DialogDescription>{timeText(event)}</DialogDescription>
        </DialogHeader>
        {event.voidedAt && (
          <Notice tone="warning" title="Операция отменена">
            {event.voidReason} Она не учитывается в остатках и статистике.
          </Notice>
        )}
        {event.editedByUser && !event.voidedAt && (
          <p>
            <StatusBadge tone="info">Изменена вручную</StatusBadge>
          </p>
        )}
        {posting && (
          <p>
            <Money
              valueMinor={posting.amountMinor}
              currency={posting.currency}
              direction={direction(event)}
              size="lg"
            />
          </p>
        )}
        <dl className="grace-fields">
          {facts.map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
        <OperationActions
          event={event}
          accounts={accounts}
          categories={categories}
          onChanged={onChanged}
        />
        <OperationHistory event={event} accounts={accounts} categories={categories} />
        {receipt && <ReceiptDetails receipt={receipt} />}
        {event.kind === 'purchase' && !receipt && !event.voidedAt && (
          <CreateReceiptForm purchase={event} />
        )}
      </DialogContent>
    </Dialog>
  )
}

function ReceiptDetails({ receipt }: { receipt: Receipt }) {
  return (
    <section aria-label="Чек" className="rounded-xl bg-muted/50 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2 text-sm">
        <span className="font-medium">Чек · {receipt.seller}</span>
        <Money valueMinor={receipt.totalMinor} currency={receipt.currency} size="sm" />
      </div>
      <ul className="space-y-1 text-sm text-muted-foreground">
        {receipt.items.map((item) => (
          <li key={item.id} className="flex items-start justify-between gap-3">
            <span>
              {item.position}. {item.name} · {item.quantityMilli / 1000}
            </span>
            <span className="shrink-0 tabular-nums">
              {formatMoney(item.lineTotalMinor, receipt.currency)}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
