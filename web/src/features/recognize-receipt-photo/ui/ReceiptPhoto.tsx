import { useEffect, useRef, useState } from 'react'

import type { components } from '@/shared/api/generated/schema'
import { formatMinorText } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { Notice } from '@/shared/ui/status'
import {
  cancelReceipt,
  getReceiptDraft,
  loadReceiptContext,
  previewReceipt,
  saveReceiptEdits,
  setReceiptConsent,
  uploadReceiptPhoto,
  type AIMode,
  type ReceiptDraft,
  type ReceiptDraftFields,
  type ReceiptDraftPreview,
} from '../api/receipt-drafts'
import { useConfirmReceiptDraft } from '../api/use-confirm-receipt-draft'
import { ReceiptReading } from './ReceiptReading'
import { ReceiptDraftEditor } from './ReceiptDraftEditor'

type Account = components['schemas']['Account']
type Category = components['schemas']['Category']

export function ReceiptPhoto() {
  const [mode, setMode] = useState<AIMode>()
  const [draft, setDraft] = useState<ReceiptDraft>()
  const [fields, setFields] = useState<ReceiptDraftFields>()
  const [preview, setPreview] = useState<ReceiptDraftPreview>()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const confirmKey = useRef<string>('')
  const confirmation = useConfirmReceiptDraft()
  useEffect(() => {
    let active = true
    void loadReceiptContext()
      .then(([cap, acc, cat]) => {
        if (!active) return
        if (cap.data) setMode(cap.data.receipt)
        else setError('Не удалось проверить доступность распознавания чеков. Обновите страницу.')
        if (acc.data) setAccounts(acc.data.items)
        if (cat.data) setCategories(cat.data.items)
        const id = new URLSearchParams(location.search).get('receiptDraft')
        if (id)
          void getReceiptDraft(id)
            .then((data) => {
              if (active && data) accept(data)
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
  function accept(next: ReceiptDraft) {
    setDraft(next)
    setFields(next.fields)
    setPreview(undefined)
  }
  function change<K extends keyof ReceiptDraftFields>(key: K, value: ReceiptDraftFields[K]) {
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
    await setReceiptConsent(mode.policyHash, grant)
    setMode((current) => (current ? { ...current, consentGranted: grant } : current))
  }
  async function upload(file: File) {
    const uploaded = await uploadReceiptPhoto(file)
    accept(uploaded)
    confirmKey.current = ''
    const url = new URL(location.href)
    url.searchParams.set('receiptDraft', uploaded.id)
    history.replaceState(null, '', url)
  }
  async function prepare() {
    if (!draft || !fields) return
    const edited = await saveReceiptEdits(draft, fields)
    accept(edited)
    setPreview(await previewReceipt(edited))
    confirmKey.current = crypto.randomUUID()
  }
  async function confirm() {
    if (!draft || !preview) return
    accept(await confirmation.mutateAsync({ draftId: draft.id, preview, key: confirmKey.current }))
  }
  async function cancel() {
    if (!draft) return
    accept(await cancelReceipt(draft.id))
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
        <CardTitle>Расход по фото или PDF чека</CardTitle>
        <CardDescription>
          Проверь распознанные данные. Расход появится после отдельного подтверждения.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {!mode.enabled && (
          <p className="text-sm text-muted-foreground">
            Распознавание недоступно. Можно загрузить источник и заполнить чек вручную.
          </p>
        )}
        {mode.enabled && (
          <>
            <p className="text-sm">
              {mode.model} · {mode.provider} через OpenRouter.
            </p>
            <p className="text-sm">
              {mode.dataDescription} {mode.privacy}
            </p>
            {mode.consentGranted && (
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => void run(() => consent(!mode.consentGranted))}
              >
                Отозвать согласие на распознавание
              </Button>
            )}
          </>
        )}
        <div>
          <Label htmlFor="receipt-photo">Фото или PDF чека</Label>
          <Input
            id="receipt-photo"
            type="file"
            accept="image/jpeg,image/png,image/webp,application/pdf"
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
        <div className="grid items-start gap-4 xl:grid-cols-2">
          {editable && draft && (
            <ReceiptReading
              draft={draft}
              available={mode.enabled}
              busy={busy}
              onChanged={accept}
              onBusyChange={setBusy}
            />
          )}
          {editable && fields && (
            <ReceiptDraftEditor
              fields={fields}
              change={change}
              accounts={accounts}
              categories={categories}
              busy={busy}
              onPrepare={() => void run(prepare)}
              onCancel={() => void run(cancel)}
            />
          )}
        </div>
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
