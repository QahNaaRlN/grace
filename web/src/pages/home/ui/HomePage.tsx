import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Activity, ArrowDownRight, ArrowRightLeft, BarChart3, CalendarClock, CircleCheck, Link2, Sparkles, WalletCards } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { CreateFinancialEventForm } from '@/features/create-financial-event'
import { CreateObligationForm } from '@/features/create-obligation'
import { CreateReceiptForm } from '@/features/create-receipt'
import { apiClient } from '@/shared/api/client'
import type { Account, CashPosition, FinancialEvent, Obligation, PeriodSummary, Receipt, ScheduledPayment } from '@/shared/api/types'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { AppLayout } from '@/shared/ui/layout/AppLayout'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'

const kindLabels: Record<FinancialEvent['kind'], string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
  debt_payment: 'Платёж по обязательству',
}

function formatMoney(valueMinor: number, currency: string) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency }).format(valueMinor / 100)
}

function localDateValue(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function localBoundary(value: string) {
  const date = new Date(`${value}T00:00:00`)
  const offset = -date.getTimezoneOffset()
  const sign = offset >= 0 ? '+' : '-'
  const hours = String(Math.floor(Math.abs(offset) / 60)).padStart(2, '0')
  const minutes = String(Math.abs(offset) % 60).padStart(2, '0')
  return `${value}T00:00:00${sign}${hours}:${minutes}`
}

function initialPeriod() {
  const now = new Date()
  return {
    from: localDateValue(new Date(now.getFullYear(), now.getMonth(), 1)),
    to: localDateValue(new Date(now.getFullYear(), now.getMonth() + 1, 1)),
  }
}

function initialThrough() {
  const now = new Date()
  return localDateValue(new Date(now.getFullYear(), now.getMonth() + 1, now.getDate()))
}

async function loadAccounts() {
  const { data } = await apiClient.GET('/api/v1/accounts')
  if (!data) throw new Error('Не удалось загрузить счета')
  return data.items
}

async function loadCategories() {
  const { data } = await apiClient.GET('/api/v1/categories')
  if (!data) throw new Error('Не удалось загрузить категории')
  return data.items
}

async function loadEvents() {
  const { data } = await apiClient.GET('/api/v1/financial-events')
  if (!data) throw new Error('Не удалось загрузить финансовые события')
  return data.items
}

async function loadReceipts() {
  const { data } = await apiClient.GET('/api/v1/receipts')
  if (!data) throw new Error('Не удалось загрузить чеки')
  return data.items
}

async function loadObligations(asOf: string) {
  const { data } = await apiClient.GET('/api/v1/obligations', {
    params: { query: { asOf } },
  })
  if (!data) throw new Error('Не удалось загрузить обязательства')
  return data.items
}

function Accounts({ items }: { items: Account[] }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {items.map((account) => (
        <Card key={account.id} className="relative isolate overflow-hidden bg-gradient-to-br from-primary to-emerald-700 text-primary-foreground ring-0">
          <div className="absolute -top-16 -right-12 -z-10 size-40 rounded-full bg-white/10" />
          <CardHeader className="pb-1">
            <CardDescription className="text-primary-foreground/70">{account.type === 'debit' ? 'Дебетовый счёт' : account.type === 'credit' ? 'Кредитный счёт' : account.type === 'cash' ? 'Наличные' : 'Рассрочка'}</CardDescription>
            <CardTitle className="text-lg text-primary-foreground">{account.name}</CardTitle>
          </CardHeader>
          <CardContent className="text-3xl font-semibold tracking-tight">
            {formatMoney(account.balanceMinor, account.currency)}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function ReceiptDetails({ receipt }: { receipt: Receipt }) {
  return (
    <div className="mt-3 rounded-lg bg-muted/50 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2 text-sm">
        <span className="font-medium">Чек · {receipt.seller}</span>
        <span>{formatMoney(receipt.totalMinor, receipt.currency)}</span>
      </div>
      <div className="space-y-1 text-sm text-muted-foreground">
        {receipt.items.map((item) => (
          <div key={item.id} className="flex items-start justify-between gap-3">
            <span>{item.position}. {item.name} · {item.quantityMilli / 1000}</span>
            <span className="shrink-0">{formatMoney(item.lineTotalMinor, receipt.currency)}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

function Events({ items, receipts }: { items: FinancialEvent[], receipts: Receipt[] }) {
  const receiptsByPurchase = new Map(receipts.map((receipt) => [receipt.purchaseEventId, receipt]))
  return (
    <Card>
      <CardHeader>
        <CardTitle>Финансовые события</CardTitle>
        <CardDescription>
          Каждое событие объясняет одну или несколько связанных проводок.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">Событий пока нет.</p>
        ) : (
          <div className="divide-y">
            {items.map((item) => {
              const posting = item.postings[0]
              const isTransfer = item.kind === 'internal_transfer'
              const positive = posting?.direction === 'credit'
              const sourcePosting = item.postings.find((value) => value.direction === 'debit')
              const destinationPosting = item.postings.find((value) => value.direction === 'credit')
              const accountDescription = isTransfer
                ? `${sourcePosting?.accountName ?? 'Неизвестный счёт'} → ${destinationPosting?.accountName ?? 'Неизвестный счёт'}`
                : posting?.accountName
              const receipt = receiptsByPurchase.get(item.id)
              return (
                <article key={item.id} className="py-3">
                  <div className="flex items-center justify-between gap-4">
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="rounded-full bg-muted p-2">
                        {isTransfer
                          ? <ArrowRightLeft aria-hidden="true" className="size-4" />
                          : <ArrowDownRight aria-hidden="true" className="size-4" />}
                      </div>
                      <div className="min-w-0">
                        <p className="truncate font-medium">
                          {item.merchant || item.description || kindLabels[item.kind]}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {item.categoryName ?? kindLabels[item.kind]} · {accountDescription}
                        </p>
                      </div>
                    </div>
                    {posting && (
                      <div className="shrink-0 space-y-1 text-right">
                        <p className={positive ? 'font-medium text-emerald-700' : 'font-medium text-destructive'}>
                          {isTransfer ? '' : positive ? '+' : '−'}
                          {formatMoney(posting.amountMinor, posting.currency)}
                        </p>
                        <Badge variant="secondary">{kindLabels[item.kind]}</Badge>
                      </div>
                    )}
                  </div>
                  {receipt && <ReceiptDetails receipt={receipt} />}
                  {item.kind === 'purchase' && !receipt && (
                    <div className="mt-3 pl-11">
                      <CreateReceiptForm purchase={item} />
                    </div>
                  )}
                </article>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

const obligationKindLabels: Record<Obligation['kind'], string> = {
  loan: 'Кредит',
  installment: 'Рассрочка',
  other_fixed: 'Другое',
}

const paymentStatusLabels: Record<ScheduledPayment['status'], string> = {
  planned: 'Запланирован',
  overdue: 'Просрочен',
  partially_paid: 'Частично оплачен',
  paid: 'Оплачен',
}

function SettlementControl({
  obligation,
  payment,
  events,
  usedEventIDs,
}: {
  obligation: Obligation
  payment: ScheduledPayment
  events: FinancialEvent[]
  usedEventIDs: Set<string>
}) {
  const queryClient = useQueryClient()
  const [eventID, setEventID] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const candidates = events.filter((event) => {
    const posting = event.postings[0]
    return event.kind === 'debt_payment' && event.postings.length === 1 &&
      !usedEventIDs.has(event.id) && posting?.currency === obligation.currency &&
      posting.direction === 'debit' && posting.amountMinor <= payment.remainingMinor
  })

  if (payment.remainingMinor === 0) return null
  if (candidates.length === 0) {
    return <p className="mt-2 text-xs text-muted-foreground">Сначала создайте подходящий платёж по обязательству.</p>
  }

  async function settle() {
    if (!eventID) return
    setIsSubmitting(true)
    const result = await apiClient.POST('/api/v1/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements', {
      params: {
        path: { obligationId: obligation.id, paymentId: payment.id },
        header: { 'Idempotency-Key': createIdempotencyKey() },
      },
      body: { financialEventId: eventID },
    })
    setIsSubmitting(false)
    if (result.error) {
      toast.error(result.error.message)
      return
    }
    setEventID('')
    await queryClient.invalidateQueries({ queryKey: ['obligations'] })
    toast.success('Фактический платёж связан с графиком')
  }

  return (
    <div className="mt-2 flex flex-wrap gap-2">
      <Select value={eventID} onValueChange={setEventID}>
        <SelectTrigger className="min-w-56 flex-1"><SelectValue placeholder="Выберите фактический платёж" /></SelectTrigger>
        <SelectContent>
          {candidates.map((event) => (
            <SelectItem key={event.id} value={event.id}>
              {event.merchant || event.description || 'Платёж'} · {formatMoney(event.postings[0].amountMinor, event.postings[0].currency)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button className="w-full sm:w-auto" type="button" variant="outline" disabled={!eventID || isSubmitting} onClick={settle}>
        <Link2 aria-hidden="true" className="mr-1 size-4" />
        {isSubmitting ? 'Связываю…' : 'Связать'}
      </Button>
    </div>
  )
}

function Obligations({ items, events }: { items: Obligation[], events: FinancialEvent[] }) {
  const usedEventIDs = new Set(items.flatMap((item) => item.payments.flatMap((payment) => payment.settlements.map((settlement) => settlement.financialEventId))))
  const remainingByCurrency = items.reduce<Record<string, number>>((totals, item) => {
    totals[item.currency] = (totals[item.currency] ?? 0) + item.remainingMinor
    return totals
  }, {})

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2"><CalendarClock className="size-5" /> Обязательства</CardTitle>
        <CardDescription>
          Осталось: {Object.entries(remainingByCurrency).map(([currency, amount]) => formatMoney(amount, currency)).join(' · ') || 'нет обязательств'}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {items.length === 0 ? <p className="text-sm text-muted-foreground">Обязательств пока нет.</p> : items.map((item) => (
          <article key={item.id} className="rounded-lg border p-4">
            <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 className="font-semibold">{item.name}</h3>
                <p className="text-sm text-muted-foreground">{obligationKindLabels[item.kind]} · оплачено {formatMoney(item.paidMinor, item.currency)} из {formatMoney(item.totalMinor, item.currency)}</p>
              </div>
              <Badge variant={item.status === 'completed' ? 'secondary' : 'outline'}>{item.status === 'completed' ? 'Завершено' : `Осталось ${formatMoney(item.remainingMinor, item.currency)}`}</Badge>
            </div>
            <div className="space-y-3">
              {item.payments.map((payment) => (
                <div key={payment.id} className="rounded-md bg-muted/50 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                    <div>
                      <span className="font-medium">{new Date(`${payment.dueOn}T00:00:00Z`).toLocaleDateString('ru-RU', { timeZone: 'UTC' })}</span>
                      <span className="text-muted-foreground"> · осталось {formatMoney(payment.remainingMinor, item.currency)}</span>
                    </div>
                    <Badge variant={payment.isOverdue ? 'destructive' : 'secondary'}>{paymentStatusLabels[payment.status]}{payment.isOverdue && payment.status === 'partially_paid' ? ' · просрочен' : ''}</Badge>
                  </div>
                  <SettlementControl obligation={item} payment={payment} events={events} usedEventIDs={usedEventIDs} />
                </div>
              ))}
            </div>
          </article>
        ))}
      </CardContent>
    </Card>
  )
}

function Analytics() {
  const defaults = initialPeriod()
  const [fromDate, setFromDate] = useState(defaults.from)
  const [toDate, setToDate] = useState(defaults.to)
  const [through, setThrough] = useState(initialThrough)
  const validPeriod = fromDate !== '' && toDate !== '' && fromDate < toDate
  const period = useQuery<PeriodSummary>({
    queryKey: ['period-summary', fromDate, toDate],
    enabled: validPeriod,
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/analytics/period-summary', {
        params: { query: { from: localBoundary(fromDate), to: localBoundary(toDate) } },
      })
      if (!data) throw new Error(error?.message ?? 'Не удалось загрузить сводку периода')
      return data
    },
  })
  const cash = useQuery<CashPosition>({
    queryKey: ['cash-position', through],
    enabled: through !== '',
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/analytics/cash-position', {
        params: { query: { through } },
      })
      if (!data) throw new Error(error?.message ?? 'Не удалось загрузить доступность денег')
      return data
    },
  })

  return (
    <div className="grid gap-6 xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><BarChart3 className="size-5" /> Сводка периода</CardTitle>
          <CardDescription>Фактический поток денег; конец периода не включается.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-2"><Label htmlFor="summary-from">С даты</Label><Input id="summary-from" type="date" value={fromDate} onChange={(event) => setFromDate(event.target.value)} /></div>
            <div className="space-y-2"><Label htmlFor="summary-to">До даты, не включая</Label><Input id="summary-to" type="date" value={toDate} onChange={(event) => setToDate(event.target.value)} /></div>
          </div>
          {!validPeriod && <p className="text-sm text-destructive">Дата окончания должна быть позже даты начала.</p>}
          {period.error && <p className="text-sm text-destructive">{period.error.message}</p>}
          {period.isPending && validPeriod && <p className="text-sm text-muted-foreground">Рассчитываю…</p>}
          {period.data?.currencies.length === 0 && <p className="text-sm text-muted-foreground">За период нет финансовых данных.</p>}
          {period.data?.currencies.map((item) => (
            <article key={item.currency} className="space-y-3 rounded-lg border p-4">
              <div className="flex items-center justify-between"><h3 className="font-semibold">{item.currency}</h3><Badge variant="outline">Итог {formatMoney(item.closingBalanceMinor, item.currency)}</Badge></div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <span className="text-muted-foreground">Начальный остаток</span><span className="text-right">{formatMoney(item.openingBalanceMinor, item.currency)}</span>
                <span className="text-muted-foreground">Доходы</span><span className="text-right">{formatMoney(item.incomeMinor, item.currency)}</span>
                <span className="text-muted-foreground">Покупки</span><span className="text-right">{formatMoney(item.purchasesMinor, item.currency)}</span>
                <span className="text-muted-foreground">Возвраты</span><span className="text-right">{formatMoney(item.refundsMinor, item.currency)}</span>
                <span className="text-muted-foreground">Платежи по обязательствам</span><span className="text-right">{formatMoney(item.debtPaymentsMinor, item.currency)}</span>
                <span className="text-muted-foreground">Внешний денежный поток</span><span className="text-right font-medium">{formatMoney(item.netExternalCashFlowMinor, item.currency)}</span>
                <span className="text-muted-foreground">Внутренние переводы, справочно</span><span className="text-right">{formatMoney(item.internalTransferVolumeMinor, item.currency)}</span>
              </div>
              {item.expenseCategories.length > 0 && (
                <div className="border-t pt-3">
                  <p className="mb-2 text-sm font-medium">Категории покупок</p>
                  <div className="space-y-1 text-sm">{item.expenseCategories.map((category) => (
                    <div key={category.categoryId ?? category.categoryName} className="flex justify-between gap-3"><span className="text-muted-foreground">{category.categoryName}</span><span>{formatMoney(category.netExpenseMinor, item.currency)}</span></div>
                  ))}</div>
                </div>
              )}
            </article>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><WalletCards className="size-5" /> Доступно после обязательств</CardTitle>
          <CardDescription>Текущие ликвидные счета минус известные платежи до выбранной даты.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2"><Label htmlFor="cash-through">Обязательства по дату включительно</Label><Input id="cash-through" type="date" value={through} onChange={(event) => setThrough(event.target.value)} /></div>
          {cash.error && <p className="text-sm text-destructive">{cash.error.message}</p>}
          {cash.isPending && <p className="text-sm text-muted-foreground">Рассчитываю…</p>}
          {cash.data?.currencies.length === 0 && <p className="text-sm text-muted-foreground">Нет ликвидных счетов и обязательств до горизонта.</p>}
          {cash.data?.currencies.map((item) => (
            <article key={item.currency} className="space-y-3 rounded-lg border p-4">
              <div className="flex items-center justify-between"><h3 className="font-semibold">{item.currency}</h3><Badge variant="outline">Доступно {formatMoney(item.availableAfterObligationsMinor, item.currency)}</Badge></div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <span className="text-muted-foreground">Ликвидный остаток</span><span className="text-right">{formatMoney(item.liquidBalanceMinor, item.currency)}</span>
                <span className="text-muted-foreground">Просрочено</span><span className="text-right">{formatMoney(item.overdueObligationsMinor, item.currency)}</span>
                <span className="text-muted-foreground">Предстоит</span><span className="text-right">{formatMoney(item.upcomingObligationsMinor, item.currency)}</span>
                <span className="font-medium">Всего обязательств</span><span className="text-right font-medium">{formatMoney(item.obligationsThroughMinor, item.currency)}</span>
              </div>
              {item.scheduledPayments.length > 0 && <div className="space-y-1 border-t pt-3 text-sm">{item.scheduledPayments.map((payment) => (
                <div key={payment.scheduledPaymentId} className="flex justify-between gap-3"><span className="text-muted-foreground">{payment.obligationName} · {payment.dueOn}{payment.timing === 'overdue' ? ' · просрочен' : ''}</span><span className="shrink-0">{formatMoney(payment.remainingMinor, item.currency)}</span></div>
              ))}</div>}
            </article>
          ))}
          <p className="text-xs text-muted-foreground">Это арифметическая позиция по подтверждённым данным, а не бюджет, прогноз или рекомендация.</p>
        </CardContent>
      </Card>
    </div>
  )
}

export function HomePage() {
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: loadAccounts })
  const categories = useQuery({ queryKey: ['categories'], queryFn: loadCategories })
  const events = useQuery({ queryKey: ['financial-events'], queryFn: loadEvents })
  const receipts = useQuery({ queryKey: ['receipts'], queryFn: loadReceipts })
  const asOf = new Date().toISOString().slice(0, 10)
  const obligations = useQuery({ queryKey: ['obligations', asOf], queryFn: () => loadObligations(asOf) })
  const error = accounts.error ?? categories.error ?? events.error ?? receipts.error ?? obligations.error
  const isPending = accounts.isPending || categories.isPending || events.isPending || receipts.isPending || obligations.isPending

  return (
    <AppLayout>
      <div className="space-y-8 sm:space-y-10">
        <section className="relative overflow-hidden rounded-3xl bg-foreground px-5 py-6 text-background shadow-xl shadow-emerald-950/10 sm:px-8 sm:py-8">
          <div className="absolute -top-24 -right-20 size-64 rounded-full bg-primary/35 blur-3xl" />
          <div className="relative flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
            <div className="max-w-2xl">
              <p className="mb-2 flex items-center gap-2 text-sm font-semibold text-emerald-300"><Sparkles className="size-4" aria-hidden="true" /> Ваша финансовая картина</p>
              <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">Деньги — одним взглядом</h1>
              <p className="mt-2 max-w-xl text-sm leading-relaxed text-background/70 sm:text-base">Счета, обязательства и фактический денежный поток собраны в одном месте.</p>
            </div>
            <div className={error ? 'flex w-fit items-center gap-2 rounded-full bg-destructive/20 px-3 py-2 text-sm font-medium text-red-100' : 'flex w-fit items-center gap-2 rounded-full bg-white/10 px-3 py-2 text-sm font-medium text-background'}>
              {error ? <Activity aria-hidden="true" className="size-4" /> : <CircleCheck aria-hidden="true" className="size-4 text-emerald-300" />}
              {error ? 'Часть данных не загрузилась' : isPending ? 'Обновляем данные…' : 'Данные актуальны'}
            </div>
          </div>
        </section>

        {error && (
          <Card className="border border-destructive/20 bg-destructive/5 ring-0">
            <CardContent className="flex items-start gap-3 text-destructive">
              <WalletCards aria-hidden="true" className="size-5" />
              <div><p className="font-semibold">Не все данные удалось обновить</p><p className="mt-1 text-muted-foreground">{error instanceof Error ? error.message : 'Повторите попытку чуть позже.'}</p></div>
            </CardContent>
          </Card>
        )}

        {accounts.data && <section aria-labelledby="accounts-title" className="space-y-4"><div><p className="text-sm font-semibold text-primary">Сейчас</p><h2 id="accounts-title" className="text-2xl font-semibold tracking-tight">Ваши счета</h2></div><Accounts items={accounts.data} /></section>}

        <section aria-labelledby="analytics-title" className="space-y-4"><div><p className="text-sm font-semibold text-primary">Понимание</p><h2 id="analytics-title" className="text-2xl font-semibold tracking-tight">Движение и доступность</h2></div><Analytics /></section>

        <section aria-labelledby="actions-title" className="space-y-4">
          <div><p className="text-sm font-semibold text-primary">Новое</p><h2 id="actions-title" className="text-2xl font-semibold tracking-tight">Добавить данные</h2><p className="mt-1 text-sm text-muted-foreground">Фиксируйте только подтверждённые события и планы.</p></div>
          <div className="grid items-start gap-6 xl:grid-cols-2">
            {accounts.data && categories.data && <CreateFinancialEventForm accounts={accounts.data} categories={categories.data} />}
            <CreateObligationForm />
          </div>
        </section>

        <section aria-labelledby="details-title" className="space-y-4">
          <div><p className="text-sm font-semibold text-primary">Подробности</p><h2 id="details-title" className="text-2xl font-semibold tracking-tight">Обязательства и события</h2></div>
          <div className="grid items-start gap-6 xl:grid-cols-2">
            {obligations.data && events.data && <Obligations items={obligations.data} events={events.data} />}
            {events.data && receipts.data && <Events items={events.data} receipts={receipts.data} />}
          </div>
        </section>
      </div>
    </AppLayout>
  )
}
