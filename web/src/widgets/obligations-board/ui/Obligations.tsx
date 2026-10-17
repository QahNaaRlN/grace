import { CalendarClock } from 'lucide-react'
import { LoanPanel } from '@/features/edit-loan'
import { ObligationSchedule } from '@/features/obligation-schedule'
import type { FinancialEvent, Obligation } from '@/shared/api'
import { formatMoney } from '@/shared/lib/money'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Money } from '@/shared/ui/money'
import { StatusBadge } from '@/shared/ui/status'

const obligationKindLabels: Record<Obligation['kind'], string> = {
  loan: 'Кредит',
  installment: 'Рассрочка',
  other_fixed: 'Другое',
}

export function Obligations({ items, events }: { items: Obligation[]; events: FinancialEvent[] }) {
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
              {item.kind === 'loan' && (
                <LoanPanel
                  obligationId={item.id}
                  currency={item.currency}
                  events={events}
                  usedEventIDs={usedEventIDs}
                />
              )}
              <ObligationSchedule obligation={item} events={events} usedEventIDs={usedEventIDs} />
            </article>
          ))
        )}
      </CardContent>
    </Card>
  )
}
