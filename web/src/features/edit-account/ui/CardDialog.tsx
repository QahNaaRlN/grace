import { useState, type FormEvent } from 'react'

import type { PaymentCard } from '@/entities/account'
import type { Account } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Notice } from '@/shared/ui/status'
import { useSaveCard } from '../api/account-mutations'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

export function CardDialog({
  account,
  card,
  onClose,
}: {
  account: Account
  card: PaymentCard | null
  onClose: () => void
}) {
  const save = useSaveCard(account.id, card)
  const [form, setForm] = useState({
    name: card?.name ?? 'Карта',
    bank: card?.bank ?? '',
    last4: card?.last4 ?? '',
  })
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    if (pending) return
    if (!/^\d{4}$/.test(form.last4)) return setError('Последние четыре цифры — ровно 4 цифры.')
    setPending(true)
    setError('')
    const body = { name: form.name.trim(), bank: form.bank.trim(), last4: form.last4 }
    const result = await save.mutateAsync(body)
    setPending(false)
    if (!result.data) return setError(errorText(result.error, 'Не удалось сохранить карту.'))
    onClose()
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{card ? 'Изменить карту' : 'Добавить карту'}</DialogTitle>
          <DialogDescription>Карта — способ платить со счёта «{account.name}».</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <Field label="Банк">
            <Input
              value={form.bank}
              maxLength={160}
              required
              onChange={(e) => setForm({ ...form, bank: e.target.value })}
            />
          </Field>
          <Field label="Последние четыре цифры">
            <Input
              value={form.last4}
              inputMode="numeric"
              maxLength={4}
              required
              onChange={(e) => setForm({ ...form, last4: e.target.value.replace(/\D/g, '') })}
            />
          </Field>
          <Field label="Название">
            <Input
              value={form.name}
              maxLength={160}
              required
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </Field>
          {error && (
            <Notice tone="danger" title="Не сохранено">
              {error}
            </Notice>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              Отмена
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? 'Сохраняем…' : 'Сохранить'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
