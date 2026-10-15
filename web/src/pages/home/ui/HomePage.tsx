import { ReminderHistory } from '@/features/payment-reminders/ReminderHistory'
import { LoanPanel } from '@/features/loan-details/LoanPanel'
import { ObligationSchedule } from '@/features/obligation-schedule/ui/ObligationSchedule'
import { Disclosure } from '@/shared/ui/disclosure'
import { PageHeader } from '@/shared/ui/page-header'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { Categories } from './Categories'
import { Accounts } from './Accounts'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { BarChart3, CalendarClock, WalletCards, Plus, RefreshCw } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { Overview } from './Overview'
import { Operations } from './Operations'
import { StatisticsPage } from '@/pages/statistics/StatisticsPage'
import { FinancialHealth } from './FinancialHealth'
import { CreateAccountForm } from '@/features/create-account/ui/CreateAccountForm'
import { today, formatMoney, dueSoonText } from '@/shared/lib/finance-display'
import { Money } from '@/shared/ui/money'
import { StatusBadge } from '@/shared/ui/status'
import { ProfilePanel } from '@/features/user-profile/ui/ProfilePanel'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { useState } from 'react'

import { CreateFinancialEventForm } from '@/features/create-financial-event'
import { CreateObligationForm } from '@/features/create-obligation'
import { apiClient } from '@/shared/api/client'
import type { CashPosition, FinancialEvent, Obligation, PeriodSummary } from '@/shared/api/types'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { AppLayout } from '@/shared/ui/layout/AppLayout'

