import { Disclosure } from '@/shared/ui/disclosure'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { PageHeader } from '@/shared/ui/page-header'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Textarea } from '@/shared/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { AppLayout } from '@/widgets/app-shell'
import { APP_UTC_OFFSET } from '@/shared/config'
import { accountLabel } from '@/shared/lib/finance-display'
import { eventTimeText } from '@/shared/lib/date'
import { formatMoney, minorToInput, parseMinor } from '@/shared/lib/money'
import { MoneyInput } from '@/shared/ui/money-input'
import { CurrencySelect } from '@/shared/ui/currency-select'
import { PreviewPanel } from './PreviewPanel'
import { MonthPanel } from './MonthPanel'
import { NewDraftCard } from './NewDraftCard'
import { ImportHistory } from './ImportHistory'
import { RecordReview, type Selection, type View } from './RecordReview'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'

type Record = components['schemas']['ImportRecord']
type Decision = components['schemas']['ImportDecision']
type Fields = components['schemas']['ImportFields']
type Preview = components['schemas']['ImportPreview']
type Receipt = components['schemas']['ImportReceipt']
const names: { [key: string]: string } = {
  unresolved: 'Не проверено',
  create: 'Новая операция',
  link_existing: 'Связать с операцией',
  duplicate_of: 'Дубль другой строки',
  exclude: 'Исключить',
  published: 'Подтверждено',
  purchase: 'Покупка',
  income_received: 'Доход',
  debt_payment: 'Погашение',
  internal_transfer: 'Перевод между счетами',
  refund: 'Возврат',
}
function message(error: unknown) {
  return error instanceof Error
    ? error.message
    : 'Не удалось выполнить действие. Обновите страницу и повторите.'
}

