import { useState, useRef } from 'react'
import type { components } from '@/shared/api/generated/schema'
import { env } from '@/shared/config'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { PrivateSourceImage } from '@/shared/ui/private-source-image'
import {
  extractReceipt,
  prepareReceiptReading,
  setReceiptConsent,
  type ReceiptDraft,
} from '../api/receipt-drafts'

type Plan = components['schemas']['ReceiptReadingPlan']
export function ReceiptReading({
  draft,
  available = true,
  busy,
  onChanged,
  onBusyChange,
}: {
  draft: ReceiptDraft
  available?: boolean
  busy: boolean
  onChanged: (draft: ReceiptDraft) => void
  onBusyChange: (busy: boolean) => void
}) {
  const [plan, setPlan] = useState<Plan>()
  const [consent, setConsent] = useState(false)
  const [page, setPage] = useState(1)
  const [error, setError] = useState('')
  const key = useRef('')
  async function prepare() {
    onBusyChange(true)
    setError('')
    setConsent(false)
    try {
      const next = await prepareReceiptReading(draft.id)
      setPlan(next)
      key.current = crypto.randomUUID()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Чтение недоступно.')
    } finally {
      onBusyChange(false)
    }
  }
  async function read() {
    if (!plan || !consent) return
    onBusyChange(true)
    setError('')
    try {
      await setReceiptConsent(plan.consentPolicyHash, true, plan.pages.length)
      onChanged(await extractReceipt(draft.id, key.current, plan))
      setPlan(undefined)
      setConsent(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Чтение не завершено; проверьте состояние попытки.')
    } finally {
      onBusyChange(false)
    }
  }
  return (
    <section className="space-y-3 rounded border p-3">
      <h3 className="font-medium">Чтение изображений чека</h3>
      <Button variant="outline" disabled={busy || !available} onClick={() => void prepare()}>
        Подготовить чтение фото или PDF
      </Button>
      {plan && (
        <>
          <p>
            {plan.model} · {plan.provider} через OpenRouter. Страниц: {plan.pages.length}. Максимум
            ${(plan.maximumMicros / 1000000).toFixed(4)}.
          </p>
          <p>
            Передаются все показанные страницы и доступный текст. Проверьте изображения; финансовые
            записи появятся только после отдельного подтверждения.
          </p>
          <label className="flex min-h-11 items-center gap-2">
            <Checkbox
              checked={consent}
              disabled={busy || plan.revision !== draft.revision}
              onCheckedChange={(v) => setConsent(v === true)}
            />
            Разрешаю передачу этих страниц выбранной модели в пределах указанной стоимости
          </label>
          <Button
            disabled={busy || !consent || plan.revision !== draft.revision}
            onClick={() => void read()}
          >
            Передать страницы и распознать чек
          </Button>
          {plan.revision !== draft.revision && (
            <p role="status">Черновик изменён. Подготовьте новый просмотр.</p>
          )}
        </>
      )}
      {draft.photoStatus === 'stored' && (
        <>
          <div className="flex flex-wrap gap-2">
            {(draft.pages ?? []).map((p) => (
              <Button
                key={p.number}
                variant={page === p.number ? 'default' : 'outline'}
                onClick={() => setPage(p.number)}
              >
                Страница {p.number}
              </Button>
            ))}
          </div>
          <PrivateSourceImage
            src={`${env.VITE_API_BASE_URL}/api/receipt-drafts/${draft.id}/pages/${page}`}
            label={`Страница ${page} чека`}
          />
        </>
      )}
      {error && <p role="alert">{error}</p>}
    </section>
  )
}
