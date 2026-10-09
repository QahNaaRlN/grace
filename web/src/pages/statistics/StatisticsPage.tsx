import { ChoiceSelect } from '@/shared/ui/choice-select'
import { useEffect, useId, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { apiClient } from '@/shared/api/client'
import type { StatisticsBucket, StatisticsCurrency } from '@/shared/api/types'
import { formatMoney } from '@/shared/lib/finance-display'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'

function defaultStatisticsPeriod() {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Yekaterinburg', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date())
  const get = (key: string) => Number(parts.find(p => p.type === key)?.value)
  const day = new Date(Date.UTC(get('year'), get('month') - 1, get('day')))
  return { from: new Date(Date.UTC(get('year'), get('month') - 6, 1)).toISOString().slice(0, 10), to: new Date(day.getTime() + 86400000).toISOString().slice(0, 10) }
}
const dayText = (day: string) => new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' }).format(new Date(day + 'T00:00:00Z'))
const rangeText = (bucket: StatisticsBucket) => `${dayText(bucket.from)} — до ${dayText(bucket.to)}, не включая`
const shortDate = (day: string) => new Intl.DateTimeFormat('ru-RU', { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(day + 'T00:00:00Z'))
type RowKey = 'purchasesMinor' | 'incomeMinor' | 'outflowsMinor'

export function SeriesChart({ item, comparison = false, compact = false }: { item: StatisticsCurrency; comparison?: boolean; compact?: boolean }) {
  const id = useId()
  const [selection, setSelection] = useState<string>()
  const plotRef = useRef<HTMLDivElement>(null)
  const [plotWidth, setPlotWidth] = useState(0)
  useEffect(() => {
    const element = plotRef.current
    if (!element || !compact) return
    const update = () => { if (element.clientWidth) setPlotWidth(element.clientWidth - 16) }
    update()
    const observer = new ResizeObserver(update)
    observer.observe(element)
    return () => observer.disconnect()
  }, [compact])
  const keys: RowKey[] = comparison ? ['incomeMinor', 'outflowsMinor'] : ['purchasesMinor']
  const labels: Record<RowKey, string> = { purchasesMinor: 'Покупки', incomeMinor: 'Доходы', outflowsMinor: 'Покупки и платежи' }
  const chosen = item.series.find(b => b.from === selection) ?? item.series.at(-1)
  const max = Math.max(1, ...item.series.flatMap(b => keys.map(k => b[k])))
  const hasValues = item.series.some(b => keys.some(k => b[k] > 0))
  const width = Math.max(compact ? plotWidth : comparison ? 900 : 480, 480, item.series.length * 56)
  const step = (width - 130) / Math.max(item.series.length, 1)
  return <div className="space-y-3">
    <div className="flex flex-wrap gap-x-5 gap-y-2 text-sm">{keys.map((key, index) => <span key={key} className="flex items-center gap-2"><span className="inline-block size-3 rounded-sm" style={{ background: `var(--chart-${index + 1})` }} />{labels[key]}</span>)}</div>
    <p className="text-sm text-muted-foreground sm:hidden">Проведите по графику в сторону, чтобы увидеть другие периоды.</p>
    <div ref={plotRef} className="overflow-x-auto rounded-xl border bg-background/40 p-2" tabIndex={0} aria-label="Область графика, прокрутка при необходимости">
      {!hasValues ? <p className="p-4 text-sm text-muted-foreground">{comparison ? 'Нет записанных доходов, покупок и платежей за этот период.' : 'Нет записанных покупок за этот период.'}</p> : <svg viewBox={`0 0 ${width} 230`} style={{ width: '100%', minWidth: width, height: compact ? 230 : 240 }} role="group" aria-labelledby={id}>
        <title id={id}>{comparison ? 'Доходы и покупки с платежами' : 'Динамика покупок'}, {item.currency}. {compact ? 'Выберите период, чтобы увидеть сумму.' : 'Значения доступны в таблице.'}</title>
        {[0, 0.5, 1].map(ratio => <g key={ratio}><line x1="110" x2={width - 10} y1={190 - ratio * 150} y2={190 - ratio * 150} stroke="var(--border)" /><text x="100" textAnchor="end" y={194 - ratio * 150} fontSize="11" fill="var(--muted-foreground)">{formatMoney(Math.round(max * ratio), item.currency)}</text></g>)}
        {item.series.map((bucket, index) => <g key={bucket.from}>
          {keys.map((key, ki) => {
            const barWidth = Math.min(compact ? 36 : 22, step / (keys.length + 1))
            const height = bucket[key] / max * 150
            const label = `${rangeText(bucket)}: ${labels[key]} ${formatMoney(bucket[key], item.currency)}`
            return <rect key={key} x={120 + index * step + ki * barWidth} y={190 - Math.max(height, 2)} width={barWidth - 2} height={Math.max(height, 2)} rx="3" fill={`var(--chart-${ki + 1})`} fillOpacity={bucket[key] === 0 ? 0.2 : 1} tabIndex={0} role="button" aria-label={label} aria-pressed={chosen?.from === bucket.from} onClick={() => setSelection(bucket.from)} onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setSelection(bucket.from) } }} className="cursor-pointer focus:outline-2 focus:outline-ring"><title>{label}</title></rect>
          })}
          {(!compact || item.series.length <= 7) && <text x={120 + index * step} y="216" fontSize="11" fill="var(--muted-foreground)">{shortDate(bucket.from)}</text>}
        </g>)}
      </svg>}
    </div>
    {chosen && <p aria-live="polite" className="text-sm"><span className="text-muted-foreground">{rangeText(chosen)}</span><br />{keys.map(key => `${labels[key]}: ${formatMoney(chosen[key], item.currency)}`).join(' · ')}</p>}
    {!compact && <details className="rounded-xl border p-3"><summary className="min-h-11 content-center font-medium">Таблица значений</summary><div className="overflow-x-auto"><table className="w-full text-left text-sm"><caption className="sr-only">Суммы записанных событий в {item.currency}</caption><thead><tr><th className="p-2">Период (конец не включается)</th><th className="p-2">Доходы</th><th className="p-2">Покупки</th><th className="p-2">Платежи</th><th className="p-2">Возвраты</th></tr></thead><tbody>{item.series.map(bucket => <tr key={bucket.from} className="border-t"><th scope="row" className="p-2 font-normal">{rangeText(bucket)}</th>{[bucket.incomeMinor, bucket.purchasesMinor, bucket.debtPaymentsMinor, bucket.refundsMinor].map((value, i) => <td key={i} className="p-2 whitespace-nowrap tabular-nums">{formatMoney(value, item.currency)}</td>)}</tr>)}</tbody></table></div></details>}
  </div>
}

