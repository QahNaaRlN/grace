import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

import { accountQueries } from '@/entities/account'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import { importErrorText, importLabels } from '@/entities/import-batch'
import type { components } from '@/shared/api/generated/schema'
import { APP_UTC_OFFSET } from '@/shared/config'
import { eventTimeText } from '@/shared/lib/date'
import { accountLabel } from '@/shared/lib/finance-display'
import { minorToInput, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { CurrencySelect } from '@/shared/ui/currency-select'
import { Disclosure } from '@/shared/ui/disclosure'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { Textarea } from '@/shared/ui/textarea'
import { useSaveImportDecision, type ImportDecision as Decision } from '../api/save-import-decision'

type Record = components['schemas']['ImportRecord']
type Fields = components['schemas']['ImportFields']

function initialFields(row: Record): Fields {
  if (row.decision.action === 'create') return row.decision.fields
  const n = row.source.normalized
  const dateOnly = n.date?.precision === 'date' || n.date?.precision === 'serial_fraction'
  return {
    kind: (['purchase', 'income_received', 'debt_payment', 'internal_transfer', 'refund'].includes(
      n.kind ?? '',
    )
      ? n.kind
      : 'purchase') as Fields['kind'],
    amountMinor: Number.isSafeInteger(n.amountMinor) ? n.amountMinor! : 0,
    currency: n.currency || 'RUB',
    accountId: n.accountId || '',
    categoryId: n.categoryId || undefined,
    occurredAt: dateOnly ? undefined : n.date?.utc || '',
    occurredDate: dateOnly ? n.date!.local.slice(0, 10) : undefined,
    merchant: n.recipient || '',
    description: n.description || '',
    timeReason: '',
  }
}

export function ReviewForm({
  record,
  batchId,
  revision,
  onSaved,
}: {
  record: Record
  batchId: string
  revision: number
  onSaved: () => Promise<void>
}) {
  const [action, setAction] = useState<Decision['action']>(record.decision.action)
  const [fields, setFields] = useState<Fields>(() => initialFields(record))
  const [reason, setReason] = useState(record.decision.reason || '')
  const [eventId, setEventId] = useState(
    record.decision.action === 'link_existing' ? record.decision.eventId : '',
  )
  const [canonicalRecordId, setCanonical] = useState(
    record.decision.action === 'duplicate_of' ? record.decision.canonicalRecordId : '',
  )
  const [resolutions, setResolutions] = useState(record.decision.resolutions || [])
  const [amountText, setAmountText] = useState(() =>
    fields.amountMinor ? minorToInput(fields.amountMinor) : '',
  )
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const saveDecision = useSaveImportDecision(batchId)
  const accounts = useQuery(accountQueries.importOptions())
  const categories = useQuery(categoryQueries.importOptions())
  const events = useQuery(financialEventQueries.importOptions())
  function field<K extends keyof Fields>(name: K, value: Fields[K]) {
    setFields({ ...fields, [name]: value })
  }
  async function save(event: React.FormEvent) {
    event.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    try {
      const common = { reason, resolutions }
      const decision: Decision =
        action === 'create'
          ? { action, fields, ...common }
          : action === 'link_existing'
            ? { action, eventId, ...common }
            : action === 'duplicate_of'
              ? { action, canonicalRecordId, ...common }
              : { action, ...common }
      if (
        action === 'create' &&
        !(Number.isSafeInteger(fields.amountMinor) && fields.amountMinor > 0)
      )
        throw new Error('Укажите положительную сумму с точностью до копейки.')
      await saveDecision.mutateAsync({ recordId: record.id, revision, decision })
      await onSaved()
    } catch (err) {
      setError(importErrorText(err))
    } finally {
      setPending(false)
    }
  }
  const chooseAccount = (name: 'accountId' | 'destinationAccountId', label: string) => (
    <div>
      <Label htmlFor={name}>{label}</Label>
      <ChoiceSelect
        id={name}
        value={fields[name] || ''}
        onValueChange={(value) => field(name, value)}
      >
        <option value="">Выберите счёт</option>
        {accounts.data?.items.map((a) => (
          <option key={a.id} value={a.id}>
            {accountLabel(a)} ({a.currency})
          </option>
        ))}
      </ChoiceSelect>
    </div>
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          Проверка: {record.source.sheet}, строка {record.source.row}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Disclosure
          triggerClassName="cursor-pointer py-3"
          title={<>Исходные значения и нормализация</>}
        >
          <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-3 text-xs">
            {JSON.stringify(record.source, null, 2)}
          </pre>
        </Disclosure>
        <Disclosure
          triggerClassName="cursor-pointer py-3"
          title={<>История решений ({record.history.length})</>}
        >
          {record.history.map((h) => (
            <p key={h.revision}>
              {new Date(h.createdAt).toLocaleString('ru-RU')} · {importLabels[h.decision.action]} ·{' '}
              {h.decision.reason}
            </p>
          ))}
        </Disclosure>
        {record.published ? (
          <p>Эта строка уже подтверждена. Решение сохранено вместе с происхождением данных.</p>
        ) : record.source.domain !== 'transaction' ? (
          <p>
            Этот лист сохранён как свидетельство. Публикация его данных появится в отдельном этапе.
          </p>
        ) : (
          <form className="space-y-4" onSubmit={(event) => void save(event)}>
            <Label htmlFor="review-action">Решение</Label>
            <ChoiceSelect
              id="review-action"
              value={action}
              onValueChange={(value) => setAction(value as Decision['action'])}
            >
              {['unresolved', 'create', 'link_existing', 'duplicate_of', 'exclude'].map((a) => (
                <option key={a} value={a}>
                  {importLabels[a]}
                </option>
              ))}
            </ChoiceSelect>
            {action === 'create' && (
              <div className="grid gap-3 sm:grid-cols-2">
                <div>
                  <Label htmlFor="kind">Вид операции</Label>
                  <ChoiceSelect
                    id="kind"
                    value={fields.kind}
                    onValueChange={(value) => field('kind', value as Fields['kind'])}
                  >
                    {[
                      'purchase',
                      'income_received',
                      'debt_payment',
                      'internal_transfer',
                      'refund',
                    ].map((k) => (
                      <option key={k} value={k}>
                        {importLabels[k]}
                      </option>
                    ))}
                  </ChoiceSelect>
                </div>
                <div>
                  <Label htmlFor="amount">Сумма</Label>
                  <MoneyInput
                    id="amount"
                    currency={/^[A-Z]{3}$/.test(fields.currency) ? fields.currency : undefined}
                    value={amountText}
                    onChange={(e) => {
                      setAmountText(e.target.value)
                      field('amountMinor', parseMinor(e.target.value) ?? Number.NaN)
                    }}
                  />
                </div>
                <div>
                  <Label htmlFor="currency">Валюта</Label>
                  <CurrencySelect
                    id="currency"
                    value={fields.currency}
                    onValueChange={(value) => field('currency', value)}
                  />
                </div>
                <div className="sm:col-span-2">
                  <label className="flex min-h-11 items-center gap-3">
                    <Checkbox
                      checked={fields.occurredDate !== undefined}
                      onCheckedChange={(checked) =>
                        setFields({
                          ...fields,
                          occurredDate: checked
                            ? record.source.normalized.date?.local.slice(0, 10) || ''
                            : undefined,
                          occurredAt: checked ? undefined : '',
                        })
                      }
                    />
                    Известен только день, время неизвестно
                  </label>
                </div>
                {fields.occurredDate !== undefined ? (
                  <div>
                    <Label htmlFor="occurredDate">Дата операции</Label>
                    <Input
                      id="occurredDate"
                      type="date"
                      value={fields.occurredDate}
                      onChange={(e) => field('occurredDate', e.target.value)}
                    />
                  </div>
                ) : (
                  <>
                    <div>
                      <Label htmlFor="occurredAt">
                        Дата и время (с часовым поясом, например {APP_UTC_OFFSET})
                      </Label>
                      <Input
                        id="occurredAt"
                        placeholder={`2026-10-06T12:30:00${APP_UTC_OFFSET}`}
                        value={fields.occurredAt || ''}
                        onChange={(e) => field('occurredAt', e.target.value)}
                      />
                    </div>
                    <div>
                      <Label htmlFor="timeReason">Основание уточнения времени</Label>
                      <Input
                        id="timeReason"
                        value={fields.timeReason || ''}
                        onChange={(e) => field('timeReason', e.target.value)}
                      />
                    </div>
                  </>
                )}
                {chooseAccount('accountId', 'Счёт операции')}
                {fields.kind === 'internal_transfer' &&
                  chooseAccount('destinationAccountId', 'Счёт получателя')}
                <div>
                  <Label htmlFor="categoryId">Категория</Label>
                  <ChoiceSelect
                    id="categoryId"
                    value={fields.categoryId || ''}
                    onValueChange={(value) => field('categoryId', value || undefined)}
                  >
                    <option value="">Без категории</option>
                    {categories.data?.items
                      .filter((c) => !c.hidden || c.id === fields.categoryId)
                      .map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                  </ChoiceSelect>
                </div>
                {fields.kind === 'refund' && (
                  <div>
                    <Label htmlFor="originalEventId">Исходная покупка</Label>
                    <ChoiceSelect
                      id="originalEventId"
                      value={fields.originalEventId || ''}
                      onValueChange={(value) => field('originalEventId', value)}
                    >
                      <option value="">Выберите покупку</option>
                      {events.data?.items
                        .filter((e) => e.kind === 'purchase')
                        .map((e) => (
                          <option key={e.id} value={e.id}>
                            {eventTimeText(e)} · {e.merchant || e.description}
                          </option>
                        ))}
                    </ChoiceSelect>
                  </div>
                )}
                <div>
                  <Label htmlFor="merchant">Получатель</Label>
                  <Input
                    id="merchant"
                    value={fields.merchant}
                    onChange={(e) => field('merchant', e.target.value)}
                  />
                </div>
                <div>
                  <Label htmlFor="description">Описание</Label>
                  <Input
                    id="description"
                    value={fields.description}
                    onChange={(e) => field('description', e.target.value)}
                  />
                </div>
              </div>
            )}
            {action === 'link_existing' && (
              <div>
                <Label htmlFor="eventId">Существующая операция</Label>
                <ChoiceSelect
                  id="eventId"
                  value={eventId}
                  onValueChange={(value) => setEventId(value)}
                >
                  <option value="">Выберите операцию</option>
                  {events.data?.items.map((e) => (
                    <option key={e.id} value={e.id}>
                      {eventTimeText(e)} · {importLabels[e.kind]} · {e.merchant || e.description}
                    </option>
                  ))}
                  {record.candidates
                    .filter((c) => c.event && !events.data?.items.some((e) => e.id === c.eventId))
                    .map((c) => (
                      <option key={c.id} value={c.eventId}>
                        {eventTimeText(c.event!)} · {c.event!.description}
                      </option>
                    ))}
                </ChoiceSelect>
              </div>
            )}
            {action === 'duplicate_of' && (
              <div>
                <Label htmlFor="canonical">Основная строка</Label>
                <ChoiceSelect
                  id="canonical"
                  value={canonicalRecordId}
                  onValueChange={(value) => setCanonical(value)}
                >
                  <option value="">Выберите кандидата</option>
                  {record.candidates
                    .filter((c) => c.recordId)
                    .map((c) => (
                      <option key={c.id} value={c.recordId}>
                        {c.source?.sheet}, строка {c.source?.row}
                      </option>
                    ))}
                </ChoiceSelect>
              </div>
            )}
            <Label htmlFor="reason">Почему принято это решение</Label>
            <Textarea
              id="reason"
              maxLength={2000}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            {record.candidates.map((c) => (
              <div key={c.id} className="space-y-2 rounded-md border p-3">
                <p className="font-medium">
                  Возможное совпадение:{' '}
                  {c.source
                    ? `${c.source.sheet}, строка ${c.source.row}`
                    : 'запись финансового журнала'}
                </p>
                <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-all text-xs">
                  {JSON.stringify(c.source?.normalized || c.event, null, 2)}
                </pre>
                <p className="text-sm">{c.differences.join(', ')}</p>
                <Label htmlFor={`resolve-${c.id}`}>
                  Если это разные операции, объясните различие
                </Label>
                <Input
                  id={`resolve-${c.id}`}
                  value={resolutions.find((r) => r.candidateId === c.id)?.reason || ''}
                  onChange={(e) =>
                    setResolutions([
                      ...resolutions.filter((r) => r.candidateId !== c.id),
                      ...(e.target.value ? [{ candidateId: c.id, reason: e.target.value }] : []),
                    ])
                  }
                />
              </div>
            ))}
            {error && (
              <p role="alert" className="text-destructive">
                {error}
              </p>
            )}
            <Button type="submit" disabled={pending}>
              Сохранить решение
            </Button>
            <p className="text-sm text-muted-foreground">
              Это сохраняет только черновик. Для записи нужен отдельный предварительный просмотр и
              подтверждение.
            </p>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
