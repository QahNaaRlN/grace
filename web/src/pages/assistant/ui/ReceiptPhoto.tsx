import { Notice } from '@/shared/ui/status'
import { useEffect, useRef, useState } from 'react'
import type { components } from '@/shared/api/generated/schema'
import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { ungroupAmount } from '@/shared/lib/amount-text'
import { Label } from '@/shared/ui/label'
import { Textarea } from '@/shared/ui/textarea'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { accountLabel } from '@/shared/lib/finance-display'
import { formatMinorText } from '@/shared/lib/money'
import { LoadingSkeleton } from '@/shared/ui/skeleton'

type Draft = components['schemas']['ReceiptDraft']
type Fields = components['schemas']['ReceiptDraftFields']
type Preview = components['schemas']['ReceiptDraftPreview']
type Account = components['schemas']['Account']
type Category = components['schemas']['Category']
type Mode = components['schemas']['AIMode']
const emptyItem = {
  name: null,
  quantity: null,
  unitPrice: null,
  lineTotal: null,
  discount: null,
  brand: null,
  package: null,
  broadGroup: null,
}
export function ReceiptPhoto() {
  const [mode, setMode] = useState<Mode>()
  const [draft, setDraft] = useState<Draft>()
  const [fields, setFields] = useState<Fields>()
  const [preview, setPreview] = useState<Preview>()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const extractKey = useRef<string>('')
  const confirmKey = useRef<string>('')
  useEffect(() => {
    let active = true
    void Promise.all([
      apiClient.GET('/api/assistant/capabilities'),
      apiClient.GET('/api/v1/accounts'),
      apiClient.GET('/api/v1/categories'),
    ])
      .then(([cap, acc, cat]) => {
        if (!active) return
        if (cap.data) setMode(cap.data.receipt)
        else setError('Не удалось проверить доступность распознавания чеков. Обновите страницу.')
        if (acc.data) setAccounts(acc.data.items)
        if (cat.data) setCategories(cat.data.items)
        const id = new URLSearchParams(location.search).get('receiptDraft')
        if (id)
          void apiClient
            .GET('/api/receipt-drafts/{draftId}', { params: { path: { draftId: id } } })
            .then((result) => {
              if (active && result.data) accept(result.data)
            })
            .catch(() => {})
      })
      .catch(() => {
        if (active)
          setError('Не удалось загрузить сведения о распознавании чеков. Обновите страницу.')
      })
    return () => {
      active = false
    }
  }, [])
  function accept(next: Draft) {
    setDraft(next)
    setFields(next.fields)
    setPreview(undefined)
  }
  function change<K extends keyof Fields>(key: K, value: Fields[K]) {
    setFields((current) => (current ? { ...current, [key]: value } : current))
    setPreview(undefined)
  }
  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError('')
    try {
      await action()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось обработать чек.')
    } finally {
      setBusy(false)
    }
  }
  async function consent(grant: boolean) {
    if (!mode?.policyHash) return
    const result = grant
      ? await apiClient.POST('/api/assistant/consents', {
          params: { header: { 'X-CSRF-Token': csrfToken } },
          body: { scope: 'receipt', policyHash: mode.policyHash },
        })
      : await apiClient.DELETE('/api/assistant/consents/{scope}', {
          params: { path: { scope: 'receipt' }, header: { 'X-CSRF-Token': csrfToken } },
        })
    if (result.error) throw new Error('Согласие не сохранено. Обнови страницу.')
    setMode((current) => (current ? { ...current, consentGranted: grant } : current))
  }
  async function upload(file: File) {
    const body = new FormData()
    body.set('photo', file)
    // multipart is sent without a JSON Content-Type; the browser supplies its boundary.
    const result = await apiClient.POST('/api/receipt-drafts', {
      params: { header: { 'X-CSRF-Token': csrfToken } },
      body: { photo: file as unknown as string },
      bodySerializer: () => body,
    })
    if (!result.data)
      throw new Error('Не удалось загрузить фото. Используй JPEG, PNG или WebP до 8 МБ.')
    accept(result.data)
    extractKey.current = ''
    confirmKey.current = ''
    const url = new URL(location.href)
    url.searchParams.set('receiptDraft', result.data.id)
    history.replaceState(null, '', url)
  }
  async function extract() {
    if (!draft) return
    if (!extractKey.current) extractKey.current = crypto.randomUUID()
    const result = await apiClient.POST('/api/receipt-drafts/{draftId}/extract', {
      params: {
        path: { draftId: draft.id },
        header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': extractKey.current },
      },
    })
    if (!result.data)
      throw new Error(
        'Распознавание не завершено. Проверь согласие и лимит. Повтор той же попытки не создаёт новый платный запрос.',
      )
    accept(result.data)
  }
  async function prepare() {
    if (!draft || !fields) return
    const edited = await apiClient.PATCH('/api/receipt-drafts/{draftId}', {
      params: { path: { draftId: draft.id }, header: { 'X-CSRF-Token': csrfToken } },
      body: { revision: draft.revision, fields },
    })
    if (!edited.data)
      throw new Error('Не удалось сохранить правки. Открой актуальный черновик и повтори.')
    accept(edited.data)
    const result = await apiClient.POST('/api/receipt-drafts/{draftId}/previews', {
      params: { path: { draftId: draft.id }, header: { 'X-CSRF-Token': csrfToken } },
      body: { revision: edited.data.revision },
    })
    if (!result.data)
      throw new Error(
        'Предварительный просмотр недоступен. Проверь сумму, дату, счёт и все позиции.',
      )
    setPreview(result.data)
    confirmKey.current = crypto.randomUUID()
  }
  async function confirm() {
    if (!draft || !preview) return
    const result = await apiClient.POST('/api/receipt-drafts/{draftId}/confirmations', {
      params: {
        path: { draftId: draft.id },
        header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': confirmKey.current },
      },
      body: { previewId: preview.id, hash: preview.hash },
    })
    if (!result.data)
      throw new Error(
        'Подтверждение не получено. Повтори с тем же просмотром; при изменении книги нужен новый просмотр.',
      )
    accept(result.data)
  }
  async function cancel() {
    if (!draft) return
    const result = await apiClient.DELETE('/api/receipt-drafts/{draftId}', {
      params: { path: { draftId: draft.id }, header: { 'X-CSRF-Token': csrfToken } },
    })
    if (!result.data) throw new Error('Отмена не завершена.')
    accept(result.data)
  }
  if (!mode)
    return error ? (
      <p role="alert" className="text-sm text-destructive">
        {error}
      </p>
    ) : (
      <LoadingSkeleton label="Проверяем распознавание чеков…" />
    )
  const editable =
    draft && !['confirmed', 'cancelled', 'expired', 'extracting'].includes(draft.status)
  return (
    <Card>
      <CardHeader>
        <CardTitle>Расход по фото чека</CardTitle>
        <CardDescription>
          Проверь распознанные данные. Расход появится после отдельного подтверждения.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {!mode.enabled && (
          <p className="text-sm text-muted-foreground">Распознавание чеков пока недоступно.</p>
        )}
        {mode.enabled && (
          <>
            <p className="text-sm">
              {mode.model} · {mode.provider} через OpenRouter.
            </p>
            <p className="text-sm">
              {mode.dataDescription} {mode.privacy}
            </p>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => void run(() => consent(!mode.consentGranted))}
            >
              {mode.consentGranted
                ? 'Отозвать согласие на распознавание'
                : 'Разрешить передачу фото для распознавания'}
            </Button>
            <div>
              <Label htmlFor="receipt-photo">Фото чека</Label>
              <Input
                id="receipt-photo"
                type="file"
                accept="image/jpeg,image/png,image/webp"
                capture="environment"
                disabled={busy}
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (file) void run(() => upload(file))
                }}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              Фото хранится временно: до подтверждения, отмены или истечения суток. Загрузка не
              отправляет фото провайдеру.
            </p>
          </>
        )}
        {draft && (
          <p className="text-sm">
            {draft.status === 'confirmed'
              ? 'Расход подтверждён.'
              : draft.status === 'cancelled'
                ? 'Черновик отменён.'
                : 'Черновик — финансовые данные ещё не изменены.'}{' '}
            {draft.photoStatus === 'deleted'
              ? 'Фото удалено.'
              : draft.photoStatus === 'pending_delete'
                ? 'Фото ожидает удаления.'
                : ''}
          </p>
        )}
        {editable && fields && (
          <fieldset disabled={busy} className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={!mode.consentGranted || !mode.enabled}
                onClick={() => void run(extract)}
              >
                Распознать чек
              </Button>
              <Button
                variant="outline"
                disabled={!mode.consentGranted || !mode.enabled}
                onClick={() => {
                  extractKey.current = crypto.randomUUID()
                  void run(extract)
                }}
              >
                Новая платная попытка
              </Button>
            </div>
            <div>
              <Label htmlFor="receipt-seller">Магазин</Label>
              <Input
                id="receipt-seller"
                value={fields.seller ?? ''}
                onChange={(e) => change('seller', e.target.value || null)}
              />
            </div>
            <div>
              <Label htmlFor="receipt-date">Дата чека · время неизвестно</Label>
              <Input
                id="receipt-date"
                type="date"
                value={fields.issuedDate ?? ''}
                onChange={(e) => {
                  change('issuedDate', e.target.value || null)
                  change('issuedAt', null)
                }}
              />
            </div>
            <div>
              <Label htmlFor="receipt-currency">Валюта чека</Label>
              <ChoiceSelect
                id="receipt-currency"
                value={fields.currency ?? ''}
                onValueChange={(value) => change('currency', value || null)}
              >
                <option value="">Не определена</option>
                <option value="RUB">Российский рубль · RUB</option>
                {fields.currency && fields.currency !== 'RUB' && (
                  <option value={fields.currency} disabled>
                    {fields.currency} · пока не поддерживается
                  </option>
                )}
              </ChoiceSelect>
            </div>
            <div>
              <Label htmlFor="receipt-total">Итог</Label>
              <MoneyInput
                id="receipt-total"
                currency={fields.currency ?? undefined}
                value={fields.total ?? ''}
                onChange={(e) => change('total', ungroupAmount(e.target.value) || null)}
              />
            </div>
            <div>
              <Label htmlFor="receipt-account">Счёт оплаты</Label>
              <ChoiceSelect
                id="receipt-account"
                value={fields.accountId}
                onValueChange={(value) => change('accountId', value)}
              >
                <option value="">Выбери счёт</option>
                {accounts
                  .filter((a) => a.currency === 'RUB')
                  .map((a) => (
                    <option key={a.id} value={a.id}>
                      {accountLabel(a)}
                    </option>
                  ))}
              </ChoiceSelect>
            </div>
            <div>
              <Label htmlFor="receipt-category">Категория</Label>
              <ChoiceSelect
                id="receipt-category"
                value={fields.categoryId ?? ''}
                onValueChange={(value) => change('categoryId', value || null)}
              >
                <option value="">Выбери категорию расхода</option>
                {categories
                  .filter((c) => c.kind === 'expense' && (!c.hidden || c.id === fields.categoryId))
                  .map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
              </ChoiceSelect>
            </div>
            <p className="text-sm font-medium">Позиции · неизвестные поля заполни по чеку</p>
            {fields.items.map((item, index) => (
              <div key={index} className="grid gap-2 rounded-lg border p-3 sm:grid-cols-2">
                {(
                  [
                    'name',
                    'quantity',
                    'unitPrice',
                    'lineTotal',
                    'discount',
                    'brand',
                    'package',
                    'broadGroup',
                  ] as const
                ).map((key, i) => (
                  <div key={key}>
                    <Label htmlFor={`receipt-${index}-${key}`}>
                      {
                        [
                          'Название',
                          'Количество',
                          'Цена',
                          'Итог позиции',
                          'Скидка позиции',
                          'Бренд',
                          'Упаковка',
                          'Группа товара',
                        ][i]
                      }
                    </Label>
                    {key === 'unitPrice' || key === 'lineTotal' || key === 'discount' ? (
                      <MoneyInput
                        id={`receipt-${index}-${key}`}
                        currency="RUB"
                        value={item[key] ?? ''}
                        onChange={(e) =>
                          change(
                            'items',
                            fields.items.map((v, j) =>
                              j === index
                                ? { ...v, [key]: ungroupAmount(e.target.value) || null }
                                : v,
                            ),
                          )
                        }
                      />
                    ) : (
                      <Input
                        id={`receipt-${index}-${key}`}
                        inputMode={key === 'quantity' ? 'decimal' : undefined}
                        value={item[key] ?? ''}
                        onChange={(e) =>
                          change(
                            'items',
                            fields.items.map((v, j) =>
                              j === index ? { ...v, [key]: e.target.value || null } : v,
                            ),
                          )
                        }
                      />
                    )}
                  </div>
                ))}
                <Button
                  variant="ghost"
                  onClick={() =>
                    change(
                      'items',
                      fields.items.filter((_, j) => j !== index),
                    )
                  }
                >
                  Удалить позицию
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              disabled={fields.items.length >= 50}
              onClick={() => change('items', [...fields.items, { ...emptyItem }])}
            >
              Добавить позицию
            </Button>
            <div>
              <Label htmlFor="receipt-action">Как сохранить</Label>
              <ChoiceSelect
                id="receipt-action"
                value={fields.action}
                onValueChange={(value) => change('action', value)}
              >
                <option value="">Выбери действие</option>
                <option value="create">Создать новый расход</option>
                <option value="link_existing">Добавить чек к существующей покупке</option>
              </ChoiceSelect>
            </div>
            {fields.action === 'link_existing' && (
              <p className="text-sm">
                Выбери покупку из совпадений ниже. Для поиска сначала покажи результат с действием
                «Создать новый расход».
              </p>
            )}
            <div>
              <Label htmlFor="receipt-candidate-reason">
                Если похожая покупка уже есть, почему нужен новый расход
              </Label>
              <Textarea
                id="receipt-candidate-reason"
                value={fields.candidateReason}
                onChange={(e) => change('candidateReason', e.target.value)}
              />
            </div>
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => void run(prepare)}>Проверить и показать результат</Button>
              <Button variant="ghost" onClick={() => void run(cancel)}>
                Отменить черновик
              </Button>
            </div>
          </fieldset>
        )}
        {preview && (
          <div className="space-y-3 rounded-lg border p-4">
            <p className="font-medium">
              {preview.action === 'create'
                ? `Будет создан один расход на ${formatMinorText(preview.amountMinor, '₽')}.`
                : 'Чек будет добавлен к покупке. Баланс не изменится.'}
            </p>
            <p className="text-sm">
              {fields?.seller} · {fields?.issuedDate ?? fields?.issuedAt} ·{' '}
              {accounts.find((a) => a.id === preview.accountId)?.name}
            </p>
            {preview.candidates.length > 0 && (
              <div>
                <p className="font-medium">Возможные совпадения</p>
                {preview.candidates.map((c) => (
                  <div key={c.eventId} className="space-y-2 py-2">
                    <p className="text-sm">
                      {c.description}
                      {c.hasReceipt ? ' · чек уже есть' : ''}
                    </p>
                    <Button
                      disabled={busy || c.hasReceipt}
                      variant="outline"
                      onClick={() => {
                        change('action', 'link_existing')
                        change('existingEventId', c.eventId)
                      }}
                    >
                      Добавить чек к этой покупке
                    </Button>
                  </div>
                ))}
              </div>
            )}
            {preview.warnings.length > 0 && (
              <Notice tone="warning" title="Есть похожие покупки">
                Проверь совпадения и укажи причину создания отдельного расхода.
              </Notice>
            )}
            <Button
              disabled={busy || preview.warnings.length > 0}
              onClick={() => void run(confirm)}
            >
              {preview.action === 'create' ? 'Подтвердить расход' : 'Добавить чек к покупке'}
            </Button>
          </div>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