export function ImportsPage() {
  const cache = useQueryClient()
  const [params, setParams] = useSearchParams()
  const batchId = params.get('batch') ?? ''
  const [view, setView] = useState<View>(initialView)
  const [selection, setSelection] = useState<Selection>({ mode: 'ids', ids: [] })
  const [recordId, setRecordId] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [preview, setPreview] = useState<Preview>()
  const [receipt, setReceipt] = useState<Receipt>()
  const [confirmationKey, setConfirmationKey] = useState('')
  const batch = useQuery({ ...importBatchQueries.batch(batchId), enabled: !!batchId })
  const detail = useQuery({
    ...importBatchQueries.record(batchId, recordId),
    enabled: !!batchId && !!recordId,
  })
  const receiptId = params.get('confirmation') ?? ''
  const savedReceipt = useQuery({
    ...importBatchQueries.receipt(batchId, receiptId),
    enabled: !!batchId && !!receiptId,
  })
  const shownReceipt = receipt || savedReceipt.data
  const selectedCount = selection.mode === 'filter' ? selection.total : selection.ids.length
  function openBatch(id: string) {
    setParams({ batch: id })
    setView(initialView)
    setSelection({ mode: 'ids', ids: [] })
    setRecordId('')
    setPreview(undefined)
    setReceipt(undefined)
    setError('')
  }
  function changeView(next: Partial<View>) {
    setView((current) => ({ ...current, ...next }))
    setPreview(undefined)
  }
  function changeSelection(next: Selection) {
    setSelection(next)
    setPreview(undefined)
  }
  async function action(run: () => Promise<void>) {
    if (pending) return
    setPending(true)
    setError('')
    try {
      await run()
    } catch (err) {
      setError(message(err))
    } finally {
      setPending(false)
    }
  }
  // SPEC-0038: the new-draft card owns its own states and messages.
  async function draftSaved(id: string) {
    await cache.invalidateQueries({ queryKey: importBatchQueries.all() })
    openBatch(id)
    setNotice('')
  }
  // SPEC-0044: walk unresolved rows in sheet order without scrolling.
  async function openNextUnresolved(after?: string) {
    const { data } = await apiClient.GET('/api/imports/batches/{batchId}/records/next-unresolved', {
      params: { path: { batchId }, query: { after: after || undefined } },
    })
    if (!data) throw new Error('Не удалось найти следующую строку.')
    if (data.record) {
      setRecordId(data.record.id)
      return true
    }
    setNotice('Нерешённых строк больше нет. Можно подготовить предварительный просмотр.')
    return false
  }
  async function makePreview() {
    if (!batch.data) return
    const filter = { domain: view.domain, status: view.status || undefined, q: view.q || undefined }
    const body =
      selection.mode === 'filter'
        ? { expectedRevision: batch.data.revision, filter }
        : { expectedRevision: batch.data.revision, recordIds: selection.ids }
    const { data, error: failure } = await apiClient.POST(
      '/api/imports/batches/{batchId}/previews',
      {
        params: {
          path: { batchId },
          header: { 'Idempotency-Key': crypto.randomUUID() },
        },
        body,
      },
    )
    if (!data)
      throw new Error(
        failure?.code === 'revision_conflict'
          ? 'Черновик изменился. Обновите страницу.'
          : failure?.code === 'selection_too_large'
            ? 'За раз можно проверить не больше 1 000 строк. Сузьте фильтр и повторите.'
            : 'Не удалось подготовить предварительный просмотр.',
      )
    setPreview(data)
    setReceipt(undefined)
    setConfirmationKey(crypto.randomUUID())
  }
  async function confirm() {
    if (!preview) return
    const { data, error: failure } = await apiClient.POST(
      '/api/imports/batches/{batchId}/confirmations',
      {
        params: {
          path: { batchId },
          header: { 'Idempotency-Key': confirmationKey },
        },
        body: { previewId: preview.id, previewHash: preview.hash },
      },
    )
    if (!data) {
      if (
        ['preview_expired', 'preview_changed', 'revision_conflict'].includes(failure?.code ?? '')
      ) {
        setPreview(undefined)
        await cache.invalidateQueries({ queryKey: importBatchQueries.all() })
        throw new Error('Предварительный просмотр устарел. Проверьте строки и получите новый.')
      }
      throw new Error(
        'Подтверждение не завершено. Можно повторить действие: повторная запись защищена.',
      )
    }
    const nextParams = new URLSearchParams(params)
    nextParams.set('batch', batchId)
    nextParams.set('confirmation', data.id)
    setReceipt(data)
    setParams(nextParams)
    setPreview(undefined)
    setSelection({ mode: 'ids', ids: [] })
    setNotice('Решения подтверждены. Оставшиеся строки можно проверить позже.')
    await Promise.all(
      [
        importBatchQueries.all(),
        accountQueries.all(),
        financialEventQueries.all(),
        periodSummaryQueries.all(),
        cashPositionQueries.all(),
      ].map((queryKey) => cache.invalidateQueries({ queryKey })),
    )
  }
  return (
    <AppLayout>
      <div className="space-y-6">
        <PageHeader
          title="Импорт"
          description="Загрузите таблицу, проверьте строки и подтвердите результат."
        />
        {error && (
          <p role="alert" className="text-destructive">
            {error}
          </p>
        )}
        {notice && <p role="status">{notice}</p>}
        <NewDraftCard onSaved={draftSaved} />
        <ImportHistory selectedId={batchId} onOpen={openBatch} />
        {batchId && batch.isPending && <LoadingSkeleton label="Загружаем черновик…" />}
        {(batch.isError || detail.isError) && (
          <p role="alert">
            Не удалось загрузить данные.{' '}
            <Button
              variant="outline"
              onClick={() => void cache.invalidateQueries({ queryKey: importBatchQueries.all() })}
            >
              Повторить
            </Button>
          </p>
        )}
        {batch.data && (
          <>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-lg font-semibold">
                {batch.data.fileName ?? batch.data.source.sourceId}
              </h2>
              <Button
                variant="outline"
                disabled={pending}
                onClick={() =>
                  void action(async () => {
                    await openNextUnresolved(recordId)
                  })
                }
              >
                Следующая нерешённая
              </Button>
            </div>
            <MonthPanel batchId={batchId} onOpenRecord={setRecordId} />
            <RecordReview
              batchId={batchId}
              batch={batch.data}
              view={view}
              onView={changeView}
              selection={selection}
              onSelection={changeSelection}
              onOpen={setRecordId}
              disabled={pending}
            />
            <Button
              disabled={
                pending || selectedCount === 0 || (selection.mode === 'ids' && selectedCount > 100)
              }
              onClick={() => void action(makePreview)}
            >
              Предварительный просмотр ({selectedCount})
            </Button>
            {preview && (
              <PreviewPanel
                preview={preview}
                pending={pending}
                onConfirm={() => void action(confirm)}
              />
            )}
            {shownReceipt && (
              <Card>
                <CardHeader>
                  <CardTitle>Подтверждение сохранено</CardTitle>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-3">
                    {shownReceipt.outcomes.map((o) => (
                      <li key={o.recordId}>
                        {names[o.action]}{' '}
                        {o.event && (
                          <Disclosure
                            id={`event-${o.eventId}`}
                            className="rounded-md border p-3"
                            triggerClassName="cursor-pointer py-2"
                            title={
                              <>
                                {names[o.event.kind]}: {o.event.merchant || o.event.description} ·{' '}
                                {eventTimeText(o.event)}
                              </>
                            }
                          >
                            <p>{o.event.description}</p>
                            {o.event.postings.map((p) => (
                              <p key={p.id}>
                                {p.accountName}:{' '}
                                {p.direction === 'debit' ? 'Списание' : 'Зачисление'}{' '}
                                {formatMoney(p.amountMinor, p.currency)}
                              </p>
                            ))}
                            <Link
                              className="underline"
                              to={`/imports?batch=${batchId}&confirmation=${shownReceipt.id}#event-${o.eventId}`}
                            >
                              Ссылка на подтверждённую операцию
                            </Link>
                          </Disclosure>
                        )}
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            )}
            {recordId && (
              <Dialog
                open
                onOpenChange={(open) => {
                  if (!open) setRecordId('')
                }}
              >
                <DialogContent
                  data-motion="menu"
                  className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(560px,calc(100%-1rem))] translate-x-0 translate-y-0 overflow-y-auto rounded-r-none sm:max-w-[560px]"
                >
                  <DialogHeader>
                    <DialogTitle>
                      {detail.data
                        ? `${detail.data.source.sheet}, строка ${detail.data.source.row}`
                        : 'Строка'}
                    </DialogTitle>
                    <DialogDescription>
                      Решение сохраняется в черновике; финансовый журнал меняется только после
                      предварительного просмотра и подтверждения.
                    </DialogDescription>
                  </DialogHeader>
                  {detail.isPending ? (
                    <LoadingSkeleton label="Загружаем строку…" />
                  ) : (
                    detail.data && (
                      <ReviewForm
                        key={`${detail.data.id}:${JSON.stringify(detail.data.decision)}`}
                        record={detail.data}
                        batchId={batchId}
                        revision={batch.data.revision}
                        onSaved={async () => {
                          const saved = recordId
                          setPreview(undefined)
                          await cache.invalidateQueries({ queryKey: importBatchQueries.all() })
                          setNotice('Решение сохранено. Финансовый журнал ещё не изменён.')
                          // After a decision, move straight to the next unresolved row.
                          try {
                            await openNextUnresolved(saved)
                          } catch {
                            /* the saved row stays open */
                          }
                        }}
                      />
                    )
                  )}
                </DialogContent>
              </Dialog>
            )}
          </>
        )}
      </div>
    </AppLayout>
  )
}

const initialView: View = {
  status: '',
  domain: 'transaction',
  q: '',
  sort: 'row',
  cursor: '',
  limit: 100,
}

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

function ReviewForm({
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
      const { data, error: failure } = await apiClient.PATCH(
        '/api/imports/batches/{batchId}/records/{recordId}',
        {
          params: {
            path: { batchId, recordId: record.id },
            header: { 'Idempotency-Key': crypto.randomUUID() },
          },
          body: { expectedRevision: revision, decision },
        },
      )
      if (!data)
        throw new Error(
          failure?.code === 'revision_conflict'
            ? 'Черновик изменился. Обновите страницу.'
            : 'Не удалось сохранить решение. Проверьте поля.',
        )
      await onSaved()
    } catch (err) {
      setError(message(err))
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
              {new Date(h.createdAt).toLocaleString('ru-RU')} · {names[h.decision.action]} ·{' '}
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
                  {names[a]}
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
                        {names[k]}
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
                      {eventTimeText(e)} · {names[e.kind]} · {e.merchant || e.description}
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