export function CategoryChart({ item }: { item: StatisticsCurrency }) {
  const max = Math.max(1, ...item.categories.map(c => c.purchasesMinor))
  return <div className="space-y-5">
    <p className="text-sm text-muted-foreground">Доли покупок до возвратов. Платежи по обязательствам показаны отдельно в денежном потоке.</p>
    {item.summary.purchasesMinor === 0 && <p>За период нет записанных покупок.</p>}
    {item.categories.map(category => <div key={category.categoryId ?? 'uncategorized'} className="space-y-2">
      <div className="flex flex-wrap items-baseline justify-between gap-2 text-sm"><span className="font-medium">{category.categoryName}</span><span className="tabular-nums">{formatMoney(category.purchasesMinor, item.currency)} · {(category.shareBasisPoints / 100).toLocaleString('ru-RU')}%</span></div>
      <div aria-hidden="true" className="h-3 overflow-hidden rounded-full bg-muted"><div className="h-full rounded-full bg-chart-1" style={{ width: `${category.purchasesMinor / max * 100}%` }} /></div>
      {category.refundsMinor > 0 && <p className="text-sm text-muted-foreground">Возвраты {formatMoney(category.refundsMinor, item.currency)} · после возвратов {formatMoney(category.netExpenseMinor, item.currency)}</p>}
    </div>)}
    <Link to="/operations" className="inline-flex min-h-11 items-center text-sm text-primary underline underline-offset-4">Открыть последние операции</Link>
  </div>
}

export function StatisticsCharts({ item, compact = false }: { item: StatisticsCurrency; compact?: boolean }) {
  const amounts = [...item.series.flatMap(b => [b.incomeMinor, b.purchasesMinor, b.debtPaymentsMinor, b.refundsMinor, b.outflowsMinor]), ...item.categories.flatMap(c => [c.purchasesMinor, c.refundsMinor, c.netExpenseMinor])]
  if (!amounts.every(Number.isSafeInteger)) return <p role="alert">Суммы слишком велики для точного отображения. График не построен.</p>
  if (compact) return <SeriesChart item={item} compact />
  return <div className="grid min-w-0 gap-6 xl:grid-cols-2">
    <Card><CardHeader><CardTitle>Куда уходят деньги</CardTitle></CardHeader><CardContent><CategoryChart item={item} /></CardContent></Card>
    <Card><CardHeader><CardTitle>Динамика покупок</CardTitle></CardHeader><CardContent><SeriesChart item={item} /></CardContent></Card>
    <Card className="xl:col-span-2"><CardHeader><CardTitle>Доходы и расходы</CardTitle></CardHeader><CardContent className="space-y-4"><p className="text-sm text-muted-foreground">Показаны покупки и платежи по обязательствам. Возвраты доступны в таблице; переводы между своими счетами исключены.</p><SeriesChart item={item} comparison /></CardContent></Card>
  </div>
}

