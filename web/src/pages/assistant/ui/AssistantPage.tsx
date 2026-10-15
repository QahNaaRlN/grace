import { Money } from '@/shared/ui/money'
import { PageHeader } from '@/shared/ui/page-header'
import { Disclosure } from '@/shared/ui/disclosure'
import { AIGuide } from './AIGuide'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { accountLabel } from '@/shared/lib/finance-display'
import { formatMoney } from '@/shared/lib/money'
import { AccountIcon } from '@/shared/ui/account-icon'
import { AlertCircle, CheckCircle2, Send, ShieldCheck } from 'lucide-react'
import {
  Component,
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'

import {
  AssistantRequestError,
  queryAssistant,
  toLocalReferenceTime,
  utf8ByteLength,
  type SubmitAssistant,
} from '@/features/ask-assistant'
import type {
  AssistantClarify,
  AssistantCompleted,
  AssistantResult,
  CurrencyPeriodSummary,
} from '@/shared/api'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Label } from '@/shared/ui/label'
import { AppLayout } from '@/shared/ui/layout/AppLayout'
import { Textarea } from '@/shared/ui/textarea'
import { AIAnalysis } from './AIAnalysis'
import { ReceiptPhoto } from './ReceiptPhoto'

const MAX_UTTERANCE_BYTES = 2000

const clarificationText: Record<AssistantClarify['clarificationCode'], string> = {
  time_precision_required:
    'В истории есть операции с неизвестным временем. Выберите полный календарный период вместо расчёта после точного момента.',
  low_confidence: 'Сформулируйте один конкретный вопрос о ваших финансовых данных.',
  clarification_required: 'Уточните, какие именно финансовые данные вы хотите увидеть.',
  composition_mismatch: 'Не удалось безопасно подобрать параметры. Уточните запрос.',
  arguments_required: 'Для этого расчёта не хватает обязательных параметров.',
  income_category_required:
    'Названную категорию дохода пока нельзя безопасно сопоставить. Спросите о последнем подтверждённом доходе без названия категории.',
  no_income_anchor: 'Подтверждённый доход, от которого можно построить период, не найден.',
}

type RequiredArgument = NonNullable<AssistantClarify['requiredArguments']>[number]

const argumentLabels: Record<RequiredArgument, string> = {
  from: 'дата начала',
  to: 'дата окончания',
  through: 'горизонт обязательств',
  asOf: 'дата состояния',
  limit: 'количество чеков',
  categoryId: 'категория дохода',
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(value),
  )
}

function Confidence({ value, route }: { value: number; route: string }) {
  return (
    <p className="text-xs text-muted-foreground">
      Маршрут: {route} · уверенность {Math.round(value * 100)}%
    </p>
  )
}

function PeriodCurrencies({ items }: { items: CurrencyPeriodSummary[] }) {
  if (items.length === 0) {
    return <p className="text-sm text-muted-foreground">За этот период финансовых данных нет.</p>
  }
  return (
    <div className="grid gap-3 lg:grid-cols-2">
      {items.map((item) => (
        <article key={item.currency} className="space-y-3 rounded-lg border p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h4 className="font-semibold">{item.currency}</h4>
            <Badge variant="outline">
              Итог {formatMoney(item.closingBalanceMinor, item.currency)}
            </Badge>
          </div>
          <dl className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-2 text-sm">
            <dt className="text-muted-foreground">Начальный остаток</dt>
            <dd>{formatMoney(item.openingBalanceMinor, item.currency)}</dd>
            <dt className="text-muted-foreground">Доходы</dt>
            <dd>{formatMoney(item.incomeMinor, item.currency)}</dd>
            <dt className="text-muted-foreground">Покупки</dt>
            <dd>{formatMoney(item.purchasesMinor, item.currency)}</dd>
            <dt className="text-muted-foreground">Возвраты</dt>
            <dd>{formatMoney(item.refundsMinor, item.currency)}</dd>
            <dt className="text-muted-foreground">Платежи по обязательствам</dt>
            <dd>{formatMoney(item.debtPaymentsMinor, item.currency)}</dd>
            <dt className="font-medium">Внешний денежный поток</dt>
            <dd className="font-medium">
              {formatMoney(item.netExternalCashFlowMinor, item.currency)}
            </dd>
          </dl>
        </article>
      ))}
    </div>
  )
}

