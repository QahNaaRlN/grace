const issues: Record<string, string> = {
  owner_field_role_review_required: 'Подтвердите роли дат, сумм, валютных колонок и остатков.',
  owner_page_review_required: 'Проверьте покрытие каждой страницы.',
  source_coverage_incomplete: 'Остались фрагменты без назначения. Проверьте полноту строк.',
  invalid_source_rows: 'Не все исходные строки удалось проверить.',
  turnover_role_unverified:
    'Проверьте подпись и источник оборота: реквизиты банка и удалённые подписи не подходят.',
  description_evidence_gap:
    'В исходном описании пропущены фрагменты между выбранными словами. Проверьте разметку.',
  description_evidence_unverified:
    'Недостаточно исходных фрагментов, чтобы проверить полноту описания. Повторите разметку.',
  book_balance_role_unverified:
    'Проверьте подпись и источник начального или конечного бухгалтерского остатка.',
  printed_turnover_mismatch: 'Сумма строк расходится с напечатанным оборотом.',
  book_balance_mismatch: 'Бухгалтерские остатки расходятся с движениями за период.',
  available_is_not_book_balance: 'Доступный остаток банка нельзя использовать как бухгалтерский.',
  invalid_span_reference: 'Значение ссылается на отсутствующий исходный фрагмент.',
  overlapping_evidence: 'Один исходный фрагмент использован как несколько разных фактов.',
  account_amount_required: 'Нужна исходная сумма по выбранному счёту.',
  operation_date_required: 'Нужна исходная дата операции.',
  field_label_required: 'Проверьте исходный заголовок поля.',
  direction_required: 'Укажите направление движения по исходному заголовку.',
  direction_conflict: 'Направление противоречит знаку исходной суммы.',
  ambiguous_money: 'Сумму нельзя однозначно прочитать с точностью до копейки.',
  amount_overflow: 'Сумма выходит за допустимый диапазон.',
  aggregate_overflow: 'Итог выходит за допустимый диапазон.',
  currency_required: 'Нужен исходный фрагмент валюты.',
  unsupported_account_currency: 'Этот импорт поддерживает только дебетовый RUB-счёт.',
}
export function statementIssueLabel(code: string) {
  const parts = code.split(':')
  const message = issues[parts.at(-1) ?? ''] ?? 'Проверьте исходные поля строки и их назначение.'
  const row = parts[0].match(/^row_(\d+)$/)
  return row ? `Строка ${row[1]}: ${message}` : message
}
