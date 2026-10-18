import type { components } from '@/shared/api/generated/schema'
import { statementIssueLabel } from '../model/issues'
import { Money } from '@/shared/ui/money'

const statusLabels: Record<string, string> = {
  matched: 'Совпало',
  mismatch: 'Расхождение',
  incomplete: 'Неполные данные',
  not_provided: 'Нет в документе',
  not_reviewed: 'Нужна проверка страниц',
  not_evaluated: 'Не проверялся',
  represented: 'Все строки связаны с операциями',
}
export function StatementReconciliation({
  review,
}: {
  review: components['schemas']['StatementReview']
}) {
  const r = review.reconciliation
  return (
    <div className="space-y-2 text-sm">
      <p>Обороты: {statusLabels[r.turnoverCheck.status] ?? r.turnoverCheck.status}</p>
      <p>
        Бухгалтерские остатки:{' '}
        {statusLabels[r.bookBalanceCheck.status] ?? r.bookBalanceCheck.status}
      </p>
      <p>
        Финансовое покрытие:{' '}
        {statusLabels[r.financialCoverage?.status ?? 'not_evaluated'] ?? 'Не проверялось'}
      </p>
      <p>Полнота источника: {statusLabels[r.rowCoverage.status] ?? r.rowCoverage.status}</p>
      <p>
        Исторический остаток Grace:{' '}
        {statusLabels[r.ledgerBalanceCheck.status] ?? r.ledgerBalanceCheck.status}
      </p>
      {review.header.availableBalanceObserved && (
        <p>
          Доступный остаток из документа:{' '}
          <Money
            valueMinor={review.header.availableBalanceObserved.amountMinor}
            currency={review.header.availableBalanceObserved.currency}
            size="sm"
          />
          . Это отдельное наблюдение банка.
        </p>
      )}
      {r.blockers.length > 0 && (
        <div role="status">
          <p>Подтверждение заблокировано:</p>
          <ul className="list-inside list-disc break-words">
            {r.blockers.map((b, i) => (
              <li key={i}>{statementIssueLabel(b)}</li>
            ))}
          </ul>
        </div>
      )}
      {r.warnings.length > 0 && (
        <p>Есть непроверенные итоги. Проверьте каждую строку перед подтверждением.</p>
      )}
    </div>
  )
}
