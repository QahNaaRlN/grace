import { useState } from 'react'

import { Button } from '@/shared/ui/button'
import { SourceText } from './SourceText'
import { MappingCoverage } from './MappingCoverage'
import type { components } from '@/shared/api/generated/schema'
import { statementIssueLabel } from '../model/issues'
import { Money } from '@/shared/ui/money'

const statusLabels: Record<string, string> = {
  matched: 'Совпало',
  owner_reviewed: 'Сверено по изображениям',
  mismatch: 'Расхождение',
  incomplete: 'Неполные данные',
  not_provided: 'Нет в документе',
  not_reviewed: 'Нужна проверка страниц',
  not_evaluated: 'Не проверялся',
  represented: 'Все строки связаны с операциями',
}
export function StatementReconciliation({
  review,
  layout,
  mapping,
}: {
  review: components['schemas']['StatementReview']
  layout?: components['schemas']['StatementLayout']
  mapping?: components['schemas']['StatementDraftMapping']
}) {
  const [selection, setSelection] = useState<string[]>([])
  const r = review.reconciliation
  return (
    <div className="space-y-2 text-sm">
      <div className="grid gap-3 sm:grid-cols-2">
        {(['debit', 'credit'] as const).map((side) => {
          const role = side === 'debit' ? 'debitTurnover' : 'creditTurnover'
          const printed = review.header[role]
          const computed = side === 'debit' ? r.debitMinor : r.creditMinor
          const evidence = review.header.evidence.filter((field) => field.role === role)
          return (
            <div key={side} className="space-y-2 rounded-xl border p-3">
              <h3 className="font-medium">{side === 'debit' ? 'Списания' : 'Зачисления'}</h3>
              <p>
                Выбранный итог:{' '}
                {printed ? (
                  <Money valueMinor={printed.amountMinor} currency={printed.currency} size="sm" />
                ) : (
                  'Не выбран или не распознан'
                )}
              </p>
              <p>
                Сумма строк:{' '}
                {computed != null ? (
                  <Money valueMinor={computed} currency="RUB" size="sm" />
                ) : (
                  'Не рассчитана'
                )}
              </p>
              {evidence.map((field, index) => (
                <div key={index}>
                  <p className="break-words">Исходные фрагменты: {field.literal}</p>
                  {layout && (
                    <Button
                      variant="outline"
                      onClick={() => setSelection(field.spans.map((span) => span.id))}
                    >
                      Показать источник {side === 'debit' ? 'списаний' : 'зачислений'}
                    </Button>
                  )}
                </div>
              ))}
            </div>
          )
        })}
      </div>
      {layout && selection.length > 0 && <SourceText layout={layout} selected={selection} />}
      {layout && mapping && <MappingCoverage layout={layout} mapping={mapping} />}
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
        <p>
          {r.warnings.includes('turnover_not_checked')
            ? 'Напечатанные обороты не удалось проверить полностью.'
            : r.warnings.includes('book_balance_not_checked')
              ? 'Бухгалтерские остатки не проверены: в документе может не быть этих данных.'
              : 'Сверьте исходные данные и назначение операций перед подтверждением.'}
        </p>
      )}
    </div>
  )
}
