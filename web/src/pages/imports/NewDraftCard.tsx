import { useRef, useState } from 'react'
import { FileText } from 'lucide-react'

import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Disclosure } from '@/shared/ui/disclosure'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Notice } from '@/shared/ui/status'

const maxBytes = 16 * 1024 * 1024
const fileLabel = 'Таблица CSV или снимок JSON (до 16 MiB)'

// SPEC-0038: «выбран» ≠ «сохраняется» ≠ «сохранён»; every state is in this card.
type State =
  | { step: 'empty' }
  | { step: 'invalid'; reason: string }
  | { step: 'selected' }
  | { step: 'saving' }
  | { step: 'saved' }
  | { step: 'failed'; message: string }

function kindOf(file: File) {
  const name = file.name.toLowerCase()
  return name.endsWith('.csv') ? 'csv' : name.endsWith('.json') ? 'json' : null
}

function sizeText(bytes: number) {
  const format = (value: number) =>
    new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 }).format(value)
  if (bytes >= 1024 * 1024) return `${format(bytes / (1024 * 1024))} МБ`
  return `${format(Math.max(bytes / 1024, 0.1))} КБ`
}

export function NewDraftCard({ onSaved }: { onSaved: (batchId: string) => Promise<void> | void }) {
  const input = useRef<HTMLInputElement>(null)
  const working = useRef(false)
  const [file, setFile] = useState<File>()
  const [state, setState] = useState<State>({ step: 'empty' })
  const [sourceId, setSourceId] = useState('Финансовое здоровье')
  // One key per chosen file: «Повторить» resends the same bytes with the same key.
  const [idempotencyKey, setIdempotencyKey] = useState('')
  const kind = file ? kindOf(file) : null
  const saving = state.step === 'saving'

  function choose(next: File | undefined) {
    if (!next) return // the picker was cancelled: keep the current choice
    setFile(next)
    setIdempotencyKey(crypto.randomUUID())
    if (!kindOf(next)) setState({ step: 'invalid', reason: 'Подходят только файлы CSV или JSON.' })
    else if (next.size === 0) setState({ step: 'invalid', reason: 'Файл пустой.' })
    else if (next.size > maxBytes)
      setState({ step: 'invalid', reason: `Файл ${sizeText(next.size)} — больше 16 МБ.` })
    else setState({ step: 'selected' })
  }

  function clear() {
    setFile(undefined)
    setState({ step: 'empty' })
    if (input.current) input.current.value = ''
  }

  async function save() {
    if (!file || !kind || working.current) return
    working.current = true
    setState({ step: 'saving' })
    try {
      const header = { 'Idempotency-Key': idempotencyKey }
      if (kind === 'csv') {
        const { data, error } = await apiClient.POST('/api/imports/csv', {
          body: '',
          bodySerializer: () => file,
          headers: { 'Content-Type': 'text/csv' },
          params: { query: { sourceId, fileName: file.name }, header },
        })
        if (!data)
          throw new Error(
            error?.code === 'invalid_csv'
              ? 'Не удалось прочитать CSV. Экспортируйте полный лист «Операции» в UTF-8 с исходными строками и столбцами A:O.'
              : (error?.message ?? 'CSV не сохранён.'),
          )
        await onSaved(data.id)
      } else {
        // Pass original bytes: parsing/re-serializing JSON would round source decimals.
        const { data, error } = await apiClient.POST('/api/imports/batches', {
          body: {} as components['schemas']['ImportSnapshot'],
          bodySerializer: () => file,
          params: { header, query: { fileName: file.name } },
        })
        if (!data)
          throw new Error(error?.message ?? 'Снимок не сохранён. Проверьте формат и размер файла.')
        await onSaved(data.id)
      }
      setState({ step: 'saved' })
    } catch (error) {
      setState({
        step: 'failed',
        message: error instanceof Error ? error.message : 'Черновик не сохранён.',
      })
    } finally {
      working.current = false
    }
  }

  const status =
    state.step === 'saving'
      ? 'Сохраняем черновик на сервере…'
      : state.step === 'selected'
        ? 'Файл выбран — готов к сохранению'
        : state.step === 'saved'
          ? 'Черновик сохранён'
          : state.step === 'failed'
            ? 'Не сохранено'
            : ''
  const ready =
    !!file && !!kind && state.step !== 'invalid' && !(kind === 'csv' && !sourceId.trim())

  return (
    <Card>
      <CardHeader>
        <CardTitle>Новый черновик</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Field label={fileLabel} error={state.step === 'invalid' ? state.reason : undefined}>
          <Input
            ref={input}
            id="snapshot"
            type="file"
            accept=".csv,.json,text/csv,application/json"
            disabled={saving}
            className={file ? 'sr-only' : undefined}
            tabIndex={file ? -1 : undefined}
            onChange={(event) => choose(event.target.files?.[0])}
          />
        </Field>
        {file && (
          <div
            data-slot="selected-file"
            className="flex flex-wrap items-center gap-3 rounded-xl border border-border p-3"
          >
            <FileText aria-hidden="true" className="size-5 shrink-0 text-primary" />
            <div className="min-w-0 flex-1">
              <p className="break-words font-medium">{file.name}</p>
              <p className="text-sm text-muted-foreground">
                {kind === 'csv'
                  ? 'Таблица CSV'
                  : kind === 'json'
                    ? 'Снимок JSON'
                    : 'Неподходящий тип'}{' '}
                · {sizeText(file.size)}
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={saving}
                onClick={() => input.current?.click()}
              >
                Заменить файл
              </Button>
              <Button type="button" variant="ghost" disabled={saving} onClick={clear}>
                Убрать
              </Button>
            </div>
          </div>
        )}
        <p role="status" aria-live="polite" className={status ? 'text-sm' : 'sr-only'}>
          {status}
        </p>
        <Disclosure
          className="rounded-xl border p-3 text-sm"
          triggerClassName="min-h-11 content-center font-medium"
          title={<>Как подготовить файл</>}
        >
          <p className="mt-2 leading-relaxed text-muted-foreground">
            Для CSV экспортируйте полный лист «Операции» из «Финансового здоровья»: UTF-8, столбцы
            A:O, заголовки в строке 4, данные с строки 5. CSV сохраняет значения; для формул и
            других листов используйте снимок JSON.
          </p>
        </Disclosure>
        {kind === 'csv' && (
          <Field label="Название источника">
            <Input
              id="csv-source"
              maxLength={500}
              value={sourceId}
              disabled={saving}
              onChange={(event) => setSourceId(event.target.value)}
            />
          </Field>
        )}
        <Button disabled={saving || !ready} onClick={() => void save()}>
          {saving ? 'Сохраняем…' : 'Сохранить черновик'}
        </Button>
        {state.step === 'saved' && (
          <Notice tone="success" title="Черновик сохранён">
            Финансовые операции ещё не созданы: проверьте строки и подтвердите их ниже.
          </Notice>
        )}
        {state.step === 'failed' && (
          <Notice
            tone="danger"
            title="Черновик не сохранён"
            action={
              <Button variant="outline" onClick={() => void save()}>
                Повторить
              </Button>
            }
          >
            {state.message}
          </Notice>
        )}
      </CardContent>
    </Card>
  )
}