function CompletedResult({ result }: { result: AssistantCompleted }) {
  switch (result.data.name) {
    case 'get_accounts':
      return (
        <ResultShell title="Ваши счета" result={result}>
          {result.data.data.items.length === 0 ? (
            <p className="text-sm text-muted-foreground">Счетов пока нет.</p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2">
              {result.data.data.items.map((account) => (
                <article key={account.id} className="rounded-lg border p-4">
                  <AccountIcon type={account.type} />
                  <p className="text-sm text-muted-foreground">{account.currency}</p>
                  <h4 className="font-semibold">{accountLabel(account)}</h4>
                  <p className="mt-2">
                    <Money valueMinor={account.balanceMinor} currency={account.currency} />
                  </p>
                </article>
              ))}
            </div>
          )}
        </ResultShell>
      )
    case 'get_obligations':
      return (
        <ResultShell title={`Обязательства на ${result.data.data.asOf}`} result={result}>
          {result.data.data.items.length === 0 ? (
            <p className="text-sm text-muted-foreground">Активных обязательств нет.</p>
          ) : (
            <div className="space-y-3">
              {result.data.data.items.map((obligation) => (
                <article key={obligation.id} className="rounded-lg border p-4">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <h4 className="font-semibold">{obligation.name}</h4>
                      <p className="text-sm text-muted-foreground">{obligation.kind}</p>
                    </div>
                    <Badge variant={obligation.status === 'completed' ? 'secondary' : 'outline'}>
                      {obligation.status === 'completed' ? 'Завершено' : 'Активно'}
                    </Badge>
                  </div>
                  <p className="mt-3 text-sm">
                    Оплачено {formatMoney(obligation.paidMinor, obligation.currency)} · осталось{' '}
                    {formatMoney(obligation.remainingMinor, obligation.currency)}
                  </p>
                  <div className="mt-3 space-y-1 border-t pt-3 text-sm">
                    {obligation.payments.map((payment) => (
                      <p key={payment.id} className="flex justify-between gap-3">
                        <span className="text-muted-foreground">
                          {payment.dueOn} · {payment.status}
                        </span>
                        <span>{formatMoney(payment.remainingMinor, obligation.currency)}</span>
                      </p>
                    ))}
                  </div>
                </article>
              ))}
            </div>
          )}
        </ResultShell>
      )
    case 'get_receipts':
      return (
        <ResultShell title="Последние чеки" result={result}>
          {result.data.data.items.length === 0 ? (
            <p className="text-sm text-muted-foreground">Подтверждённых чеков пока нет.</p>
          ) : (
            <div className="space-y-3">
              {result.data.data.items.map((receipt) => (
                <article key={receipt.id} className="rounded-lg border p-4">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <h4 className="font-semibold">{receipt.seller}</h4>
                      <p className="text-sm text-muted-foreground">
                        {receipt.issuedDate
                          ? `${receipt.issuedDate} · время неизвестно`
                          : receipt.issuedAt
                            ? formatDateTime(receipt.issuedAt)
                            : 'Дата неизвестна'}
                      </p>
                    </div>
                    <p className="font-semibold">
                      {formatMoney(receipt.totalMinor, receipt.currency)}
                    </p>
                  </div>
                  <div className="mt-3 space-y-1 border-t pt-3 text-sm">
                    {receipt.items.map((item) => (
                      <p key={item.id} className="flex justify-between gap-3">
                        <span className="text-muted-foreground">
                          {item.position}. {item.name} · {item.quantityMilli / 1000}
                        </span>
                        <span>{formatMoney(item.lineTotalMinor, receipt.currency)}</span>
                      </p>
                    ))}
                  </div>
                </article>
              ))}
            </div>
          )}
        </ResultShell>
      )
    case 'get_period_since_income':
      return (
        <ResultShell title="Период после последнего дохода" result={result}>
          <div className="rounded-lg border bg-muted/30 p-4">
            <p className="text-sm text-muted-foreground">Точка отсчёта</p>
            <p className="font-semibold">
              {formatDateTime(result.data.data.anchor.occurredAt)} ·{' '}
              {result.data.data.anchor.categoryName ?? 'Без категории'}
            </p>
            <div className="mt-2 text-sm text-muted-foreground">
              {result.data.data.anchor.postings.map((posting) => (
                <p key={`${posting.accountId}-${posting.currency}`}>
                  {posting.accountName}: {formatMoney(posting.amountMinor, posting.currency)}
                </p>
              ))}
            </div>
          </div>
          <PeriodCurrencies items={result.data.data.summary.currencies} />
        </ResultShell>
      )
    default:
      return <MalformedResult />
  }
}

