import type { components } from '@/shared/api/generated/schema'

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
      <h3 className="font-medium">Фрагменты банковского источника</h3>
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
