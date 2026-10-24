import { useState } from 'react'

import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { useStatementMutations } from '../api/mutations'

export function StatementUpload({
  accounts,
  onSaved,
}: {
  accounts: { id: string; name: string }[]
  onSaved: (id: string) => void
}) {
  const { upload } = useStatementMutations('')
  const [file, setFile] = useState<File>()
  const [accountId, setAccountId] = useState('')
  const [key, setKey] = useState('')
  const [error, setError] = useState('')
  async function save() {
    if (!file || !accountId) return
    setError('')
    try {
      const s = await upload.mutateAsync({ file, accountId, key })
      onSaved(s.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'PDF не сохранён.')
    }
  }
  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        PDF или фото выписки одного дебетового RUB-счёта. Загрузка обрабатывается локально. Денежные
        операции появятся после отдельной проверки и подтверждения.
      </p>
      <Field label="Счёт выписки">
        <ChoiceSelect
          value={accountId}
          onValueChange={(id) => {
            setAccountId(id)
            setKey(crypto.randomUUID())
          }}
        >
          {accounts
            .map((a) => ({ value: a.id, label: a.name }))
            .map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
        </ChoiceSelect>
      </Field>
      <Field label="PDF до 10 MiB или фото до 8 MiB">
        <Input
          type="file"
          accept="application/pdf,image/jpeg,image/png,image/webp,.pdf"
          disabled={upload.isPending}
          onChange={(e) => {
            const next = e.target.files?.[0]
            if (!next) return
            setFile(next)
            setKey(crypto.randomUUID())
            setError(next.size > 10 * 1024 * 1024 ? 'Файл больше 10 MiB.' : '')
          }}
        />
      </Field>
      {file && <p className="break-all text-sm">Выбран: {file.name}</p>}
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      <Button
        disabled={!file || !accountId || file.size > 10 * 1024 * 1024 || upload.isPending}
        onClick={save}
      >
        {upload.isPending ? 'Обрабатывается локально…' : 'Загрузить выписку'}
      </Button>
    </div>
  )
}
