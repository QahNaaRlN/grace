import { statisticsQueries } from '@/entities/analytics-report'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import type { StatisticsCurrency } from '@/shared/api'
import { APP_TIME_ZONE } from '@/shared/config'
import { formatMoney } from '@/shared/lib/money'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { defaultStatisticsPeriod, dayText } from './statistics-format'
import { SeriesChart, CategoryChart } from './SeriesChart'

export function StatisticsCharts({
  item,
  compact = false,
}: {
  item: StatisticsCurrency
  compact?: boolean
}) {
  const amounts = [
    ...item.series.flatMap((b) => [
      b.incomeMinor,
      b.purchasesMinor,
      b.debtPaymentsMinor,
      b.refundsMinor,
      b.outflowsMinor,
    ]),
    ...item.categories.flatMap((c) => [c.purchasesMinor, c.refundsMinor, c.netExpenseMinor]),
  ]
  if (!amounts.every(Number.isSafeInteger))
    return <p role="alert">Суммы слишком велики для точного отображения. График не построен.</p>
  if (compact) return <SeriesChart item={item} compact />
  return (
    <div className="grid min-w-0 gap-6 xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Куда уходят деньги</CardTitle>
        </CardHeader>
        <CardContent>
          <CategoryChart item={item} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Динамика покупок</CardTitle>
        </CardHeader>
        <CardContent>
          <SeriesChart item={item} />
        </CardContent>
      </Card>
      {item.summary.debtPaymentsMinor > 0 && <DebtSplitCard item={item} />}
      <Card className="xl:col-span-2">
        <CardHeader>
          <CardTitle>Доходы и расходы</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Показаны покупки и платежи по обязательствам. Возвраты доступны в таблице; переводы
            между своими счетами исключены.
          </p>
          <SeriesChart item={item} comparison />
        </CardContent>
      </Card>
    </div>
  )
}

// SPEC-0053: what loan payments did — reduced principal or paid the bank for
// the credit. Payments without a known split stay visible as unallocated.
function DebtSplitCard({ item }: { item: StatisticsCurrency }) {
  const s = item.summary
  const rows: [string, number][] = [
    ['Основной долг', s.debtPrincipalMinor],
    ['Проценты', s.debtInterestMinor],
    ['Комиссии и прочее', s.debtOtherMinor],
    ['Не распределено', s.debtUnallocatedMinor],
  ]
  return (
    <Card className="xl:col-span-2">
      <CardHeader>
        <CardTitle>Платежи по кредитам</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">
          Проценты, комиссии и прочее — стоимость кредитов. Основной долг уменьшает задолженность и
          не считается расходом второй раз.
        </p>
        <dl className="grid grid-cols-2 gap-x-3 gap-y-1 text-sm sm:max-w-md">
          {rows.map(([label, value]) => (
            <div key={label} className="contents">
              <dt className="text-muted-foreground">{label}</dt>
              <dd className="text-right tabular-nums">{formatMoney(value, item.currency)}</dd>
            </div>
          ))}
        </dl>
      </CardContent>
    </Card>
  )
}

export function StatisticsPanel({ compact = false }: { compact?: boolean }) {
  const defaults = defaultStatisticsPeriod()
  const [from, setFrom] = useState(defaults.from)
  const [to, setTo] = useState(defaults.to)
  const [granularity, setGranularity] = useState<'month' | 'week'>('month')
  const [currency, setCurrency] = useState('RUB')
  const maxTo = new Date(from + 'T00:00:00Z')
  maxTo.setUTCFullYear(maxTo.getUTCFullYear() + 1)
  const valid =
    from !== '' &&
    to !== '' &&
    from < to &&
    Number.isFinite(maxTo.getTime()) &&
    to <= maxTo.toISOString().slice(0, 10)
  const query = useQuery({
    ...statisticsQueries.range(from, to, granularity),
    enabled: valid,
  })
  const item =
    query.data?.currencies.find((c) => c.currency === currency) ?? query.data?.currencies[0]
  const body = (
    <div className="min-w-0 space-y-5">
      {!compact && (
        <div className="grid gap-4 sm:grid-cols-4">
          <div className="space-y-2">
            <Label htmlFor="stats-from">С даты</Label>
            <Input
              id="stats-from"
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="stats-to">До даты, не включая</Label>
            <Input id="stats-to" type="date" value={to} onChange={(e) => setTo(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="stats-group">Группировка</Label>
            <ChoiceSelect
              id="stats-group"
              value={granularity}
              onValueChange={(value) => setGranularity(value as 'month' | 'week')}
              className="h-11 w-full rounded-xl border border-input bg-card px-3"
            >
              <option value="month">По месяцам</option>
              <option value="week">По неделям</option>
            </ChoiceSelect>
          </div>
          <div className="space-y-2">
            <Label htmlFor="stats-currency">Валюта</Label>
            <ChoiceSelect
              id="stats-currency"
              value={item?.currency ?? ''}
              onValueChange={(value) => setCurrency(value)}
              disabled={!query.data?.currencies.length}
              className="h-11 w-full rounded-xl border border-input bg-card px-3 disabled:cursor-not-allowed"
            >
              {query.data?.currencies.map((c) => (
                <option key={c.currency}>{c.currency}</option>
              ))}
            </ChoiceSelect>
          </div>
        </div>
      )}
      <p className="text-sm text-muted-foreground">
        Только записанные события · {APP_TIME_ZONE}. Полнота истории не подтверждена; ноль на
        графике означает отсутствие записей. Неполные крайние периоды показаны по выбранным датам.
      </p>
      {!valid ? (
        <p role="alert">Выберите период до одного года; конец должен быть позже начала.</p>
      ) : query.isError ? (
        <div role="alert" className="space-y-3">
          <p>{query.error.message}</p>
          <Button variant="outline" onClick={() => void query.refetch()}>
            Повторить загрузку
          </Button>
        </div>
      ) : query.isPending ? (
        <LoadingSkeleton label="Загружаем статистику…" />
      ) : item ? (
        <>
          <p className="text-sm">
            {dayText(from)} — до {dayText(to)}, не включая · {item.currency} · Покупки{' '}
            {formatMoney(item.summary.purchasesMinor, item.currency)}
          </p>
          {compact && query.data.currencies.length > 1 && (
            <div className="space-y-2">
              <Label htmlFor="overview-stats-currency">Валюта графика</Label>
              <ChoiceSelect
                id="overview-stats-currency"
                className="h-11 rounded-xl border bg-card px-3"
                value={item.currency}
                onValueChange={(value) => setCurrency(value)}
              >
                {query.data.currencies.map((c) => (
                  <option key={c.currency}>{c.currency}</option>
                ))}
              </ChoiceSelect>
            </div>
          )}
          <StatisticsCharts
            key={`${from}:${to}:${granularity}:${item.currency}`}
            item={item}
            compact={compact}
          />
        </>
      ) : (
        <p>Пока нет финансовых данных для графиков.</p>
      )}
    </div>
  )
  if (compact)
    return (
      <Card>
        <CardHeader>
          <CardTitle>Как меняются покупки</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {body}
          <Link
            className="inline-flex min-h-11 items-center text-primary underline underline-offset-4"
            to="/analytics"
          >
            Открыть статистику
          </Link>
        </CardContent>
      </Card>
    )
  return (
    <section className="min-w-0 space-y-6" aria-label="Статистика">
      {body}
    </section>
  )
}