function ResultShell({
  title,
  result,
  children,
}: {
  title: string
  result: AssistantCompleted
  children: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2 text-success">
          <CheckCircle2 className="size-5" aria-hidden="true" />
          <CardTitle>{title}</CardTitle>
        </div>
        <CardDescription>Данные рассчитаны детерминированными модулями Grace.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {children}
        <Confidence value={result.confidence} route={result.route} />
      </CardContent>
    </Card>
  )
}

function MalformedResult() {
  return (
    <Notice
      title="Не удалось обработать ответ"
      text="Повторите запрос позже. Финансовые данные не изменялись."
    />
  )
}

class ResultBoundary extends Component<
  { result: AssistantResult; children: ReactNode },
  { failed: boolean }
> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidUpdate(previous: Readonly<{ result: AssistantResult; children: ReactNode }>) {
    if (previous.result !== this.props.result && this.state.failed) {
      this.setState({ failed: false })
    }
  }

  render() {
    return this.state.failed ? <MalformedResult /> : this.props.children
  }
}

function Notice({ title, text, children }: { title: string; text: string; children?: ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2 text-amber-700">
          <AlertCircle className="size-5" aria-hidden="true" />
          <CardTitle>{title}</CardTitle>
        </div>
        <CardDescription>{text}</CardDescription>
      </CardHeader>
      {children && <CardContent>{children}</CardContent>}
    </Card>
  )
}

function AssistantResultView({ result }: { result: AssistantResult }) {
  switch (result.status) {
    case 'completed':
      return <CompletedResult result={result} />
    case 'clarify': {
      const required = result.requiredArguments ?? []
      return (
        <Notice
          title="Нужно уточнение"
          text={clarificationText[result.clarificationCode] ?? 'Уточните запрос и повторите.'}
        >
          {required.length > 0 && (
            <p className="text-sm">
              Нужно указать: {required.map((argument) => argumentLabels[argument]).join(', ')}.
            </p>
          )}
          <Confidence value={result.confidence} route={result.route} />
        </Notice>
      )
    }
    case 'unsupported':
      return (
        <Notice
          title="Запрос пока не поддерживается"
          text="Этот ассистент только читает поддерживаемые финансовые данные и ничего не изменяет."
        >
          <Confidence value={result.confidence} route={result.route} />
        </Notice>
      )
    case 'unavailable':
      return result.errorCode === 'dependency_unavailable' ? (
        <Notice
          title="Финансовые данные временно недоступны"
          text="Grace не смог получить данные. Повторите запрос позже."
        />
      ) : (
        <Notice
          title="Ассистент недоступен"
          text="Он выключен или временно не отвечает. Остальные функции Grace продолжают работать."
        />
      )
    default:
      return <MalformedResult />
  }
}

function errorMessage(error: unknown) {
  if (error instanceof AssistantRequestError) {
    if (error.kind === 'validation') return 'Проверьте текст вопроса и повторите.'
    if (error.kind === 'malformed') return 'Не удалось обработать ответ. Повторите запрос позже.'
  }
  return 'Не удалось связаться с ассистентом. Проверьте соединение и повторите.'
}

export function AssistantPage({
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
    <AppLayout>
      <section className="space-y-6">
        <PageHeader
          title="Ассистент"
          description="Спросите о своих деньгах. Ответы опираются на проверяемые расчёты Grace."
        />
        <AIAnalysis />
        <ReceiptPhoto />
        <AIGuide />
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
                  aria-invalid={Boolean(
                    error && (byteCount === 0 || byteCount > MAX_UTTERANCE_BYTES),
                  )}
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
      </section>
    </AppLayout>
  )
}
