import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import type { components } from '@/shared/api/generated/schema'
import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Textarea } from '@/shared/ui/textarea'

type Capabilities = components['schemas']['AICapabilities']
type Message = components['schemas']['AIConversationMessage']
function amount(value: string, currency: string) {
  const n = BigInt(value), positive = n < 0n ? -n : n
  return `${n < 0n ? '−' : ''}${(positive / 100n).toLocaleString('ru-RU')},${(positive % 100n).toString().padStart(2, '0')} ${currency}`
}
function dollars(value: string) {
  const n = BigInt(value)
  return `$${n / 1000000n}.${(n % 1000000n).toString().padStart(6, '0')}`
}
export function AIAnalysis() {
  const [params, setParams] = useSearchParams()
  const initialConversation = useRef(params.get('conversation') || '')
  const [capabilities, setCapabilities] = useState<Capabilities>()
  const [month, setMonth] = useState(() => new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Yekaterinburg', year: 'numeric', month: '2-digit' }).format(new Date()))
  const [question, setQuestion] = useState('Помоги понять мои расходы за выбранный месяц.')
  const [conversation, setConversation] = useState(initialConversation.current)
  const [messages, setMessages] = useState<Message[]>([])
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const request = useRef<{ body: string, key: string } | undefined>(undefined)
  async function refresh() {
    const result = await apiClient.GET('/api/assistant/capabilities')
    if (result.data) setCapabilities(result.data)
  }
  useEffect(() => {
    let active = true
    void apiClient.GET('/api/assistant/capabilities').then(result => { if (active && result.data) setCapabilities(result.data) }).catch(() => {})
    if (initialConversation.current) void apiClient.GET('/api/assistant/conversations/{conversationId}', { params: { path: { conversationId: initialConversation.current } } }).then(result => { if (active && result.data) setMessages(result.data.messages) }).catch(() => {})
    return () => { active = false }
  }, [])
  async function consent(grant: boolean) {
    if (!capabilities) return
    setPending(true); setError('')
    try {
      const result = grant
        ? await apiClient.POST('/api/assistant/consents', { params: { header: { 'X-CSRF-Token': csrfToken } }, body: { scope: 'analysis', policyHash: capabilities.analysis.policyHash! } })
        : await apiClient.DELETE('/api/assistant/consents/{scope}', { params: { path: { scope: 'analysis' }, header: { 'X-CSRF-Token': csrfToken } } })
      if (result.error) throw new Error('Не удалось изменить согласие. Обнови страницу и проверь выбранного провайдера.')
      await refresh()
    } catch (e) { setError(e instanceof Error ? e.message : 'Согласие не сохранено.') }
    finally { setPending(false) }
  }
  async function submit(e: FormEvent) {
    e.preventDefault(); setPending(true); setError('')
    try {
      let id = conversation
      if (!id) {
        const created = await apiClient.POST('/api/assistant/conversations', { params: { header: { 'X-CSRF-Token': csrfToken } } })
        if (!created.data) throw new Error('Не удалось открыть диалог.')
        id = created.data.id; setConversation(id)
        const nextParams = new URLSearchParams(params); nextParams.set('conversation', id); setParams(nextParams, { replace: true })
      }
      const body = JSON.stringify({ question, month })
      if (request.current?.body !== body) request.current = { body, key: crypto.randomUUID() }
      const result = await apiClient.POST('/api/assistant/conversations/{conversationId}/messages', { params: { path: { conversationId: id }, header: { 'X-CSRF-Token': csrfToken, 'Idempotency-Key': request.current!.key } }, body: { question, month } })
      if (!result.data) throw new Error('Ответ пока не получен. Можно повторить запрос: сохранённый ответ откроется без повторного платного вызова.')
      setMessages(current => [...current.filter(m => m.id !== result.data!.id), result.data!])
      if (result.data.status === 'completed') request.current = undefined
      await refresh()
    } catch (e) { setError(e instanceof Error ? e.message : 'Анализ сейчас недоступен.') }
    finally { setPending(false) }
  }
  if (!capabilities) return null
  const mode = capabilities.analysis
  return <Card>
    <CardHeader><CardTitle>Разобраться в расходах</CardTitle><CardDescription>Grace считает. ИИ помогает понять и обсудить результат.</CardDescription></CardHeader>
    <CardContent className="space-y-4">
      {!mode.enabled && <p className="text-sm text-muted-foreground">ИИ-анализ пока недоступен. <Link className="underline" to="/health">Обычные отчёты Grace</Link> продолжают работать.</p>}
      {mode.enabled && <>
        <p className="text-sm">Модель {mode.model} · {mode.provider} через OpenRouter. Доступно для эксперимента: {dollars(capabilities.remainingMicros)}.</p>
        {!mode.consentGranted ? <div className="space-y-3 rounded-lg border p-4">
          <p className="font-medium">Что получит ИИ</p><p className="text-sm">{mode.dataDescription}</p><p className="text-sm text-muted-foreground">{mode.privacy}</p>
          <Button disabled={pending} onClick={() => void consent(true)}>Разрешить передачу для анализа</Button>
        </div> : <form className="space-y-3" onSubmit={submit}>
          <div><Label htmlFor="ai-month">Месяц для анализа</Label><Input id="ai-month" type="month" value={month} required onChange={e => setMonth(e.target.value)} /></div>
          <div><Label htmlFor="ai-question">Что хочешь понять</Label><Textarea id="ai-question" value={question} maxLength={500} required onChange={e => setQuestion(e.target.value)} /></div>
          <p className="text-xs text-muted-foreground">Только подтверждённые операции. Строки импорта, ожидающие сверки, в расчёт не входят. Запрос оплачивается из общего лимита эксперимента.</p>
          <Button type="submit" disabled={pending}>{pending ? 'Анализируем…' : 'Обсудить расходы'}</Button>
        </form>}
      </>}
      {mode.consentGranted && <Button type="button" variant="ghost" disabled={pending} onClick={() => void consent(false)}>Отозвать согласие</Button>}
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      {messages.map(message => <article key={message.id} className="space-y-3 rounded-lg border p-4">
        <h3 className="font-semibold">{message.month} · {message.question}</h3>
        <p className="text-sm text-muted-foreground">Расчёт Grace по подтверждённым операциям. Полнота истории неизвестна.</p>
        <dl className="space-y-2">{message.facts.facts.map(fact => <div id={`ai-${message.id}-${fact.id}`} key={fact.id} className="flex flex-wrap items-start justify-between gap-2 text-sm"><dt className="min-w-0 break-words">{fact.label}</dt><dd className="font-medium">{amount(fact.valueMinor, fact.currency)}</dd></div>)}</dl>
        {message.status === 'failed' && <p role="alert" className="text-sm">ИИ не смог завершить объяснение. Расчёт Grace сохранён; финансовые данные не изменены.</p>}
        {message.response && <div className="space-y-3 border-t pt-3"><p className="text-xs text-muted-foreground">{message.response.notice}</p>{message.response.interpretations.map((item, i) => <div key={i}><p className="text-xs font-medium text-muted-foreground">{item.kind === 'assumption' ? 'Возможная интерпретация' : 'Что можно проверить'}</p><p className="text-sm">{item.text}</p><div className="flex flex-wrap gap-2 text-xs">{item.factIds.map(id => <a key={id} className="underline" href={`#ai-${message.id}-${id}`}>{message.facts.facts.find(f => f.id === id)?.label}</a>)}</div></div>)}<p className="text-xs text-muted-foreground">Стоимость запроса: {dollars(message.response.costMicros)}.</p></div>}
      </article>)}
    </CardContent>
  </Card>
}
