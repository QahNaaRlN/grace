import type { components } from '@/shared/api/generated/schema'
import { Money } from '@/shared/ui/money'

const labels: Record<string, string> = {
  operationDate: 'Дата операции',
  bookingDate: 'Дата списания',
  accountAmount: 'Сумма по счёту',
  transactionAmount: 'Исходная валютная сумма',
  description: 'Описание',
  accountAmountCurrency: 'Валюта счёта',
  transactionAmountCurrency: 'Валюта операции',
}
export function StatementFieldEvidence({ row }: { row: components['schemas']['StatementRow'] }) {
  return (
    <div className="space-y-2 text-sm">
      <h3 className="font-medium">
        {row.evidence.length ? 'Фрагменты банковского источника' : 'Значения из таблицы выписки'}
      </h3>
      {row.evidence.length === 0 && (
        <dl className="space-y-1">
          <dt>Дата и время операции</dt>
          <dd>{row.operationDate?.local.replace('T', ' ') ?? 'Не указаны'}</dd>
          <dt>Дата и время списания</dt>
          <dd>{row.bookingDate?.local.replace('T', ' ') ?? 'Не указаны'}</dd>
          <dt>Движение по счёту</dt>
          <dd>
            {row.direction === 'debit'
              ? 'Списание'
              : row.direction === 'credit'
                ? 'Зачисление'
                : 'Не определено'}
          </dd>
          <dt>Сумма по счёту</dt>
          <dd>
            {row.accountAmount ? (
              <Money
                valueMinor={row.accountAmount.amountMinor}
                currency={row.accountAmount.currency}
                size="sm"
              />
            ) : (
              'Не указана'
            )}
          </dd>
          <dt>Исходная валютная сумма</dt>
          <dd>
            {row.transactionAmount ? (
              <Money
                valueMinor={row.transactionAmount.amountMinor}
                currency={row.transactionAmount.currency}
                size="sm"
              />
            ) : (
              'Не указана'
            )}
          </dd>
          <dt>Описание</dt>
          <dd>{row.description || 'Не указано'}</dd>
        </dl>
      )}
      {row.evidence.map((field, i) => (
        <p key={i} className="break-words">
          <span className="text-muted-foreground">
            {labels[field.role.split(':').at(-1) ?? ''] ?? 'Исходное поле'} · стр.{' '}
            {[...new Set(field.spans.map((span) => span.page))].join(', ')}:{' '}
          </span>
          {field.literal}
        </p>
      ))}
      {row.operationDate && row.operationDate.precision !== 'date' && !row.operationDate.utc && (
        <p>
          Часовой пояс не указан. История переносится с точностью до даты; исходное время сохранено
          выше.
        </p>
      )}
    </div>
  )
}
