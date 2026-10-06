import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowDownRight,
  ArrowRightLeft,
  BarChart3,
  CalendarClock,
  Link2,
  WalletCards,
  Plus,
  RefreshCw,
} from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { Overview } from './Overview'
import { FinancialHealth } from './FinancialHealth'
import { CreateAccountForm } from '@/features/create-account/ui/CreateAccountForm'
import { accountLabel, today, dateText } from '@/shared/lib/finance-display'
import { AuthActions } from '@/shared/ui/layout/AuthActions'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
  DialogClose,
} from '@/shared/ui/dialog'
import { Fragment, useRef, useState } from 'react'
import { toast } from 'sonner'

import { CreateFinancialEventForm } from '@/features/create-financial-event'
import { CreateObligationForm } from '@/features/create-obligation'
import { CreateReceiptForm } from '@/features/create-receipt'
import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import type {
  Account,
  CashPosition,
  FinancialEvent,
  Obligation,
  PeriodSummary,
  Receipt,
  ScheduledPayment,
} from '@/shared/api/types'
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
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency }).format(
    valueMinor / 100,
  )
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
  return localDateValue(new Date(now.getFullYear(), now.getMonth() + 1, 0))
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
  const [selected, setSelected] = useState<Account | null>(null)
  const opener = useRef<HTMLButtonElement | null>(null)
  return (
    <>
      <div className="rounded-2xl border bg-card p-5">
        {items.map((account) => (
          <button
            key={account.id}
            className="grace-row mt-1"
            onClick={(event) => {
              opener.current = event.currentTarget
              setSelected(account)
            }}
          >
            <WalletCards
              aria-hidden="true"
              className="size-5 text-muted-foreground"
            />
            <span className="min-w-0 flex-1">
              <strong className="block break-words">
                {accountLabel(account)}
              </strong>
              <small className="mt-1 block text-muted-foreground">
                {account.type === 'cash'
                  ? 'Наличные'
                  : account.type === 'credit'
                    ? 'Кредитный счёт'
                    : account.type === 'debit'
                      ? 'Дебетовый счёт'
                      : 'Рассрочка'}
              </small>
            </span>
            <strong className="grace-money max-w-[48%] text-right">
              {formatMoney(account.balanceMinor, account.currency)}
            </strong>
          </button>
        ))}
      </div>
      <Dialog
        open={!!selected}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
      >
        <DialogContent
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            opener.current?.focus()
          }}
        >
          <DialogHeader>
            <DialogTitle>{selected?.name}</DialogTitle>
            <DialogDescription>
              Сведения о счёте и его платёжных инструментах
            </DialogDescription>
          </DialogHeader>
          {selected && (
            <>
              <dl className="grace-fields">
                <div>
                  <dt>Название счёта</dt>
                  <dd>{selected.name}</dd>
                </div>
                <div>
                  <dt>Текущий баланс</dt>
                  <dd>
                    {formatMoney(selected.balanceMinor, selected.currency)}
                  </dd>
                </div>
                {selected.paymentInstruments?.map((card) => (
                  <Fragment key={card.id}>
                    <div>
                      <dt>Название карты</dt>
                      <dd>{card.name}</dd>
                    </div>
                    <div>
                      <dt>Банк</dt>
                      <dd>{card.bank}</dd>
                    </div>
                    <div>
                      <dt>Последние четыре цифры</dt>
                      <dd>{card.last4}</dd>
                    </div>
                  </Fragment>
                ))}
                {!selected.paymentInstruments?.length && (
                  <div>
                    <dt>Карта / банк / номер</dt>
                    <dd>
                      {selected.type === 'cash'
                        ? 'Не применяется'
                        : 'Не указано'}
                    </dd>
                  </div>
                )}
              </dl>
              {selected.type === 'credit' && (
                <details>
                  <summary className="min-h-11 rounded-xl px-3 py-2 hover:bg-accent">
                    Кредитные условия и выписка
                  </summary>
                  <dl className="grace-fields">
                    {[
                      'Лимит',
                      'Доступно',
                      'Начало текущего периода',
                      'Конец текущего периода',
                      'Дата платежа',
                      'Нужно внести',
                      'Минимальный платёж',
                      'Покупки периода',
                      'Для сохранения grace',
                      'Оплачено',
                      'Осталось внести',
                      'Статус',
                      'Комментарий',
                    ].map((label) => (
                      <div key={label}>
                        <dt>{label}</dt>
                        <dd>Не указано</dd>
                      </div>
                    ))}
                  </dl>
                </details>
              )}
              <p className="text-sm text-muted-foreground">
                Баланс хранится на счёте. Карта не создаёт отдельный остаток.
              </p>
            </>
          )}
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Закрыть</Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
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
            <span>
              {item.position}. {item.name} · {item.quantityMilli / 1000}
            </span>
            <span className="shrink-0">
              {formatMoney(item.lineTotalMinor, receipt.currency)}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function Events({
  items,
  receipts,
  accounts,
}: {
  items: FinancialEvent[]
  receipts: Receipt[]
  accounts: Account[]
}) {
  const receiptsByPurchase = new Map(
    receipts.map((receipt) => [receipt.purchaseEventId, receipt]),
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle>Операции</CardTitle>
        <CardDescription>
          Подтверждённые операции и связанные чеки.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">Событий пока нет.</p>
        ) : (
          <div className="space-y-2">
            {items.map((item) => {
              const posting = item.postings[0]
              const isTransfer = item.kind === 'internal_transfer'
              const positive = posting?.direction === 'credit'
              const sourcePosting = item.postings.find(
                (value) => value.direction === 'debit',
              )
              const destinationPosting = item.postings.find(
                (value) => value.direction === 'credit',
              )
              const label = (
                id: string | undefined,
                fallback: string | undefined,
              ) => {
                const a = accounts.find((a) => a.id === id)
                return a ? accountLabel(a) : (fallback ?? 'Не указан счёт')
              }
              const accountDescription = isTransfer
                ? `${label(sourcePosting?.accountId, sourcePosting?.accountName)} → ${label(destinationPosting?.accountId, destinationPosting?.accountName)}`
                : label(posting?.accountId, posting?.accountName)
              const receipt = receiptsByPurchase.get(item.id)
              return (
                <article key={item.id} className="rounded-xl border p-3 sm:p-4">
                  <div className="flex flex-wrap items-center justify-between gap-4">
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="rounded-full bg-muted p-2">
                        {isTransfer ? (
                          <ArrowRightLeft
                            aria-hidden="true"
                            className="size-4"
                          />
                        ) : (
                          <ArrowDownRight
                            aria-hidden="true"
                            className="size-4"
                          />
                        )}
                      </div>
                      <div className="min-w-0">
                        <p className="break-words font-medium">
                          {item.merchant ||
                            item.description ||
                            kindLabels[item.kind]}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {dateText(item.occurredAt)} ·{' '}
                          {item.categoryName ?? kindLabels[item.kind]} ·{' '}
                          {accountDescription}
                        </p>
                      </div>
                    </div>
                    {posting && (
                      <div className="min-w-0 space-y-1 text-right">
                        <p
                          className={
                            positive && !isTransfer
                              ? 'grace-money font-medium text-success'
                              : 'grace-money font-medium'
                          }
                        >
                          {isTransfer ? '' : positive ? '+' : '−'}
                          {formatMoney(posting.amountMinor, posting.currency)}
                        </p>
                        <Badge variant="secondary">
                          {kindLabels[item.kind]}
                        </Badge>
                      </div>
                    )}
                  </div>
                  <details className="mt-4">
                    <summary className="min-h-11 rounded-xl px-3 py-2 hover:bg-accent">
                      Все сведения операции
                    </summary>
                    <dl className="grace-fields">
                      {[
                        [
                          'Дата',
                          new Date(item.occurredAt).toLocaleString('ru-RU'),
                        ],
                        ['Тип', kindLabels[item.kind]],
                        ['Описание', item.description || 'Не указано'],
                        ['Категория', item.categoryName || 'Не указано'],
                        ['Подкатегория', 'Не указано'],
                        [
                          'Сумма',
                          posting
                            ? formatMoney(posting.amountMinor, posting.currency)
                            : 'Не указано',
                        ],
                        ['Источник финансирования', 'Не указано'],
                        ['Счёт / карта', accountDescription],
                        ['Приоритет', 'Не указано'],
                        ['Регулярный?', 'Не указано'],
                        ['Получатель / долг', item.merchant || 'Не указано'],
                        ['Комментарий', item.description || 'Не указано'],
                        [
                          'Месяц',
                          new Intl.DateTimeFormat('ru-RU', {
                            timeZone: 'Asia/Yekaterinburg',
                            month: 'long',
                            year: 'numeric',
                          }).format(new Date(item.occurredAt)),
                        ],
                        ['Источник записи', item.source],
                        ['Проверка источника', 'Не указано'],
                        ['ID', item.id],
                      ].map(([label, value]) => (
                        <div key={label}>
                          <dt>{label}</dt>
                          <dd>{value}</dd>
                        </div>
                      ))}
                    </dl>
                  </details>
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
    return (
      event.kind === 'debt_payment' &&
      event.postings.length === 1 &&
      !usedEventIDs.has(event.id) &&
      posting?.currency === obligation.currency &&
      posting.direction === 'debit' &&
      posting.amountMinor <= payment.remainingMinor
    )
  })

  if (payment.remainingMinor === 0) return null
  if (candidates.length === 0) {
    return (
      <p className="mt-2 text-xs text-muted-foreground">
        Сначала создайте подходящий платёж по обязательству.
      </p>
    )
  }

  async function settle() {
    if (!eventID) return
    setIsSubmitting(true)
    const result = await apiClient.POST(
      '/api/v1/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements',
      {
        params: {
          path: { obligationId: obligation.id, paymentId: payment.id },
          header: {
            'Idempotency-Key': createIdempotencyKey(),
            'X-CSRF-Token': csrfToken,
          },
        },
        body: { financialEventId: eventID },
      },
    )
    setIsSubmitting(false)
    if (result.error) {
      toast.error(result.error.message)
      return
    }
    setEventID('')
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['obligations'] }),
      queryClient.invalidateQueries({ queryKey: ['cash-position'] }),
    ])
    toast.success('Фактический платёж связан с графиком')
  }

  return (
    <div className="mt-2 flex flex-wrap gap-2">
      <Select value={eventID} onValueChange={setEventID}>
        <SelectTrigger className="min-w-0 flex-1">
          <SelectValue placeholder="Выберите фактический платёж" />
        </SelectTrigger>
        <SelectContent>
          {candidates.map((event) => (
            <SelectItem key={event.id} value={event.id}>
              {event.merchant || event.description || 'Платёж'} ·{' '}
              {formatMoney(
                event.postings[0].amountMinor,
                event.postings[0].currency,
              )}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button
        className="w-full sm:w-auto"
        type="button"
        variant="outline"
        disabled={!eventID || isSubmitting}
        onClick={settle}
      >
        <Link2 aria-hidden="true" className="mr-1 size-4" />
        {isSubmitting ? 'Связываю…' : 'Связать'}
      </Button>
    </div>
  )
}

function Obligations({
  items,
  events,
}: {
  items: Obligation[]
  events: FinancialEvent[]
}) {
  const usedEventIDs = new Set(
    items.flatMap((item) =>
      item.payments.flatMap((payment) =>
        payment.settlements.map((settlement) => settlement.financialEventId),
      ),
    ),
  )
  const remainingByCurrency = items.reduce<Record<string, number>>(
    (totals, item) => {
      totals[item.currency] = (totals[item.currency] ?? 0) + item.remainingMinor
      return totals
    },
    {},
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CalendarClock className="size-5" /> Обязательства
        </CardTitle>
        <CardDescription>
          Осталось:{' '}
          {Object.entries(remainingByCurrency)
            .map(([currency, amount]) => formatMoney(amount, currency))
            .join(' · ') || 'нет обязательств'}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Обязательств пока нет.
          </p>
        ) : (
          items.map((item) => (
            <article key={item.id} className="rounded-lg border p-4">
              <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 className="font-semibold">{item.name}</h3>
                  <p className="text-sm text-muted-foreground">
                    {obligationKindLabels[item.kind]} · оплачено{' '}
                    {formatMoney(item.paidMinor, item.currency)} из{' '}
                    {formatMoney(item.totalMinor, item.currency)}
                  </p>
                </div>
                <Badge
                  variant={
                    item.status === 'completed' ? 'secondary' : 'outline'
                  }
                >
                  {item.status === 'completed'
                    ? 'Завершено'
                    : `Осталось ${formatMoney(item.remainingMinor, item.currency)}`}
                </Badge>
              </div>
              <div className="space-y-3">
                {item.payments.map((payment) => (
                  <div key={payment.id} className="rounded-md bg-muted/50 p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                      <div>
                        <span className="font-medium">
                          {new Date(
                            `${payment.dueOn}T00:00:00Z`,
                          ).toLocaleDateString('ru-RU', { timeZone: 'UTC' })}
                        </span>
                        <span className="text-muted-foreground">
                          {' '}
                          · осталось{' '}
                          {formatMoney(payment.remainingMinor, item.currency)}
                        </span>
                      </div>
                      <Badge
                        variant={
                          payment.isOverdue ? 'destructive' : 'secondary'
                        }
                      >
                        {paymentStatusLabels[payment.status]}
                        {payment.isOverdue &&
                        payment.status === 'partially_paid'
                          ? ' · просрочен'
                          : ''}
                      </Badge>
                    </div>
                    <SettlementControl
                      obligation={item}
                      payment={payment}
                      events={events}
                      usedEventIDs={usedEventIDs}
                    />
                  </div>
                ))}
              </div>
            </article>
          ))
        )}
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
      const { data, error } = await apiClient.GET(
        '/api/v1/analytics/period-summary',
        {
          params: {
            query: { from: localBoundary(fromDate), to: localBoundary(toDate) },
          },
        },
      )
      if (!data)
        throw new Error(error?.message ?? 'Не удалось загрузить сводку периода')
      return data
    },
  })
  const cash = useQuery<CashPosition>({
    queryKey: ['cash-position', through],
    enabled: through !== '',
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/analytics/cash-position',
        {
          params: { query: { through } },
        },
      )
      if (!data)
        throw new Error(
          error?.message ?? 'Не удалось загрузить доступность денег',
        )
      return data
    },
  })

  return (
    <div className="grid gap-6 xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="size-5" /> Сводка периода
          </CardTitle>
          <CardDescription>
            Фактический поток денег; конец периода не включается.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="summary-from">С даты</Label>
              <Input
                id="summary-from"
                type="date"
                value={fromDate}
                onChange={(event) => setFromDate(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="summary-to">До даты, не включая</Label>
              <Input
                id="summary-to"
                type="date"
                value={toDate}
                onChange={(event) => setToDate(event.target.value)}
              />
            </div>
          </div>
          {!validPeriod && (
            <p className="text-sm text-destructive">
              Дата окончания должна быть позже даты начала.
            </p>
          )}
          {period.error && (
            <p className="text-sm text-destructive">{period.error.message}</p>
          )}
          {period.isPending && validPeriod && (
            <p className="text-sm text-muted-foreground">Рассчитываю…</p>
          )}
          {period.data?.currencies.length === 0 && (
            <p className="text-sm text-muted-foreground">
              За период нет финансовых данных.
            </p>
          )}
          {period.data?.currencies.map((item) => (
            <article
              key={item.currency}
              className="space-y-3 rounded-lg border p-4"
            >
              <div className="flex items-center justify-between">
                <h3 className="font-semibold">{item.currency}</h3>
                <Badge variant="outline">
                  Итог {formatMoney(item.closingBalanceMinor, item.currency)}
                </Badge>
              </div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <span className="text-muted-foreground">Начальный остаток</span>
                <span className="text-right">
                  {formatMoney(item.openingBalanceMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Доходы</span>
                <span className="text-right">
                  {formatMoney(item.incomeMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Покупки</span>
                <span className="text-right">
                  {formatMoney(item.purchasesMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Возвраты</span>
                <span className="text-right">
                  {formatMoney(item.refundsMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">
                  Платежи по обязательствам
                </span>
                <span className="text-right">
                  {formatMoney(item.debtPaymentsMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">
                  Внешний денежный поток
                </span>
                <span className="text-right font-medium">
                  {formatMoney(item.netExternalCashFlowMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">
                  Внутренние переводы, справочно
                </span>
                <span className="text-right">
                  {formatMoney(item.internalTransferVolumeMinor, item.currency)}
                </span>
              </div>
              {item.expenseCategories.length > 0 && (
                <div className="border-t pt-3">
                  <p className="mb-2 text-sm font-medium">Категории покупок</p>
                  <div className="space-y-1 text-sm">
                    {item.expenseCategories.map((category) => (
                      <div
                        key={category.categoryId ?? category.categoryName}
                        className="flex justify-between gap-3"
                      >
                        <span className="text-muted-foreground">
                          {category.categoryName}
                        </span>
                        <span>
                          {formatMoney(category.netExpenseMinor, item.currency)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </article>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <WalletCards className="size-5" /> Доступно после обязательств
          </CardTitle>
          <CardDescription>
            Текущие ликвидные счета минус известные платежи до выбранной даты.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cash-through">
              Обязательства по дату включительно
            </Label>
            <Input
              id="cash-through"
              type="date"
              value={through}
              onChange={(event) => setThrough(event.target.value)}
            />
          </div>
          {cash.error && (
            <p className="text-sm text-destructive">{cash.error.message}</p>
          )}
          {cash.isPending && (
            <p className="text-sm text-muted-foreground">Рассчитываю…</p>
          )}
          {cash.data?.currencies.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Нет ликвидных счетов и обязательств до горизонта.
            </p>
          )}
          {cash.data?.currencies.map((item) => (
            <article
              key={item.currency}
              className="space-y-3 rounded-lg border p-4"
            >
              <div className="flex items-center justify-between">
                <h3 className="font-semibold">{item.currency}</h3>
                <Badge variant="outline">
                  Доступно{' '}
                  {formatMoney(
                    item.availableAfterObligationsMinor,
                    item.currency,
                  )}
                </Badge>
              </div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <span className="text-muted-foreground">Ликвидный остаток</span>
                <span className="text-right">
                  {formatMoney(item.liquidBalanceMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Просрочено</span>
                <span className="text-right">
                  {formatMoney(item.overdueObligationsMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Предстоит</span>
                <span className="text-right">
                  {formatMoney(item.upcomingObligationsMinor, item.currency)}
                </span>
                <span className="font-medium">Всего обязательств</span>
                <span className="text-right font-medium">
                  {formatMoney(item.obligationsThroughMinor, item.currency)}
                </span>
              </div>
              {item.scheduledPayments.length > 0 && (
                <div className="space-y-1 border-t pt-3 text-sm">
                  {item.scheduledPayments.map((payment) => (
                    <div
                      key={payment.scheduledPaymentId}
                      className="flex justify-between gap-3"
                    >
                      <span className="text-muted-foreground">
                        {payment.obligationName} · {payment.dueOn}
                        {payment.timing === 'overdue' ? ' · просрочен' : ''}
                      </span>
                      <span className="shrink-0">
                        {formatMoney(payment.remainingMinor, item.currency)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </article>
          ))}
          <p className="text-xs text-muted-foreground">
            Это арифметическая позиция по подтверждённым данным, а не бюджет,
            прогноз или рекомендация.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

export function HomePage() {
  const path = useLocation().pathname
  const queryClient = useQueryClient()
  const [operationOpen, setOperationOpen] = useState(false)
  const [accountOpen, setAccountOpen] = useState(false)
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: loadAccounts })
  const categories = useQuery({
    queryKey: ['categories'],
    queryFn: loadCategories,
  })
  const events = useQuery({
    queryKey: ['financial-events'],
    queryFn: loadEvents,
  })
  const receipts = useQuery({ queryKey: ['receipts'], queryFn: loadReceipts })
  const asOf = today()
  const obligations = useQuery({
    queryKey: ['obligations', asOf],
    queryFn: () => loadObligations(asOf),
  })
  const through = initialThrough()
  const cash = useQuery({
    queryKey: ['cash-position', through],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/analytics/cash-position',
        { params: { query: { through } } },
      )
      if (!data) throw new Error(error?.message ?? 'Не удалось получить расчёт')
      return data
    },
  })
  const queries = [accounts, categories, events, receipts, obligations, cash]
  const error = queries.find((q) => q.isError)?.error
  const pending = queries.some((q) => q.isPending)
  const empty = accounts.isSuccess && accounts.data.length === 0
  const title: Record<string, string> = {
    '/': 'Обзор',
    '/operations': 'Операции',
    '/payments': 'Платежи',
    '/accounts': 'Счета и карты',
    '/analytics': 'Подробный расчёт',
    '/health': 'Финансовое здоровье',
    '/profile': 'Профиль и доступ',
  }
  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="mb-1 text-sm text-muted-foreground">Личные финансы</p>
            <h1>{title[path] ?? 'Обзор'}</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              {error
                ? 'Часть данных недоступна'
                : pending
                  ? 'Обновляем данные…'
                  : 'Данные загружены'}
            </p>
          </div>
          <div className="flex flex-wrap gap-3">
            <Dialog open={accountOpen} onOpenChange={setAccountOpen}>
              <DialogTrigger asChild>
                <Button variant={empty ? 'default' : 'outline'}>
                  <Plus className="size-4" />
                  Добавить {empty ? 'первый ' : ''}счёт
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Новый счёт</DialogTitle>
                  <DialogDescription>
                    Название, реквизиты и текущий остаток
                  </DialogDescription>
                </DialogHeader>
                <CreateAccountForm onSaved={() => setAccountOpen(false)} />
              </DialogContent>
            </Dialog>
            {!empty && (
              <Dialog open={operationOpen} onOpenChange={setOperationOpen}>
                <DialogTrigger asChild>
                  <Button
                    disabled={
                      !accounts.data?.length ||
                      categories.isPending ||
                      categories.isError
                    }
                  >
                    <Plus className="size-4" />
                    Добавить операцию
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Добавить операцию</DialogTitle>
                    <DialogDescription>
                      Проверьте данные перед подтверждением
                    </DialogDescription>
                  </DialogHeader>
                  {accounts.data && categories.data && (
                    <CreateFinancialEventForm
                      accounts={accounts.data}
                      categories={categories.data}
                      events={events.data ?? []}
                      onSaved={() => setOperationOpen(false)}
                    />
                  )}
                </DialogContent>
              </Dialog>
            )}
          </div>
        </div>
        {error && (
          <div
            role="alert"
            className="flex flex-wrap items-center gap-4 rounded-xl border bg-card p-4"
          >
            <div className="min-w-0 flex-1">
              <strong>Обзор загружен частично</strong>
              <p className="mt-1 text-sm text-muted-foreground">
                {error.message}
              </p>
            </div>
            <Button
              variant="outline"
              onClick={() => void queryClient.invalidateQueries()}
            >
              <RefreshCw className="size-4" />
              Повторить
            </Button>
          </div>
        )}
        {empty && path !== '/profile' ? (
          <section className="rounded-2xl border bg-card px-5 py-12 text-center">
            <WalletCards
              aria-hidden="true"
              className="mx-auto mb-5 size-10 text-primary"
            />
            <h2>Начните с первого счёта</h2>
            <p className="mt-3 text-muted-foreground">
              Добавьте дебетовую карту или наличные и текущий остаток.
            </p>
            <div className="grace-actions justify-center">
              <Button onClick={() => setAccountOpen(true)}>
                Добавить первый счёт
              </Button>
            </div>
          </section>
        ) : (
          <>
            {path === '/' && (
              <Overview
                cash={cash.data}
                pending={cash.isPending}
                failed={cash.isError}
                events={events.data ?? []}
                accounts={accounts.data ?? []}
                retry={() => void cash.refetch()}
              />
            )}
            {path === '/accounts' && accounts.data && (
              <Accounts items={accounts.data} />
            )}
            {path === '/operations' && events.data && (
              <Events
                items={events.data}
                receipts={receipts.data ?? []}
                accounts={accounts.data ?? []}
              />
            )}
            {path === '/payments' && (
              <div className="space-y-6">
                <details className="rounded-2xl border bg-card p-5">
                  <summary className="min-h-11 rounded-xl px-3 py-2 hover:bg-accent">
                    Добавить обязательство
                  </summary>
                  <div className="mt-5">
                    <CreateObligationForm />
                  </div>
                </details>
                {obligations.data && (
                  <Obligations
                    items={obligations.data}
                    events={events.data ?? []}
                  />
                )}
              </div>
            )}
            {path === '/analytics' && <Analytics />}
            {path === '/health' && (
              <FinancialHealth
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                events={events.data ?? []}
                obligations={obligations.data ?? []}
                receipts={receipts.data ?? []}
              />
            )}
            {path === '/profile' && (
              <section className="rounded-2xl border bg-card p-6">
                <h2>Ваше пространство</h2>
                <p className="mt-3 text-muted-foreground">
                  Управление приглашениями и сеансами
                </p>
                <div className="grace-actions">
                  <AuthActions />
                </div>
              </section>
            )}
          </>
        )}
      </div>
    </AppLayout>
  )
}
