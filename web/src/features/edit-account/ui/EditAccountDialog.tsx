import { useState, type FormEvent } from 'react'

import { accountTypeLabels } from '@/entities/account'
import type { Account } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
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
import { useUpdateAccount } from '../api/account-mutations'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

export function EditAccountDialog({ account, onClose }: { account: Account; onClose: () => void }) {
  const update = useUpdateAccount(account.id)
  const [name, setName] = useState(account.name)
  const [type, setType] = useState<string>(account.type)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function save(submit: FormEvent) {
    submit.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    const result = await update.mutateAsync({
      revision: account.revision,
      name: name.trim(),
      ...(type !== account.type ? { type: type as Account['type'] } : {}),
    })
    setPending(false)
    if (!result.data) return setError(errorText(result.error, 'Не удалось сохранить счёт.'))
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
          <DialogTitle>Сведения счёта</DialogTitle>
          <DialogDescription>Валюта счёта не меняется.</DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="space-y-4">
          <Field label="Название">
            <Input
              value={name}
              maxLength={160}
              required
              onChange={(e) => setName(e.target.value)}
            />
          </Field>
          <Field
            label="Тип"
            hint={
              account.hasOperations
                ? 'По счёту есть операции: смена типа изменила бы смысл остатка.'
                : undefined
            }
          >
            <ChoiceSelect value={type} onValueChange={setType} disabled={account.hasOperations}>
              {Object.entries(accountTypeLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </ChoiceSelect>
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