export function StatisticsPage({ compact = false }: { compact?: boolean }) {
  const defaults = defaultStatisticsPeriod()
  const [from, setFrom] = useState(defaults.from)
  const [to, setTo] = useState(defaults.to)
  const [granularity, setGranularity] = useState<'month' | 'week'>('month')
  const [currency, setCurrency] = useState('RUB')
  const maxTo = new Date(from + 'T00:00:00Z')
  maxTo.setUTCFullYear(maxTo.getUTCFullYear() + 1)
  const valid = from !== '' && to !== '' && from < to && Number.isFinite(maxTo.getTime()) && to <= maxTo.toISOString().slice(0, 10)
  const query = useQuery({ queryKey: ['statistics', from, to, granularity], enabled: valid, queryFn: async () => {
    const { data, error } = await apiClient.GET('/api/v1/analytics/statistics', { params: { query: { from, to, granularity } } })
    if (!data || !Array.isArray(data.currencies)) throw new Error(error?.message ?? 'Не удалось загрузить статистику')
    return data
  } })
  const item = query.data?.currencies.find(c => c.currency === currency) ?? query.data?.currencies[0]
  const body = <div className="min-w-0 space-y-5">
    {!compact && <div className="grid gap-4 sm:grid-cols-4">
      <div className="space-y-2"><Label htmlFor="stats-from">С даты</Label><Input id="stats-from" type="date" value={from} onChange={e => setFrom(e.target.value)} /></div>
      <div className="space-y-2"><Label htmlFor="stats-to">До даты, не включая</Label><Input id="stats-to" type="date" value={to} onChange={e => setTo(e.target.value)} /></div>
      <div className="space-y-2"><Label htmlFor="stats-group">Группировка</Label><ChoiceSelect id="stats-group" value={granularity} onValueChange={value => setGranularity(value as 'month' | 'week')} className="h-11 w-full rounded-xl border border-input bg-card px-3 focus-visible:outline-2 focus-visible:outline-ring"><option value="month">По месяцам</option><option value="week">По неделям</option></ChoiceSelect></div>
      <div className="space-y-2"><Label htmlFor="stats-currency">Валюта</Label><ChoiceSelect id="stats-currency" value={item?.currency ?? ''} onValueChange={value => setCurrency(value)} disabled={!query.data?.currencies.length} className="h-11 w-full rounded-xl border border-input bg-card px-3 focus-visible:outline-2 focus-visible:outline-ring disabled:cursor-not-allowed">{query.data?.currencies.map(c => <option key={c.currency}>{c.currency}</option>)}</ChoiceSelect></div>
    </div>}
    <p className="text-sm text-muted-foreground">Только записанные события · Asia/Yekaterinburg. Полнота истории не подтверждена; ноль на графике означает отсутствие записей. Неполные крайние периоды показаны по выбранным датам.</p>
    {!valid ? <p role="alert">Выберите период до одного года; конец должен быть позже начала.</p> : query.isError ? <div role="alert" className="space-y-3"><p>{query.error.message}</p><Button variant="outline" onClick={() => void query.refetch()}>Повторить загрузку</Button></div> : query.isPending ? <p role="status">Загружаем статистику…</p> : item ? <>
      <p className="text-sm">{dayText(from)} — до {dayText(to)}, не включая · {item.currency} · Покупки {formatMoney(item.summary.purchasesMinor, item.currency)}</p>
      {compact && query.data.currencies.length > 1 && <div className="space-y-2"><Label htmlFor="overview-stats-currency">Валюта графика</Label><ChoiceSelect id="overview-stats-currency" className="h-11 rounded-xl border bg-card px-3" value={item.currency} onValueChange={value => setCurrency(value)}>{query.data.currencies.map(c => <option key={c.currency}>{c.currency}</option>)}</ChoiceSelect></div>}
      <StatisticsCharts key={`${from}:${to}:${granularity}:${item.currency}`} item={item} compact={compact} />
    </> : <p>Пока нет финансовых данных для графиков.</p>}
  </div>
  if (compact) return <Card><CardHeader><CardTitle>Как меняются покупки</CardTitle></CardHeader><CardContent className="space-y-4">{body}<Link className="inline-flex min-h-11 items-center text-primary underline underline-offset-4" to="/analytics">Открыть статистику</Link></CardContent></Card>
  return <section className="min-w-0 space-y-6" aria-label="Статистика">{body}</section>
}
