import { useQuery } from '@tanstack/react-query'
import { installmentQueries } from '@/entities/installment'
import { formatMoney } from '@/shared/lib/money'

export function InstallmentLinks({
  accountId,
  purchaseEventId,
}: {
  accountId?: string
  purchaseEventId?: string
}) {
  const plans = useQuery(installmentQueries.list())
  const linked =
    plans.data?.filter((p) =>
      accountId ? p.accountId === accountId : p.purchaseEventId === purchaseEventId,
    ) ?? []
  if (!linked.length) return null
  return (
    <section className="space-y-1 text-sm">
      <h4 className="font-medium">Связанные рассрочки</h4>
      {linked.map((p) => (
        <p key={p.id}>
          <a className="underline" href="/payments">
            {p.name}
          </a>{' '}
          · остаток {formatMoney(p.remainingMinor, p.currency)}
          {p.mode === 'credit_card' ? ' (в составе долга карты)' : ''}
        </p>
      ))}
    </section>
  )
}
