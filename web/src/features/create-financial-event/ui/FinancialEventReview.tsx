// The confirmation step: what will be written, before anything is sent.
export function FinancialEventReview({
  kind,
  amount,
  account,
  target,
  occurredAt,
  category,
  merchant,
  description,
}: {
  kind: string
  amount: string
  account: string
  /** The destination account of a transfer; other kinds have none. */
  target?: string
  occurredAt: string
  category: string
  merchant: string
  description: string
}) {
  return (
    <dl className="grace-fields">
      <div>
        <dt>Тип</dt>
        <dd>{kind}</dd>
      </div>
      <div>
        <dt>Сумма</dt>
        <dd>{amount}</dd>
      </div>
      <div>
        <dt>Счёт / карта</dt>
        <dd>{account}</dd>
      </div>
      {target !== undefined && (
        <div>
          <dt>На счёт</dt>
          <dd>{target}</dd>
        </div>
      )}
      <div>
        <dt>Дата</dt>
        <dd>{new Date(occurredAt).toLocaleString('ru-RU')}</dd>
      </div>
      <div>
        <dt>Категория</dt>
        <dd>{category}</dd>
      </div>
      <div>
        <dt>Получатель</dt>
        <dd>{merchant || 'Не указано'}</dd>
      </div>
      <div>
        <dt>Комментарий</dt>
        <dd>{description || 'Не указано'}</dd>
      </div>
    </dl>
  )
}
