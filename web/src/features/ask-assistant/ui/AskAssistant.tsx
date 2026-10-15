import { Send, ShieldCheck } from 'lucide-react'
import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'

import type { AssistantResult } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Disclosure } from '@/shared/ui/disclosure'
import { Label } from '@/shared/ui/label'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { Textarea } from '@/shared/ui/textarea'
import { AssistantRequestError, queryAssistant, type SubmitAssistant } from '../api/query-assistant'
import { toLocalReferenceTime, utf8ByteLength } from '../model/submission'
import { AssistantResultView, ResultBoundary } from './AssistantResult'
import { Notice } from './result-parts'

const MAX_UTTERANCE_BYTES = 2000

function errorMessage(error: unknown) {
  if (error instanceof AssistantRequestError) {
    if (error.kind === 'validation') return 'Проверьте текст вопроса и повторите.'
    if (error.kind === 'malformed') return 'Не удалось обработать ответ. Повторите запрос позже.'
  }
  return 'Не удалось связаться с ассистентом. Проверьте соединение и повторите.'
}

/** Quick read-only questions about the user's data, answered by Grace's calculations. */
export function AskAssistant({
  submitAssistant = queryAssistant,
}: {
  submitAssistant?: SubmitAssistant
}) {
  const [utterance, setUtterance] = useState('')
  const [result, setResult] = useState<AssistantResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const controller = useRef<AbortController | null>(null)
  const trimmed = utterance.trim()
  const byteCount = utf8ByteLength(trimmed)

  useEffect(() => () => controller.current?.abort(), [])

  async function submit(event?: FormEvent) {
    event?.preventDefault()
    if (isSubmitting) return
    if (byteCount === 0) {
      setError('Введите вопрос.')
      setResult(null)
      return
    }
    if (byteCount > MAX_UTTERANCE_BYTES) {
      setError('Вопрос длиннее 2000 байт. Сократите его.')
      setResult(null)
      return
    }

    const requestController = new AbortController()
    controller.current = requestController
    setIsSubmitting(true)
    setError(null)
    setResult(null)
    try {
      const nextResult = await submitAssistant(
        {
          utterance: trimmed,
          language: 'ru',
          referenceTime: toLocalReferenceTime(new Date()),
        },
        requestController.signal,
      )
      if (!requestController.signal.aborted) setResult(nextResult)
    } catch (submissionError) {
      if (!requestController.signal.aborted) setError(errorMessage(submissionError))
    } finally {
      if (controller.current === requestController) {
        controller.current = null
        setIsSubmitting(false)
      }
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault()
      void submit()
    }
  }

  return (
    <Disclosure title="Быстрые запросы к данным Grace">
      <div className="flex items-start gap-3 rounded-2xl border border-border bg-secondary p-4 text-sm leading-relaxed text-secondary-foreground shadow-sm sm:p-5">
        <div className="grid size-9 shrink-0 place-items-center rounded-xl bg-card">
          <ShieldCheck className="size-5" aria-hidden="true" />
        </div>
        <p>
          <span className="font-semibold">Перед отправкой:</span> текст текущего вопроса, язык и
          время передаются TypeSafe. Финансовые записи и результаты инструментов провайдеру не
          отправляются.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Что вы хотите узнать?</CardTitle>
          <CardDescription>
            Например: «Покажи мои счета», «Какие обязательства актуальны?» или «Куда ушли деньги
            после последнего дохода?»
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-3" onSubmit={submit}>
            <Label htmlFor="assistant-question">Вопрос</Label>
            <Textarea
              id="assistant-question"
              value={utterance}
              onChange={(event) => setUtterance(event.target.value)}
              onKeyDown={handleKeyDown}
              aria-invalid={Boolean(error && (byteCount === 0 || byteCount > MAX_UTTERANCE_BYTES))}
              aria-describedby="assistant-hint assistant-counter"
              placeholder="Покажи мои последние чеки"
              disabled={isSubmitting}
            />
            <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
              <span id="assistant-hint">Ctrl+Enter или ⌘+Enter — отправить</span>
              <span
                id="assistant-counter"
                className={byteCount > MAX_UTTERANCE_BYTES ? 'text-destructive' : undefined}
              >
                {byteCount} / {MAX_UTTERANCE_BYTES} байт
              </span>
            </div>
            <Button className="w-full sm:w-auto" type="submit" disabled={isSubmitting}>
              <Send className="mr-2 size-4" aria-hidden="true" />
              {isSubmitting ? 'Спрашиваю…' : 'Спросить Grace'}
            </Button>
          </form>
        </CardContent>
      </Card>

      <div role="status" aria-live="polite">
        {isSubmitting && <LoadingSkeleton label="Grace выбирает безопасный источник данных…" />}
        {error && <Notice title="Не удалось выполнить запрос" text={error} />}
        {result && (
          <ResultBoundary result={result}>
            <AssistantResultView result={result} />
          </ResultBoundary>
        )}
      </div>
    </Disclosure>
  )
}
