import { Disclosure } from '@/shared/ui/disclosure'
import { useEffect, useId, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import type { StatisticsCurrency } from '@/shared/api'
import { formatMoney } from '@/shared/lib/money'
import { rangeText, shortDate, type RowKey } from './statistics-format'

export function SeriesChart({
  item,
  comparison = false,
  compact = false,
}: {
  item: StatisticsCurrency
  comparison?: boolean
  compact?: boolean
}) {
  const id = useId()
  const [selection, setSelection] = useState<string>()
  const plotRef = useRef<HTMLDivElement>(null)
  const [plotWidth, setPlotWidth] = useState(0)
  useEffect(() => {
    const element = plotRef.current
    if (!element || !compact) return
    const update = () => {
      if (element.clientWidth) setPlotWidth(element.clientWidth - 16)
    }
    update()
    const observer = new ResizeObserver(update)
    observer.observe(element)
    return () => observer.disconnect()
  }, [compact])
  const keys: RowKey[] = comparison ? ['incomeMinor', 'outflowsMinor'] : ['purchasesMinor']
  const labels: Record<RowKey, string> = {
    purchasesMinor: 'Покупки',
    incomeMinor: 'Доходы',
    outflowsMinor: 'Покупки и платежи',
  }
  const chosen = item.series.find((b) => b.from === selection) ?? item.series.at(-1)
  const max = Math.max(1, ...item.series.flatMap((b) => keys.map((k) => b[k])))
  const hasValues = item.series.some((b) => keys.some((k) => b[k] > 0))
  const width = Math.max(compact ? plotWidth : comparison ? 900 : 480, 480, item.series.length * 56)
  const step = (width - 130) / Math.max(item.series.length, 1)
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-x-5 gap-y-2 text-sm">
        {keys.map((key, index) => (
          <span key={key} className="flex items-center gap-2">
            <span
              className="inline-block size-3 rounded-sm"
              style={{ background: `var(--chart-${index + 1})` }}
            />
            {labels[key]}
          </span>
        ))}
      </div>
      <p className="text-sm text-muted-foreground sm:hidden">
        Проведите по графику в сторону, чтобы увидеть другие периоды.
      </p>
      <div
        ref={plotRef}
        className="overflow-x-auto rounded-xl border bg-background/40 p-2"
        tabIndex={0}
        aria-label="Область графика, прокрутка при необходимости"
      >
        {!hasValues ? (
          <p className="p-4 text-sm text-muted-foreground">
            {comparison
              ? 'Нет записанных доходов, покупок и платежей за этот период.'
              : 'Нет записанных покупок за этот период.'}
          </p>
        ) : (
          <svg
            viewBox={`0 0 ${width} 230`}
            style={{ width: '100%', minWidth: width, height: compact ? 230 : 240 }}
            role="group"
            aria-labelledby={id}
          >
            <title id={id}>
              {comparison ? 'Доходы и покупки с платежами' : 'Динамика покупок'}, {item.currency}.{' '}
              {compact ? 'Выберите период, чтобы увидеть сумму.' : 'Значения доступны в таблице.'}
            </title>
            {[0, 0.5, 1].map((ratio) => (
              <g key={ratio}>
                <line
                  x1="110"
                  x2={width - 10}
                  y1={190 - ratio * 150}
                  y2={190 - ratio * 150}
                  stroke="var(--border)"
                />
                <text
                  x="100"
                  textAnchor="end"
                  y={194 - ratio * 150}
                  fontSize="11"
                  fill="var(--muted-foreground)"
                >
                  {formatMoney(Math.round(max * ratio), item.currency)}
                </text>
              </g>
            ))}
            {item.series.map((bucket, index) => (
              <g key={bucket.from}>
                {keys.map((key, ki) => {
                  const barWidth = Math.min(compact ? 36 : 22, step / (keys.length + 1))
                  const height = (bucket[key] / max) * 150
                  const label = `${rangeText(bucket)}: ${labels[key]} ${formatMoney(bucket[key], item.currency)}`
                  return (
                    <rect
                      key={key}
                      x={120 + index * step + ki * barWidth}
                      y={190 - Math.max(height, 2)}
                      width={barWidth - 2}
                      height={Math.max(height, 2)}
                      rx="3"
                      fill={`var(--chart-${ki + 1})`}
                      fillOpacity={bucket[key] === 0 ? 0.2 : 1}
                      tabIndex={0}
                      role="button"
                      aria-label={label}
                      aria-pressed={chosen?.from === bucket.from}
                      onClick={() => setSelection(bucket.from)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          setSelection(bucket.from)
                        }
                      }}
                      className="cursor-pointer focus:outline-2 focus:outline-ring"
                    >
                      <title>{label}</title>
                    </rect>
                  )
                })}
                {(!compact || item.series.length <= 7) && (
                  <text x={120 + index * step} y="216" fontSize="11" fill="var(--muted-foreground)">
                    {shortDate(bucket.from)}
                  </text>
                )}
              </g>
            ))}
          </svg>
        )}
      </div>
      {chosen && (
        <p aria-live="polite" className="text-sm">
          <span className="text-muted-foreground">{rangeText(chosen)}</span>
          <br />
          {keys
            .map((key) => `${labels[key]}: ${formatMoney(chosen[key], item.currency)}`)
            .join(' · ')}
        </p>
      )}
      {!compact && (
        <Disclosure
          className="rounded-xl border p-3"
          triggerClassName="min-h-11 content-center font-medium"
          title={<>Таблица значений</>}
        >
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <caption className="sr-only">Суммы записанных событий в {item.currency}</caption>
              <thead>
                <tr>
                  <th className="p-2">Период (конец не включается)</th>
                  <th className="p-2">Доходы</th>
                  <th className="p-2">Покупки</th>
                  <th className="p-2">Платежи</th>
                  <th className="p-2">Возвраты</th>
                </tr>
              </thead>
              <tbody>
                {item.series.map((bucket) => (
                  <tr key={bucket.from} className="border-t">
                    <th scope="row" className="p-2 font-normal">
                      {rangeText(bucket)}
                    </th>
                    {[
                      bucket.incomeMinor,
                      bucket.purchasesMinor,
                      bucket.debtPaymentsMinor,
                      bucket.refundsMinor,
                    ].map((value, i) => (
                      <td key={i} className="p-2 whitespace-nowrap tabular-nums">
                        {formatMoney(value, item.currency)}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Disclosure>
      )}
    </div>
  )
}

export function CategoryChart({ item }: { item: StatisticsCurrency }) {
  const max = Math.max(1, ...item.categories.map((c) => c.purchasesMinor))
  return (
    <div className="space-y-5">
      <p className="text-sm text-muted-foreground">
        Доли покупок до возвратов. Платежи по обязательствам показаны отдельно в денежном потоке.
      </p>
      {item.summary.purchasesMinor === 0 && <p>За период нет записанных покупок.</p>}
      {item.categories.map((category) => (
        <div key={category.categoryId ?? 'uncategorized'} className="space-y-2">
          <div className="flex flex-wrap items-baseline justify-between gap-2 text-sm">
            <span className="font-medium">{category.categoryName}</span>
            <span className="tabular-nums">
              {formatMoney(category.purchasesMinor, item.currency)} ·{' '}
              {(category.shareBasisPoints / 100).toLocaleString('ru-RU')}%
            </span>
          </div>
          <div aria-hidden="true" className="h-3 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-chart-1"
              style={{ width: `${(category.purchasesMinor / max) * 100}%` }}
            />
          </div>
          {category.refundsMinor > 0 && (
            <p className="text-sm text-muted-foreground">
              Возвраты {formatMoney(category.refundsMinor, item.currency)} · после возвратов{' '}
              {formatMoney(category.netExpenseMinor, item.currency)}
            </p>
          )}
          <Link
            to={`/operations?category=${encodeURIComponent(category.categoryId ?? 'uncategorized')}`}
            className="inline-flex min-h-11 items-center text-sm text-primary underline underline-offset-4"
          >
            Посмотреть покупки: {category.categoryName}
          </Link>
        </div>
      ))}
      <Link
        to="/operations"
        className="inline-flex min-h-11 items-center text-sm text-primary underline underline-offset-4"
      >
        Открыть последние операции
      </Link>
    </div>
  )
}
