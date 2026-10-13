import { Disclosure } from '@/shared/ui/disclosure'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { PageHeader } from '@/shared/ui/page-header'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Textarea } from '@/shared/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { AppLayout } from '@/shared/ui/layout/AppLayout'
import { formatMoney, eventTimeText, accountLabel, minorToInput, parseMinor } from '@/shared/lib/finance-display'
import { MoneyInput } from '@/shared/ui/money-input'
import { PreviewPanel } from './PreviewPanel'
import { MonthPanel } from './MonthPanel'

type Record = components['schemas']['ImportRecord']
type Decision = components['schemas']['ImportDecision']
type Fields = components['schemas']['ImportFields']
type Preview = components['schemas']['ImportPreview']
type Receipt = components['schemas']['ImportReceipt']
const names: { [key: string]: string } = { unresolved: 'Не проверено', create: 'Новая операция', link_existing: 'Связать с операцией', duplicate_of: 'Дубль другой строки', exclude: 'Исключить', published: 'Подтверждено', purchase: 'Покупка', income_received: 'Доход', debt_payment: 'Погашение', internal_transfer: 'Перевод между счетами', refund: 'Возврат' }
function message(error: unknown) { return error instanceof Error ? error.message : 'Не удалось выполнить действие. Обновите страницу и повторите.' }

