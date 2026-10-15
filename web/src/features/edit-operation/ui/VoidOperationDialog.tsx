import { useState } from 'react'
import type { FinancialEvent } from '@/shared/api'
import { formatMoney } from '@/shared/lib/money'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Notice } from '@/shared/ui/status'
import { Textarea } from '@/shared/ui/textarea'
import { useVoidOperation } from '../api/operation-mutations'
import { failure } from './operation-labels'

export function VoidDialog({
  event,
  onClose,
  onVoided,
}: {
  event: FinancialEvent
  onClose: () => void
  onVoided: (next: FinancialEvent) => void
}) {
  const [key] = useState(createIdempotencyKey)
  const voiding = useVoidOperation(event, key)
  const [reason, setReason] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function confirm() {
    if (pending || !reason.trim()) return
    setPending(true)
    setError('')
    const result = await voiding.mutateAsync(reason.trim())
    setPending(false)
    if (!result.data) return setError(failure(result.error, 'Не удалось отменить операцию.'))
    onVoided(result.data)
  }
  const posting = event.postings[0]
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Отменить операцию?</DialogTitle>
          <DialogDescription>
            {posting
              ? `${formatMoney(posting.amountMinor, posting.currency)} перестанут учитываться в остатках и статистике.`
              : 'Операция перестанет учитываться.'}{' '}
            Запись и история сохранятся; вернуть отменённую операцию нельзя.
          </DialogDescription>
        </DialogHeader>
        <Field label="Причина">
          <Textarea
            value={reason}
            maxLength={500}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Например, дубль из выписки"
          />
        </Field>
        {error && (
          <Notice tone="danger" title="Не отменена">
            {error}
          </Notice>
        )}
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={pending}>
            Не отменять
          </Button>
          <Button
            variant="destructive-solid"
            disabled={pending || !reason.trim()}
            onClick={() => void confirm()}
          >
            {pending ? 'Отменяем…' : 'Отменить операцию'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
