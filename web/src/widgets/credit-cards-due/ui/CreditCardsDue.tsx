import { useQueries } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { creditCardQueries } from '@/entities/credit-card'
import type { Account } from '@/shared/api'
import { dateText } from '@/shared/lib/date'
import { accountLabel } from '@/shared/lib/finance-display'
import { formatMoney } from '@/shared/lib/money'
import { StatusBadge } from '@/shared/ui/status'

/** SPEC-0051: on the overview, what each credit card needs next, nearest date first. */
export function CreditCardsDue({ accounts }: { accounts: Account[] }) {
  const cards = accounts.filter((account) => account.type === 'credit')
  const results = useQueries({ queries: cards.map((card) => creditCardQueries.detail(card.id)) })
  const due = cards
    .map((card, index) => ({ card, status: results[index]?.data?.status }))
    .flatMap(({ card, status }) => {
      if (!status) return []
      const grace = status.graceAmountMinor ?? 0
      const minimum = status.minimumMinor ?? 0
      const amount = grace > 0 ? grace : minimum
      const date = grace > 0 ? status.graceDueOn : status.minimumDueOn
      return amount > 0 && date ? [{ card, status, amount, date, keepsGrace: grace > 0 }] : []
    })
    .sort((a, b) => a.date.localeCompare(b.date))
  if (due.length === 0) return null
  return (
    <section aria-labelledby="cards-due" className="space-y-3 rounded-2xl border bg-card p-5">
      <h2 id="cards-due" className="font-semibold">
        Кредитные карты
      </h2>
      <ul className="space-y-2">
        {due.map(({ card, status, amount, date, keepsGrace }) => (
          <li key={card.id} className="flex flex-wrap items-center gap-2 text-sm">
            <Link to="/accounts" className="font-medium underline-offset-4 hover:underline">
              {accountLabel(card)}
            </Link>
            <span>
              {keepsGrace ? 'заплатите' : 'минимальный платёж'}{' '}
              {!keepsGrace && status.minimumIsUpperBound ? 'до ' : ''}
              <strong>{formatMoney(amount, card.currency)}</strong> до {dateText(date)}
              {keepsGrace ? ', чтобы не платить проценты' : ''}
            </span>
            {status.graceLost && <StatusBadge tone="danger">Льготный период потерян</StatusBadge>}
            {!status.graceLost && status.graceAtRisk && (
              <StatusBadge tone="warning">Скоро срок</StatusBadge>
            )}
          </li>
        ))}
      </ul>
    </section>
  )
}
