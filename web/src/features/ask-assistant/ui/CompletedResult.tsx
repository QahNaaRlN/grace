import { CheckCircle2 } from 'lucide-react'
import type { ReactNode } from 'react'

import type { AssistantCompleted, CurrencyPeriodSummary } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { formatMoney } from '@/shared/lib/money'
import { AccountIcon } from '@/shared/ui/account-icon'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { Money } from '@/shared/ui/money'
import { Confidence, MalformedResult } from './result-parts'

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(
    new Date(value),
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

export function CompletedResult({ result }: { result: AssistantCompleted }) {
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
