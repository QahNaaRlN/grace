import { useState } from 'react'

import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { Disclosure } from '@/shared/ui/disclosure'
import { useStatementAI } from '../api/ai-mutations'

type Statement = components['schemas']['BankStatement']
export function StatementAI({ statement }: { statement: Statement }) {
  const { preview, extract } = useStatementAI(statement.id)
  const [consent, setConsent] = useState(false)
  const [key, setKey] = useState('')
  const [error, setError] = useState('')
  const p = preview.data
  if (statement.deletedAt || statement.publishedRows > 0) return null
  async function prepare() {
    setError('')
    setConsent(false)
    setKey(crypto.randomUUID())
    try {
      await preview.mutateAsync(statement.revision)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Предпросмотр недоступен.')
    }
  }
  async function send() {
    if (!p || !consent) return
    try {
      await extract.mutateAsync({ preview: p, key })
      setError('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Распознавание не завершено.')
    }
  }
  return (
    <div className="space-y-3">
      <h3 className="font-medium">AI-предложение структуры</h3>
      <p className="text-sm text-muted-foreground">
        Сначала проверьте состав данных и стоимость. Подтверждение передачи данных отдельно от
        подтверждения денежных операций.
      </p>
      <Button variant="outline" onClick={prepare} disabled={preview.isPending || extract.isPending}>
        Подготовить AI-распознавание
      </Button>
      {p && (
        <>
          <p className="break-words text-sm">
            Модель: {p.plan.policy.model}. Провайдер: {p.plan.policy.providerName} (
            {p.plan.policy.provider}). До {p.plan.chunks.length} вызовов, максимум $
            {(p.plan.maximumMicros / 1000000).toFixed(4)} из общего бюджета.
          </p>
          <p className="text-sm">
            Передаётся только показанный текст и расположение фрагментов. PDF и изображения остаются
            в Grace. Суммы и даты могут быть личными даже после скрытия имён и номеров.
          </p>
          <Disclosure title="Проверить передаваемый текст">
            <div className="max-h-80 overflow-auto rounded border p-3 text-xs">
              {p.plan.chunks.map((c) => (
                <section key={c.index} className="mb-3">
                  <p>Часть {c.index}</p>
                  <p className="whitespace-pre-wrap break-words">
                    {[...c.context, ...c.spans].map((span) => span.text).join(' ')}
                  </p>
                </section>
              ))}
            </div>
          </Disclosure>
          <label className="flex min-h-11 items-center gap-3">
            <Checkbox checked={consent} onCheckedChange={(v) => setConsent(v === true)} />
            Разрешаю передачу показанных данных выбранному провайдеру в пределах указанной стоимости
          </label>
          <Button disabled={!consent || extract.isPending} onClick={send}>
            {extract.isPending ? 'Распознавание…' : 'Передать данные и подготовить черновик'}
          </Button>
          {extract.data && (
            <p role="status">
              {extract.data.status === 'completed'
                ? 'Предложение сохранено. Нужна проверка страниц, полей и строк.'
                : 'Попытка не завершила черновик. Проверьте статус и используйте ручную подготовку; автоматического повтора нет.'}
            </p>
          )}
        </>
      )}
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
