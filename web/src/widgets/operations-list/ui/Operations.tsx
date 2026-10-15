import { useState, type FormEvent } from 'react'
import { useInfiniteQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { Search } from 'lucide-react'
import { financialEventQueries } from '@/entities/financial-event'
import type { Account, Category, FinancialEvent, Receipt } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { Money } from '@/shared/ui/money'
import { Notice } from '@/shared/ui/status'
import { kindLabels, localDay, dayTitle, accountText, direction } from './operation-display'
import { EventIcon, OperationSheet } from './OperationSheet'

const filterKeys = ['q', 'category', 'account', 'kind', 'from', 'to', 'voided'] as const

type DayTotal = { date: string; currency: string; inflowMinor: number; outflowMinor: number }

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
