import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Activity, ArrowDownRight, ArrowRightLeft, CalendarClock, Link2, WalletCards } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { CreateFinancialEventForm } from '@/features/create-financial-event'
import { CreateObligationForm } from '@/features/create-obligation'
import { CreateReceiptForm } from '@/features/create-receipt'
import { apiClient } from '@/shared/api/client'
import type { Account, FinancialEvent, Obligation, Receipt, ScheduledPayment } from '@/shared/api/types'
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
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {items.map((account) => (
        <Card key={account.id}>
          <CardHeader className="pb-2">
            <CardDescription>{account.type}</CardDescription>
            <CardTitle className="text-lg">{account.name}</CardTitle>
          </CardHeader>
          <CardContent className="text-2xl font-semibold">
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
      <Button type="button" variant="outline" disabled={!eventID || isSubmitting} onClick={settle}>
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
      <section className="space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-muted-foreground">Grace</p>
            <h1 className="text-3xl font-semibold tracking-tight">Финансовая картина</h1>
          </div>
          <Badge variant={error ? 'destructive' : 'secondary'}>
            <Activity aria-hidden="true" className="mr-1 size-3" />
            {error ? 'API недоступен' : isPending ? 'Загрузка…' : 'Данные актуальны'}
          </Badge>
        </div>

        {error && (
          <Card>
            <CardContent className="flex items-center gap-3 pt-6 text-destructive">
              <WalletCards aria-hidden="true" className="size-5" />
              <p>{error instanceof Error ? error.message : 'Не удалось связаться с API'}</p>
            </CardContent>
          </Card>
        )}

        {accounts.data && <Accounts items={accounts.data} />}
        {accounts.data && categories.data && (
          <CreateFinancialEventForm accounts={accounts.data} categories={categories.data} />
        )}
        <CreateObligationForm />
        {obligations.data && events.data && <Obligations items={obligations.data} events={events.data} />}
        {events.data && receipts.data && <Events items={events.data} receipts={receipts.data} />}
      </section>
    </AppLayout>
  )
}
