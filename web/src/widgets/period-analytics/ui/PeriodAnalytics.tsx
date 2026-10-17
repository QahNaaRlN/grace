import { useQuery } from '@tanstack/react-query'
import { BarChart3, WalletCards } from 'lucide-react'
import { Fragment, useState } from 'react'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { browserMonthEnd, browserMonthPeriod } from '@/shared/lib/date'
import { dueSoonText } from '@/shared/lib/finance-display'
import { formatMoney } from '@/shared/lib/money'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'

export function PeriodAnalytics() {
  const defaults = browserMonthPeriod()
  const [fromDate, setFromDate] = useState(defaults.from)
  const [toDate, setToDate] = useState(defaults.to)
  const [through, setThrough] = useState(browserMonthEnd)
  const validPeriod = fromDate !== '' && toDate !== '' && fromDate < toDate
  const period = useQuery({
    ...periodSummaryQueries.period(fromDate, toDate),
    enabled: validPeriod,
  })
  const cash = useQuery({ ...cashPositionQueries.through(through), enabled: through !== '' })

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
                {item.debtPaymentsMinor > 0 &&
                  (
                    [
                      ['основной долг', item.debtPrincipalMinor],
                      ['проценты', item.debtInterestMinor],
                      ['комиссии и прочее', item.debtOtherMinor],
                      ['не распределено', item.debtUnallocatedMinor],
                    ] as const
                  )
                    .filter(([, value]) => value > 0)
                    .map(([label, value]) => (
                      <Fragment key={label}>
                        <span className="pl-3 text-muted-foreground">в т. ч. {label}</span>
                        <span className="text-right">{formatMoney(value, item.currency)}</span>
                      </Fragment>
                    ))}
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