function localDateValue(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function localBoundary(value: string) {
  return `${value}T00:00:00+05:00`
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

async function loadEvents(categoryId = '', from = '', to = '') {
  const { data, error } = await apiClient.GET('/api/v1/financial-events', {
    params: {
      query: { categoryId: categoryId || undefined, from: from || undefined, to: to || undefined },
    },
  })
  if (!data) throw new Error(error?.message ?? 'Не удалось загрузить финансовые события')
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

const obligationKindLabels: Record<Obligation['kind'], string> = {
  loan: 'Кредит',
  installment: 'Рассрочка',
  other_fixed: 'Другое',
}

function Obligations({ items, events }: { items: Obligation[]; events: FinancialEvent[] }) {
  const usedEventIDs = new Set(
    items.flatMap((item) =>
      item.payments.flatMap((payment) =>
        payment.settlements.map((settlement) => settlement.financialEventId),
      ),
    ),
  )
  const remainingByCurrency = items.reduce<Record<string, number>>((totals, item) => {
    totals[item.currency] = (totals[item.currency] ?? 0) + item.remainingMinor
    return totals
  }, {})

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
          <p className="text-sm text-muted-foreground">Обязательств пока нет.</p>
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
                {item.status === 'completed' ? (
                  <StatusBadge tone="success">Завершено</StatusBadge>
                ) : (
                  <p className="text-right text-sm text-muted-foreground">
                    Осталось <Money valueMinor={item.remainingMinor} currency={item.currency} />
                  </p>
                )}
              </div>
              {item.kind === 'loan' && <LoanPanel obligationId={item.id} />}
              <ObligationSchedule obligation={item} events={events} usedEventIDs={usedEventIDs} />
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
      const { data, error } = await apiClient.GET('/api/v1/analytics/period-summary', {
        params: {
          query: { from: localBoundary(fromDate), to: localBoundary(toDate) },
        },
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
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="size-5" /> Сводка периода
          </CardTitle>
          <CardDescription>Фактический поток денег; конец периода не включается.</CardDescription>
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
          {period.error && <p className="text-sm text-destructive">{period.error.message}</p>}
          {period.isPending && validPeriod && (
            <p className="text-sm text-muted-foreground">Рассчитываю…</p>
          )}
          {period.data?.currencies.length === 0 && (
            <p className="text-sm text-muted-foreground">За период нет финансовых данных.</p>
          )}
          {period.data?.currencies.map((item) => (
            <article key={item.currency} className="space-y-3 rounded-lg border p-4">
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
                <span className="text-right">{formatMoney(item.incomeMinor, item.currency)}</span>
                <span className="text-muted-foreground">Покупки</span>
                <span className="text-right">
                  {formatMoney(item.purchasesMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Возвраты</span>
                <span className="text-right">{formatMoney(item.refundsMinor, item.currency)}</span>
                <span className="text-muted-foreground">Платежи по обязательствам</span>
                <span className="text-right">
                  {formatMoney(item.debtPaymentsMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Внешний денежный поток</span>
                <span className="text-right font-medium">
                  {formatMoney(item.netExternalCashFlowMinor, item.currency)}
                </span>
                <span className="text-muted-foreground">Внутренние переводы, справочно</span>
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
                        <span className="text-muted-foreground">{category.categoryName}</span>
                        <span>{formatMoney(category.netExpenseMinor, item.currency)}</span>
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
            <Label htmlFor="cash-through">Обязательства по дату включительно</Label>
            <Input
              id="cash-through"
              type="date"
              value={through}
              onChange={(event) => setThrough(event.target.value)}
            />
          </div>
          {cash.error && <p className="text-sm text-destructive">{cash.error.message}</p>}
          {cash.isPending && <p className="text-sm text-muted-foreground">Рассчитываю…</p>}
          {cash.data?.currencies.length === 0 && (
            <p className="text-sm text-muted-foreground">
              Нет ликвидных счетов и обязательств до горизонта.
            </p>
          )}
          {cash.data?.currencies.map((item) => (
            <article key={item.currency} className="space-y-3 rounded-lg border p-4">
              <div className="flex items-center justify-between">
                <h3 className="font-semibold">{item.currency}</h3>
                <Badge variant="outline">
                  Доступно {formatMoney(item.availableAfterObligationsMinor, item.currency)}
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
                    <div key={payment.scheduledPaymentId} className="flex justify-between gap-3">
                      <span className="text-muted-foreground">
                        {payment.obligationName} · {payment.dueOn}
                        {payment.timing === 'overdue'
                          ? ' · просрочен'
                          : payment.timing === 'due_soon'
                            ? ` · ${dueSoonText(payment.daysUntilDue).toLowerCase()}`
                            : ''}
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
            Это арифметическая позиция по подтверждённым данным, а не бюджет, прогноз или
            рекомендация.
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
    queryKey: ['financial-events', '', '', ''],
    queryFn: () => loadEvents(),
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
      const { data, error } = await apiClient.GET('/api/v1/analytics/cash-position', {
        params: { query: { through } },
      })
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
    '/analytics': 'Статистика',
    '/health': 'Финансовое здоровье',
    '/profile': 'Профиль и доступ',
    '/categories': 'Категории',
  }
  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <PageHeader
            title={title[path] ?? 'Обзор'}
            description={
              {
                '/': 'Деньги и ближайшие дела — в одном месте.',
                '/operations': 'Покупки, поступления и переводы.',
                '/accounts': 'Счета, карты и подтверждённые остатки.',
                '/payments': 'Обязательства и ближайшие платежи.',
                '/analytics': 'Посмотрите, куда уходят деньги и как меняются траты.',
                '/health': 'Ближайшие обязательства, доступные деньги и качество данных.',
                '/profile': 'Личные данные, почта и управление доступом.',
              }[path]
            }
            status={error ? 'Часть данных недоступна' : pending ? 'Обновляем данные…' : undefined}
          />
          <div className="flex flex-wrap gap-3">
            <Dialog open={accountOpen} onOpenChange={setAccountOpen}>
              {/* SPEC-0036: only where money lives, never on profile or settings pages. */}
              {(path === '/accounts' ||
                (empty && ['/', '/operations', '/payments'].includes(path))) && (
                <DialogTrigger asChild>
                  <Button>
                    <Plus className="size-4" />
                    Добавить {empty ? 'первый ' : ''}счёт
                  </Button>
                </DialogTrigger>
              )}
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Новый счёт</DialogTitle>
                  <DialogDescription>Название, реквизиты и текущий остаток</DialogDescription>
                </DialogHeader>
                <CreateAccountForm onSaved={() => setAccountOpen(false)} />
              </DialogContent>
            </Dialog>
            {!empty && (
              <Dialog open={operationOpen} onOpenChange={setOperationOpen}>
                <DialogTrigger asChild>
                  <Button
                    className={!['/', '/operations'].includes(path) ? 'hidden' : undefined}
                    disabled={!accounts.data?.length || categories.isPending || categories.isError}
                  >
                    <Plus className="size-4" />
                    Добавить операцию
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Добавить операцию</DialogTitle>
                    <DialogDescription>Проверьте данные перед подтверждением</DialogDescription>
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
              <p className="mt-1 text-sm text-muted-foreground">{error.message}</p>
            </div>
            <Button variant="outline" onClick={() => void queryClient.invalidateQueries()}>
              <RefreshCw className="size-4" />
              Повторить
            </Button>
          </div>
        )}
        {path === '/categories' &&
          (categories.isPending ? (
            <LoadingSkeleton label="Загружаем категории…" />
          ) : (
            categories.data && <Categories items={categories.data} />
          ))}
        {pending && !['/profile', '/categories'].includes(path) ? (
          <LoadingSkeleton />
        ) : empty && !['/profile', '/categories'].includes(path) ? (
          <section className="rounded-2xl border bg-card px-5 py-12 text-center">
            <WalletCards aria-hidden="true" className="mx-auto mb-5 size-10 text-primary" />
            <h2>Начните с первого счёта</h2>
            <p className="mt-3 text-muted-foreground">
              Добавьте дебетовую карту или наличные и текущий остаток.
            </p>
            <div className="grace-actions justify-center">
              <Button onClick={() => setAccountOpen(true)}>Добавить первый счёт</Button>
            </div>
          </section>
        ) : (
          <>
            {path === '/' && (
              <div className="space-y-6">
                <Overview
                  cash={cash.data}
                  pending={cash.isPending}
                  failed={cash.isError}
                  events={events.data ?? []}
                  accounts={accounts.data ?? []}
                  retry={() => void cash.refetch()}
                />
                <StatisticsPage compact />
              </div>
            )}
            {path === '/accounts' && accounts.data && <Accounts items={accounts.data} />}
            {path === '/operations' && (
              <Operations
                categories={categories.data ?? []}
                accounts={accounts.data ?? []}
                receipts={receipts.data ?? []}
              />
            )}
            {path === '/payments' && (
              <div className="space-y-6">
                <ReminderHistory />
                <Disclosure
                  className="rounded-2xl border bg-card p-5"
                  triggerClassName="min-h-11 rounded-xl px-3 py-2 hover:bg-accent"
                  title={<>Добавить обязательство</>}
                >
                  <div className="mt-5">
                    <CreateObligationForm />
                  </div>
                </Disclosure>
                {obligations.data && (
                  <Obligations items={obligations.data} events={events.data ?? []} />
                )}
              </div>
            )}
            {path === '/analytics' && (
              <div className="space-y-8">
                <StatisticsPage />
                <Disclosure
                  className="rounded-2xl border bg-card p-5"
                  triggerClassName="min-h-11 content-center font-medium"
                  title={<>Подробные расчёты остатков и платежей</>}
                >
                  <div className="mt-5">
                    <Analytics />
                  </div>
                </Disclosure>
              </div>
            )}
            {path === '/health' && (
              <FinancialHealth
                incomplete={Boolean(error)}
                cash={cash.isError ? undefined : cash.data}
                accounts={accounts.data ?? []}
                categories={categories.data ?? []}
                events={events.data ?? []}
                obligations={obligations.data ?? []}
                receipts={receipts.data ?? []}
              />
            )}
            {path === '/profile' && <ProfilePanel />}
          </>
        )}
      </div>
    </AppLayout>
  )
}
