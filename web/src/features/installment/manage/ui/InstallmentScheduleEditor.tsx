import { useState } from 'react'
import type { InstallmentDraft, InstallmentPlan } from '@/entities/installment'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { minorToInput } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Notice } from '@/shared/ui/status'
import {
  useApplyInstallmentDraft,
  useInstallmentDraft,
  useSaveInstallmentDraft,
} from '../api/use-installments'
import { InstallmentRows } from './InstallmentRows'
import { parseRows, type RowInput } from '../model/rows'

export function InstallmentScheduleEditor({ plan }: { plan: InstallmentPlan }) {
  const copy = useInstallmentDraft(plan.id)
  const [draft, setDraft] = useState<InstallmentDraft | null>(null)
  const [error, setError] = useState('')
  if (draft)
    return <DraftEditor key={draft.id} plan={plan} draft={draft} onDone={() => setDraft(null)} />
  return (
    <div className="space-y-2">
      {plan.drafts?.[0] && (
        <Button variant="outline" onClick={() => setDraft(plan.drafts[0])}>
          Продолжить сохранённую копию
        </Button>
      )}
      <Button
        variant="outline"
        disabled={copy.isPending}
        onClick={() => {
          setError('')
          void copy
            .mutateAsync(plan.revision)
            .then(setDraft)
            .catch((e: Error) => setError(e.message))
        }}
      >
        Изменить будущий график
      </Button>
      {error && (
        <Notice tone="danger" title="Копия недоступна">
          {error}
        </Notice>
      )}
    </div>
  )
}
function DraftEditor({
  plan,
  draft,
  onDone,
}: {
  plan: InstallmentPlan
  draft: InstallmentDraft
  onDone: () => void
}) {
  const save = useSaveInstallmentDraft(plan.id)
  const apply = useApplyInstallmentDraft(plan.id)
  const [rows, setRows] = useState<RowInput[]>(() =>
    draft.payments.map((p, n) => ({
      dueOn: p.dueOn,
      amount: minorToInput(p.amountMinor),
      locked: plan.payments[n]?.paidMinor > 0,
    })),
  )
  const [saved, setSaved] = useState<InstallmentDraft | null>(null)
  const [error, setError] = useState('')
  const [reviewed, setReviewed] = useState(false)
  const [saveKey, setSaveKey] = useState(createIdempotencyKey)
  const [applyKey] = useState(createIdempotencyKey)
  async function persist() {
    try {
      setError('')
      const value = await save.mutateAsync({
        draftId: draft.id,
        revision: saved?.revision ?? draft.revision,
        payments: parseRows(rows),
        key: saveKey,
      })
      setSaved(value)
      setReviewed(true)
      setSaveKey(createIdempotencyKey())
    } catch (e) {
      setError((e as Error).message)
    }
  }
  async function confirm() {
    if (!saved) return
    try {
      await apply.mutateAsync({
        draftId: saved.id,
        revision: saved.revision,
        planRevision: saved.planRevision,
        key: applyKey,
      })
      onDone()
    } catch (e) {
      setError((e as Error).message)
    }
  }
  return (
    <div className="space-y-3">
      <InstallmentRows
        rows={rows}
        onChange={(value) => {
          setRows(value)
          setReviewed(false)
        }}
        currency={plan.currency}
      />
      <p className="text-sm">
        Оплаченные строки сохраняются. Изменения вступят в силу после применения копии.
      </p>
      {saved && reviewed && (
        <Notice tone="info" title="Копия сохранена">
          Проверьте даты и суммы, затем примените график.
        </Notice>
      )}
      {error && (
        <Notice tone="danger" title="График не изменён">
          {error}
        </Notice>
      )}
      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          disabled={save.isPending || apply.isPending}
          onClick={() => void persist()}
        >
          Сохранить копию
        </Button>
        {saved && reviewed && (
          <Button disabled={apply.isPending} onClick={() => void confirm()}>
            Применить график
          </Button>
        )}
        <Button variant="ghost" onClick={onDone}>
          Закрыть копию
        </Button>
      </div>
    </div>
  )
}
