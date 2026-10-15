import type { components } from '@/shared/api/generated/schema'
import { eventDateText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { CategoryIcon } from '@/shared/ui/category-icon'
import { LoadingSkeleton } from '@/shared/ui/skeleton'

type Category = components['schemas']['Category']

export function SubscriptionCategory({ category }: { category?: Category }) {
  if (!category) return <p className="mt-1 text-sm text-muted-foreground">Без категории</p>
  return (
    <p className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground">
      <CategoryIcon icon={category.icon} className="size-4" />
      {category.name}
    </p>
  )
}
export function ExpenseHistory({
  loading,
  error,
  events,
}: {
  loading: boolean
  error: Error | null
  events: components['schemas']['FinancialEvent'][]
}) {
  if (loading) return <LoadingSkeleton label="Загружаем расходы…" />
  if (error) return <p role="alert">{error.message}</p>
  if (!events.length)
    return <p className="text-sm text-muted-foreground">Подтверждённых расходов пока нет.</p>
  return (
    <section className="space-y-3">
      <h3 className="font-medium">Последние связанные покупки · до 100</h3>
      {events.map((e) => (
        <div
          key={e.id}
          className="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-muted/50 p-3 text-sm"
        >
          <div>
            <p>{e.merchant || e.description || 'Покупка'}</p>
            <p className="text-xs text-muted-foreground">{eventDateText(e)}</p>
          </div>
          <div className="tabular-nums">
            {e.postings
              .filter((p) => p.direction === 'debit')
              .map((p) => (
                <p key={p.id}>{formatMoney(p.amountMinor, p.currency)}</p>
              ))}
          </div>
        </div>
      ))}
    </section>
  )
}