export function ImportsPage() {
  const cache = useQueryClient()
  const [params, setParams] = useSearchParams()
  const batchId = params.get('batch') ?? ''
  const [batchCursor, setBatchCursor] = useState('')
  const [cursor, setCursor] = useState('')
  const [status, setStatus] = useState('')
  const [domain, setDomain] = useState('')
  const [selected, setSelected] = useState<string[]>([])
  const [recordId, setRecordId] = useState('')
  const [file, setFile] = useState<File>()
  const [sourceId, setSourceId] = useState('Финансовое здоровье')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [preview, setPreview] = useState<Preview>()
  const [receipt, setReceipt] = useState<Receipt>()
  const [confirmationKey, setConfirmationKey] = useState('')
  const batches = useQuery({ queryKey: ['imports', 'batches', batchCursor], queryFn: async () => {
    const { data } = await apiClient.GET('/api/imports/batches', { params: { query: { cursor: batchCursor || undefined } } }); if (!data) throw new Error('Не удалось загрузить черновики.'); return data
  } })
  const batch = useQuery({ queryKey: ['imports', 'batch', batchId], enabled: !!batchId, queryFn: async () => {
    const { data } = await apiClient.GET('/api/imports/batches/{batchId}', { params: { path: { batchId } } }); if (!data) throw new Error('Черновик недоступен.'); return data
  } })
  const records = useQuery({ queryKey: ['imports', 'records', batchId, cursor, status, domain], enabled: !!batchId, queryFn: async () => {
    const { data } = await apiClient.GET('/api/imports/batches/{batchId}/records', { params: { path: { batchId }, query: { cursor: cursor || undefined, status: status || undefined, domain: domain || undefined } } }); if (!data) throw new Error('Не удалось загрузить строки.'); return data
  } })
  const detail = useQuery({ queryKey: ['imports', 'record', batchId, recordId], enabled: !!batchId && !!recordId, queryFn: async () => {
    const { data } = await apiClient.GET('/api/imports/batches/{batchId}/records/{recordId}', { params: { path: { batchId, recordId } } }); if (!data) throw new Error('Строка недоступна.'); return data
  } })
  const receiptId = params.get('confirmation') ?? ''
  const savedReceipt = useQuery({ queryKey: ['imports', 'receipt', batchId, receiptId], enabled: !!batchId && !!receiptId, queryFn: async () => {
    const { data } = await apiClient.GET('/api/imports/batches/{batchId}/confirmations/{confirmationId}', { params: { path: { batchId, confirmationId: receiptId } } }); if (!data) throw new Error('Подтверждение недоступно.'); return data
  } })
  const shownReceipt = receipt || savedReceipt.data
  function openBatch(id: string) {
    setParams({ batch: id }); setCursor(''); setSelected([]); setRecordId(''); setPreview(undefined); setReceipt(undefined); setError('')
  }
  async function action(run: () => Promise<void>) {
    if (pending) return
    setPending(true); setError('')
    try { await run() } catch (err) { setError(message(err)) } finally { setPending(false) }
  }
  async function upload() {
    if (!file) return
    if (file.name.toLowerCase().endsWith('.csv')) {
      const { data, error: failure } = await apiClient.POST('/api/imports/csv', { body: '', bodySerializer: () => file, headers: { 'Content-Type': 'text/csv' }, params: { query: { sourceId }, header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': crypto.randomUUID() } } })
      if (!data) throw new Error(failure?.code === 'invalid_csv' ? 'Не удалось прочитать CSV. Экспортируйте полный лист «Операции» в UTF-8 с исходными строками и столбцами A:O.' : failure?.message ?? 'CSV не сохранён.')
      await cache.invalidateQueries({ queryKey: ['imports'] }); openBatch(data.id); setNotice('Сохранён черновик CSV. Выберите месяц и проверьте строки перед подтверждением.'); return
    }
    // Pass original bytes: parsing/re-serializing JSON would round source decimals.
    const { data, error: failure } = await apiClient.POST('/api/imports/batches', { body: {} as components['schemas']['ImportSnapshot'], bodySerializer: () => file, params: { header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': crypto.randomUUID() } } })
    if (!data) throw new Error(failure?.message ?? 'Снимок не сохранён. Проверьте формат и размер файла.')
    await cache.invalidateQueries({ queryKey: ['imports'] }); openBatch(data.id); setNotice('Сохранён черновик. Финансовые операции ещё не созданы.')
  }
  async function makePreview() {
    if (!batch.data) return
    const { data, error: failure } = await apiClient.POST('/api/imports/batches/{batchId}/previews', { params: { path: { batchId }, header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': crypto.randomUUID() } }, body: { expectedRevision: batch.data.revision, recordIds: selected } })
    if (!data) throw new Error(failure?.code === 'revision_conflict' ? 'Черновик изменился. Обновите страницу.' : 'Не удалось подготовить предварительный просмотр.')
    setPreview(data); setReceipt(undefined); setConfirmationKey(crypto.randomUUID())
  }
  async function confirm() {
    if (!preview) return
    const { data, error: failure } = await apiClient.POST('/api/imports/batches/{batchId}/confirmations', { params: { path: { batchId }, header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': confirmationKey } }, body: { previewId: preview.id, previewHash: preview.hash } })
    if (!data) {
      if (['preview_expired', 'preview_changed', 'revision_conflict'].includes(failure?.code ?? '')) { setPreview(undefined); await cache.invalidateQueries({ queryKey: ['imports'] }); throw new Error('Предварительный просмотр устарел. Проверьте строки и получите новый.') }
      throw new Error('Подтверждение не завершено. Можно повторить действие: повторная запись защищена.')
    }
    const nextParams = new URLSearchParams(params); nextParams.set('batch', batchId); nextParams.set('confirmation', data.id)
    setReceipt(data); setParams(nextParams); setPreview(undefined); setSelected([]); setNotice('Решения подтверждены. Оставшиеся строки можно проверить позже.')
    await Promise.all(['imports', 'accounts', 'financial-events', 'period-summary', 'cash-position'].map(key => cache.invalidateQueries({ queryKey: [key] })))
  }
  return <AppLayout><div className="space-y-6">
    <PageHeader title="Импорт" description="Загрузите таблицу, проверьте строки и подтвердите результат." />
    {error && <p role="alert" className="text-destructive">{error}</p>}
    {notice && <p role="status">{notice}</p>}
    {(batches.isPending || (batchId && (batch.isPending || records.isPending))) && <LoadingSkeleton label="Загружаем черновики…" />}
    <Card><CardHeader><CardTitle>Новый черновик</CardTitle></CardHeader><CardContent className="space-y-3">
      <Label htmlFor="snapshot">Таблица CSV или снимок JSON (до 16 MiB)</Label><Input id="snapshot" type="file" accept=".csv,.json,text/csv,application/json" onChange={event => setFile(event.target.files?.[0])} />
      <Disclosure className="rounded-xl border p-3 text-sm" triggerClassName="min-h-11 content-center font-medium" title={<>Как подготовить файл</>}><p className="mt-2 leading-relaxed text-muted-foreground">Для CSV экспортируйте полный лист «Операции» из «Финансового здоровья»: UTF-8, столбцы A:O, заголовки в строке 4, данные с строки 5. CSV сохраняет значения; для формул и других листов используйте снимок JSON.</p></Disclosure>
      {file?.name.toLowerCase().endsWith('.csv') && <div><Label htmlFor="csv-source">Название источника</Label><Input id="csv-source" maxLength={500} value={sourceId} onChange={event => setSourceId(event.target.value)} /></div>}
      <Button disabled={pending || !file || file.name.toLowerCase().endsWith('.csv') && !sourceId.trim()} onClick={() => void action(upload)}>Сохранить черновик</Button>
    </CardContent></Card>
    <div className="flex flex-wrap gap-2" aria-label="Сохранённые черновики">
      {batches.data?.items.map(item => <Button key={item.id} variant={item.id === batchId ? 'default' : 'outline'} onClick={() => openBatch(item.id)}>{item.source.sourceId} · {item.status === 'completed' ? 'завершён' : 'на проверке'}</Button>)}
      {batches.data?.nextCursor && <Button variant="outline" onClick={() => setBatchCursor(batches.data!.nextCursor!)}>Другие черновики</Button>}
      {batchCursor && <Button variant="outline" onClick={() => setBatchCursor('')}>Первые черновики</Button>}
    </div>
    {(batches.isError || batch.isError || records.isError || detail.isError) && <p role="alert">Не удалось загрузить данные. <Button variant="outline" onClick={() => void cache.invalidateQueries({ queryKey: ['imports'] })}>Повторить</Button></p>}
    {batch.data && <>
      <h2 className="text-xl font-semibold">{batch.data.source.sourceId}</h2>
      <MonthPanel batchId={batchId} onOpenRecord={setRecordId} />
      <p>Операций по статусам: {Object.entries(batch.data.counts).map(([k, v]) => `${names[k] ?? k}: ${v}`).join(' · ')}</p>
      <p className="text-sm text-muted-foreground">Другие листы сохранены как свидетельства. Их данные не публикуются в финансовый журнал.</p>
      <div className="grid gap-4 rounded-2xl border border-border/70 bg-card p-5 sm:grid-cols-2" role="group" aria-label="Фильтры строк">
        <div className="space-y-2"><Label htmlFor="review-filter">Показать строки</Label><ChoiceSelect id="review-filter" value={status} onValueChange={value => { setStatus(value); setCursor('') }}><option value="">Все</option><option value="needs_review">Совпадения и вопросы в исходнике</option>{['unresolved', 'create', 'link_existing', 'duplicate_of', 'exclude', 'published'].map(s => <option key={s} value={s}>{names[s]}</option>)}</ChoiceSelect></div>
        <div className="space-y-2"><Label htmlFor="domain-filter">Листы</Label><ChoiceSelect id="domain-filter" value={domain} onValueChange={value => { setDomain(value); setCursor('') }}><option value="">Все</option><option value="transaction">Финансовые операции</option><option value="unsupported">Свидетельства без публикации</option></ChoiceSelect></div>
      </div>
      <div className="grid gap-3 lg:grid-cols-2">{records.data?.items.map(row => <Card key={row.id}><CardContent className="space-y-2 pt-4">
        <div className="flex items-center gap-3"><label className="grid size-11 shrink-0 cursor-pointer place-items-center"><Checkbox className="size-5"  aria-label={`Выбрать строку ${row.source.row}`} disabled={row.published || row.source.domain !== 'transaction' || pending} checked={selected.includes(row.id)} onCheckedChange={checked => { setSelected(checked ? [...selected, row.id] : selected.filter(id => id !== row.id)); setPreview(undefined) }} /></label>
          <Button variant="ghost" onClick={() => setRecordId(row.id)}>{row.source.sheet}, строка {row.source.row}</Button></div>
        <p>{names[row.source.normalized.kind ?? ''] ?? row.source.domain} · {row.source.normalized.date?.local} · {Number.isSafeInteger(row.source.normalized.amountMinor) && /^[A-Z]{3}$/.test(row.source.normalized.currency ?? '') ? formatMoney(row.source.normalized.amountMinor!, row.source.normalized.currency!) : 'Точная сумма — в исходнике'}</p>
        <p>{row.source.normalized.description}</p><p>{names[row.published ? 'published' : row.decision.action]}</p>
        {row.source.issues.length > 0 && <p className="text-sm text-muted-foreground">Вопросов в исходнике: {row.source.issues.length}</p>}
      </CardContent></Card>)}</div>
      <div className="flex flex-wrap gap-2">{records.data?.nextCursor && <Button variant="outline" onClick={() => setCursor(records.data!.nextCursor!)}>Следующие строки</Button>}{cursor && <Button variant="outline" onClick={() => setCursor('')}>Первые строки</Button>}</div>
      {detail.data && <ReviewForm key={`${detail.data.id}:${JSON.stringify(detail.data.decision)}`} record={detail.data} batchId={batchId} revision={batch.data.revision} onSaved={async () => { setPreview(undefined); await cache.invalidateQueries({ queryKey: ['imports'] }); setNotice('Решение сохранено. Финансовый журнал ещё не изменён.') }} />}
      <Button disabled={pending || selected.length === 0 || selected.length > 100} onClick={() => void action(makePreview)}>Предварительный просмотр ({selected.length})</Button>
      {preview && <PreviewPanel preview={preview} pending={pending} onConfirm={() => void action(confirm)} />}
      {shownReceipt && <Card><CardHeader><CardTitle>Подтверждение сохранено</CardTitle></CardHeader><CardContent><ul className="space-y-3">{shownReceipt.outcomes.map(o => <li key={o.recordId}>{names[o.action]} {o.event && <Disclosure id={`event-${o.eventId}`} className="rounded-md border p-3" triggerClassName="cursor-pointer py-2" title={<>{names[o.event.kind]}: {o.event.merchant || o.event.description} · {eventTimeText(o.event)}</>}><p>{o.event.description}</p>{o.event.postings.map(p => <p key={p.id}>{p.accountName}: {p.direction === 'debit' ? 'Списание' : 'Зачисление'} {p.amountMinor} коп. {p.currency}</p>)}<Link className="underline" to={`/imports?batch=${batchId}&confirmation=${shownReceipt.id}#event-${o.eventId}`}>Ссылка на подтверждённую операцию</Link></Disclosure>}</li>)}</ul></CardContent></Card>}
    </>}
  </div></AppLayout>
}

function initialFields(row: Record): Fields {
  if (row.decision.action === 'create') return row.decision.fields
  const n = row.source.normalized
  const dateOnly = n.date?.precision === 'date' || n.date?.precision === 'serial_fraction'
  return { kind: (['purchase', 'income_received', 'debt_payment', 'internal_transfer', 'refund'].includes(n.kind ?? '') ? n.kind : 'purchase') as Fields['kind'], amountMinor: Number.isSafeInteger(n.amountMinor) ? n.amountMinor! : 0, currency: n.currency || 'RUB', accountId: n.accountId || '', categoryId: n.categoryId || undefined, occurredAt: dateOnly ? undefined : n.date?.utc || '', occurredDate: dateOnly ? n.date!.local.slice(0, 10) : undefined, merchant: n.recipient || '', description: n.description || '', timeReason: '' }
}

function ReviewForm({ record, batchId, revision, onSaved }: { record: Record; batchId: string; revision: number; onSaved: () => Promise<void> }) {
  const [action, setAction] = useState<Decision['action']>(record.decision.action)
  const [fields, setFields] = useState<Fields>(() => initialFields(record))
  const [reason, setReason] = useState(record.decision.reason || '')
  const [eventId, setEventId] = useState(record.decision.action === 'link_existing' ? record.decision.eventId : '')
  const [canonicalRecordId, setCanonical] = useState(record.decision.action === 'duplicate_of' ? record.decision.canonicalRecordId : '')
  const [resolutions, setResolutions] = useState(record.decision.resolutions || [])
  const [amountText, setAmountText] = useState(() => (fields.amountMinor ? minorToInput(fields.amountMinor) : ''))
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const accounts = useQuery({ queryKey: ['accounts', 'import-options'], queryFn: async () => (await apiClient.GET('/api/v1/accounts')).data })
  const categories = useQuery({ queryKey: ['categories', 'import-options'], queryFn: async () => (await apiClient.GET('/api/v1/categories')).data })
  const events = useQuery({ queryKey: ['financial-events', 'import-options'], queryFn: async () => (await apiClient.GET('/api/v1/financial-events')).data })
  function field<K extends keyof Fields>(name: K, value: Fields[K]) { setFields({ ...fields, [name]: value }) }
  async function save(event: React.FormEvent) {
    event.preventDefault(); if (pending) return
    setPending(true); setError('')
    try {
      const common = { reason, resolutions }
      const decision: Decision = action === 'create' ? { action, fields, ...common } : action === 'link_existing' ? { action, eventId, ...common } : action === 'duplicate_of' ? { action, canonicalRecordId, ...common } : { action, ...common }
      if (action === 'create' && !(Number.isSafeInteger(fields.amountMinor) && fields.amountMinor > 0)) throw new Error('Укажите положительную сумму с точностью до копейки.')
      const { data, error: failure } = await apiClient.PATCH('/api/imports/batches/{batchId}/records/{recordId}', { params: { path: { batchId, recordId: record.id }, header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': crypto.randomUUID() } }, body: { expectedRevision: revision, decision } })
      if (!data) throw new Error(failure?.code === 'revision_conflict' ? 'Черновик изменился. Обновите страницу.' : 'Не удалось сохранить решение. Проверьте поля.')
      await onSaved()
    } catch (err) { setError(message(err)) } finally { setPending(false) }
  }
  const chooseAccount = (name: 'accountId' | 'destinationAccountId', label: string) => <div><Label htmlFor={name}>{label}</Label><ChoiceSelect id={name} value={fields[name] || ''} onValueChange={value => field(name, value)}><option value="">Выберите счёт</option>{accounts.data?.items.map(a => <option key={a.id} value={a.id}>{accountLabel(a)} ({a.currency})</option>)}</ChoiceSelect></div>
  return <Card><CardHeader><CardTitle>Проверка: {record.source.sheet}, строка {record.source.row}</CardTitle></CardHeader><CardContent className="space-y-4">
    <Disclosure triggerClassName="cursor-pointer py-3" title={<>Исходные значения и нормализация</>}><pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-3 text-xs">{JSON.stringify(record.source, null, 2)}</pre></Disclosure>
    <Disclosure triggerClassName="cursor-pointer py-3" title={<>История решений ({record.history.length})</>}>{record.history.map(h => <p key={h.revision}>{new Date(h.createdAt).toLocaleString('ru-RU')} · {names[h.decision.action]} · {h.decision.reason}</p>)}</Disclosure>
    {record.published ? <p>Эта строка уже подтверждена. Решение сохранено вместе с происхождением данных.</p> : record.source.domain !== 'transaction' ? <p>Этот лист сохранён как свидетельство. Публикация его данных появится в отдельном этапе.</p> : <form className="space-y-4" onSubmit={event => void save(event)}>
      <Label htmlFor="review-action">Решение</Label><ChoiceSelect id="review-action" value={action} onValueChange={value => setAction(value as Decision['action'])}>{['unresolved', 'create', 'link_existing', 'duplicate_of', 'exclude'].map(a => <option key={a} value={a}>{names[a]}</option>)}</ChoiceSelect>
      {action === 'create' && <div className="grid gap-3 sm:grid-cols-2">
        <div><Label htmlFor="kind">Вид операции</Label><ChoiceSelect id="kind" value={fields.kind} onValueChange={value => field('kind', value as Fields['kind'])}>{['purchase', 'income_received', 'debt_payment', 'internal_transfer', 'refund'].map(k => <option key={k} value={k}>{names[k]}</option>)}</ChoiceSelect></div>
        <div><Label htmlFor="amount">Сумма</Label><MoneyInput id="amount" currency={/^[A-Z]{3}$/.test(fields.currency) ? fields.currency : undefined} value={amountText} onChange={e => { setAmountText(e.target.value); field('amountMinor', parseMinor(e.target.value) ?? Number.NaN) }} /></div>
        <div><Label htmlFor="currency">Валюта</Label><Input id="currency" maxLength={3} value={fields.currency} onChange={e => field('currency', e.target.value.toUpperCase())} /></div>
        <div className="sm:col-span-2"><label className="flex min-h-11 items-center gap-3"><Checkbox  checked={fields.occurredDate !== undefined} onCheckedChange={checked => setFields({ ...fields, occurredDate: checked ? record.source.normalized.date?.local.slice(0, 10) || '' : undefined, occurredAt: checked ? undefined : '' })} />Известен только день, время неизвестно</label></div>
        {fields.occurredDate !== undefined ? <div><Label htmlFor="occurredDate">Дата операции</Label><Input id="occurredDate" type="date" value={fields.occurredDate} onChange={e => field('occurredDate', e.target.value)} /></div> : <><div><Label htmlFor="occurredAt">Дата и время (с часовым поясом, например +05:00)</Label><Input id="occurredAt" placeholder="2026-10-06T12:30:00+05:00" value={fields.occurredAt || ''} onChange={e => field('occurredAt', e.target.value)} /></div><div><Label htmlFor="timeReason">Основание уточнения времени</Label><Input id="timeReason" value={fields.timeReason || ''} onChange={e => field('timeReason', e.target.value)} /></div></>}
        {chooseAccount('accountId', 'Счёт операции')}
        {fields.kind === 'internal_transfer' && chooseAccount('destinationAccountId', 'Счёт получателя')}
        <div><Label htmlFor="categoryId">Категория</Label><ChoiceSelect id="categoryId" value={fields.categoryId || ''} onValueChange={value => field('categoryId', value || undefined)}><option value="">Без категории</option>{categories.data?.items.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}</ChoiceSelect></div>
        {fields.kind === 'refund' && <div><Label htmlFor="originalEventId">Исходная покупка</Label><ChoiceSelect id="originalEventId" value={fields.originalEventId || ''} onValueChange={value => field('originalEventId', value)}><option value="">Выберите покупку</option>{events.data?.items.filter(e => e.kind === 'purchase').map(e => <option key={e.id} value={e.id}>{eventTimeText(e)} · {e.merchant || e.description}</option>)}</ChoiceSelect></div>}
        <div><Label htmlFor="merchant">Получатель</Label><Input id="merchant" value={fields.merchant} onChange={e => field('merchant', e.target.value)} /></div>
        <div><Label htmlFor="description">Описание</Label><Input id="description" value={fields.description} onChange={e => field('description', e.target.value)} /></div>
      </div>}
      {action === 'link_existing' && <div><Label htmlFor="eventId">Существующая операция</Label><ChoiceSelect id="eventId" value={eventId} onValueChange={value => setEventId(value)}><option value="">Выберите операцию</option>{events.data?.items.map(e => <option key={e.id} value={e.id}>{eventTimeText(e)} · {names[e.kind]} · {e.merchant || e.description}</option>)}{record.candidates.filter(c => c.event && !events.data?.items.some(e => e.id === c.eventId)).map(c => <option key={c.id} value={c.eventId}>{eventTimeText(c.event!)} · {c.event!.description}</option>)}</ChoiceSelect></div>}
      {action === 'duplicate_of' && <div><Label htmlFor="canonical">Основная строка</Label><ChoiceSelect id="canonical" value={canonicalRecordId} onValueChange={value => setCanonical(value)}><option value="">Выберите кандидата</option>{record.candidates.filter(c => c.recordId).map(c => <option key={c.id} value={c.recordId}>{c.source?.sheet}, строка {c.source?.row}</option>)}</ChoiceSelect></div>}
      <Label htmlFor="reason">Почему принято это решение</Label><Textarea id="reason" maxLength={2000} value={reason} onChange={e => setReason(e.target.value)} />
      {record.candidates.map(c => <div key={c.id} className="space-y-2 rounded-md border p-3"><p className="font-medium">Возможное совпадение: {c.source ? `${c.source.sheet}, строка ${c.source.row}` : 'запись финансового журнала'}</p><pre className="max-h-56 overflow-auto whitespace-pre-wrap break-all text-xs">{JSON.stringify(c.source?.normalized || c.event, null, 2)}</pre><p className="text-sm">{c.differences.join(', ')}</p><Label htmlFor={`resolve-${c.id}`}>Если это разные операции, объясните различие</Label><Input id={`resolve-${c.id}`} value={resolutions.find(r => r.candidateId === c.id)?.reason || ''} onChange={e => setResolutions([...resolutions.filter(r => r.candidateId !== c.id), ...(e.target.value ? [{ candidateId: c.id, reason: e.target.value }] : [])])} /></div>)}
      {error && <p role="alert" className="text-destructive">{error}</p>}
      <Button type="submit" disabled={pending}>Сохранить решение</Button><p className="text-sm text-muted-foreground">Это сохраняет только черновик. Для записи нужен отдельный предварительный просмотр и подтверждение.</p>
    </form>}
  </CardContent></Card>
}
