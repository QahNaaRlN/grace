import { Disclosure } from '@/shared/ui/disclosure'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'

function blockerMessage(value: string) {
  const parts = value.split(':')
  const code = parts[0].length === 36 ? parts[1] : parts[0]
  const messages: { [key: string]: string } = {
    time_confirmation_required: 'Уточните время операции и укажите основание.',
    unresolved_candidate: 'Решите, является ли возможное совпадение дублем.',
    unresolved: 'Выберите решение для строки.',
    unsupported_domain: 'Этот лист пока сохраняется только как свидетельство.',
    unsupported_semantics: 'Смысл исходной строки пока не поддерживается.',
    borrowed_is_not_income: 'Заёмные средства нельзя публиковать как доход.',
    review_reason_required: 'Укажите основание решения.',
    canonical_not_confirmed: 'Включите основную строку в группу или подтвердите её сначала.',
    incompatible_duplicate: 'Вид операции или валюта основной строки отличаются.',
    incompatible_kind: 'Вид связанной операции отличается.',
    incompatible_currency: 'Валюта связанной операции отличается.',
    currency_mismatch: 'Валюта операции отличается от валюты счёта.',
    already_published: 'Строка уже подтверждена.',
    event_not_found: 'Связанная операция недоступна.',
    refund_group_exceeds_purchase: 'Сумма возвратов превышает остаток исходной покупки.',
    amount_overflow: 'Сумма группы слишком велика. Выберите меньшую группу.',
    invalid_financial_event: 'Проверьте счета, категории и исходную покупку для возврата.',
  }
  return messages[code] || 'Проверьте заполнение полей этой строки.'
}

export function PreviewPanel({
  preview,
  pending,
  onConfirm,
}: {
  preview: components['schemas']['ImportPreview']
  pending: boolean
  onConfirm: () => void
}) {
  const unsafe = preview.changes.some((c) => !Number.isSafeInteger(c.deltaMinor))
  const actionNames: { [key: string]: string } = {
    create: 'Создать операцию',
    link_existing: 'Связать с существующей операцией',
    duplicate_of: 'Сохранить как дубль',
    exclude: 'Исключить',
    unresolved: 'Оставить на проверке',
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>Предварительный просмотр</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>
          Новых операций: {preview.newEvents}. Связей: {preview.links}. Исключений:{' '}
          {preview.exclusions}.
        </p>
        <p>Действует до {new Date(preview.expiresAt).toLocaleTimeString('ru-RU')}.</p>
        {preview.lines.map((line) => (
          <div key={line.recordId} className="space-y-2 rounded-md border p-3">
            <p className="font-semibold">
              {line.sourceSheet}, строка {line.sourceRow}: {actionNames[line.decision.action]}
            </p>
            <p>{line.decision.reason}</p>
            {line.decision.action === 'create' && (
              <p>
                {line.decision.fields.description} ·{' '}
                {line.decision.fields.occurredDate
                  ? `${line.decision.fields.occurredDate} · время неизвестно`
                  : line.decision.fields.occurredAt}{' '}
                · {line.decision.fields.amountMinor} коп. {line.decision.fields.currency}
              </p>
            )}
            {line.blockers.length > 0 && (
              <p>Нужна проверка: {line.blockers.map(blockerMessage).join(' ')}</p>
            )}
            <Disclosure
              triggerClassName="cursor-pointer py-3"
              title={<>Свидетельства, совпадения и предупреждения</>}
            >
              <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all text-xs">
                {JSON.stringify(line, null, 2)}
              </pre>
            </Disclosure>
          </div>
        ))}
        <h3 className="font-semibold">Изменения по счетам</h3>
        {preview.changes.map((c) => (
          <p key={`${c.accountId}:${c.currency}`}>
            {c.accountName}:{' '}
            {Number.isSafeInteger(c.deltaMinor)
              ? `${c.deltaMinor} коп. ${c.currency}`
              : 'Для точного отображения выберите меньшую группу операций'}
          </p>
        ))}
        {preview.blockers.length > 0 && (
          <div role="alert">
            <p>Подтверждение заблокировано. Проверьте строки:</p>
            <ul>
              {preview.blockers.map((b) => (
                <li key={b}>{blockerMessage(b)}</li>
              ))}
            </ul>
          </div>
        )}
        <Button disabled={pending || unsafe || preview.blockers.length > 0} onClick={onConfirm}>
          {preview.newEvents
            ? `Подтвердить импорт ${preview.newEvents} операций`
            : 'Подтвердить решения'}
        </Button>
        <p className="text-sm text-muted-foreground">
          Группа сохраняется целиком. При ошибке ни одна операция из группы не будет записана.
        </p>
      </CardContent>
    </Card>
  )
}
