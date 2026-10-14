import { Link } from 'react-router-dom'
import { ArrowRight, CircleAlert } from 'lucide-react'
import type { Account, CashPosition, FinancialEvent } from '@/shared/api/types'
import {
  dateText,
  dueSoonText,
  eventDateText,
  accountLabel,
} from '@/shared/lib/finance-display'
import { Button } from '@/shared/ui/button'
import { AccountIcon } from '@/shared/ui/account-icon'
import { Money } from '@/shared/ui/money'
import { StatusBadge } from '@/shared/ui/status'
import { LoadingSkeleton } from '@/shared/ui/skeleton'

export function Overview({
  cash,
  pending,
  failed,
  events,
  accounts,
  retry,
}: {
  cash?: CashPosition
  pending: boolean
  failed: boolean
  events: FinancialEvent[]
  accounts: Account[]
  retry: () => void
}) {
  const payments =
    (failed ? undefined : cash)?.currencies.flatMap((c) =>
      c.scheduledPayments.map((p) => ({ ...p, currency: c.currency })),
    ) ?? []
  const late = payments.filter((p) => p.timing === 'overdue')
  const upcoming = payments
    .filter((p) => p.timing !== 'overdue')
    .sort((a, b) => a.dueOn.localeCompare(b.dueOn))
    .slice(0, 3)
  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1.65fr)_minmax(260px,1fr)]">
      <section
        className="min-w-0 rounded-2xl bg-secondary p-6 lg:col-start-1 lg:row-start-1"
        aria-labelledby="balance-title"
      >
        <h2 id="balance-title">После известных платежей</h2>
        {failed ? (
          <>
            <p className="mt-4 text-muted-foreground">Расчёт недоступен</p>
            <div className="grace-actions">
              <Button variant="outline" onClick={retry}>
                Повторить расчёт
              </Button>
            </div>
          </>
        ) : pending ? (
          <LoadingSkeleton label="Рассчитываем доступные деньги…" />
        ) : cash?.currencies.length ? (
          cash.currencies.map((c) => (
            <div key={c.currency}>
              <p className="mt-3">
                <Money valueMinor={c.availableAfterObligationsMinor} currency={c.currency} size="display" />
              </p>
              <p className="mt-2 text-sm text-muted-foreground">
                Расчёт по {dateText(cash.through)} включительно
              </p>
              <dl className="grace-fields mt-5 text-sm">
                <div>
                  <dt>Деньги на дебетовых счетах и наличные</dt>
                  <dd><Money valueMinor={c.liquidBalanceMinor} currency={c.currency} size="sm" /></dd>
                </div>
                <div>
                  <dt>Известные платежи до этой даты</dt>
                  <dd><Money valueMinor={c.obligationsThroughMinor} currency={c.currency} size="sm" /></dd>
                </div>
              </dl>
            </div>
          ))
        ) : (
          <p className="mt-4 text-muted-foreground">
            Нет ликвидных счетов и платежей до этой даты.
          </p>
        )}
        <p className="mt-4 text-sm text-muted-foreground">
          Будущие доходы и незапланированные расходы не учтены.
        </p>
      </section>
      {late.length > 0 && (
        <section className="min-w-0 rounded-2xl border border-border bg-card p-6 lg:col-start-2 lg:row-start-1">
          <h2 className="flex items-center gap-2">
            <CircleAlert
              aria-hidden="true"
              className="size-5 text-destructive"
            />
            Требует внимания
          </h2>
          {late.slice(0, 3).map((p) => (
            <div key={p.scheduledPaymentId} className="mt-4">
              <p className="font-medium">{p.obligationName}</p>
              <p className="mt-2 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                <StatusBadge tone="danger">Просрочено</StatusBadge>
                {dateText(p.dueOn)}
              </p>
              <p className="mt-3">
                <Money valueMinor={p.remainingMinor} currency={p.currency} size="lg" />
              </p>
            </div>
          ))}
          <div className="grace-actions">
            <Button variant="outline" asChild>
              <Link to="/payments">
                Посмотреть платежи
                <ArrowRight className="size-4" />
              </Link>
            </Button>
          </div>
        </section>
      )}
      <section className={`min-w-0 rounded-2xl border border-border bg-card p-6 lg:col-start-2 ${late.length ? 'lg:row-start-2' : 'lg:row-start-1'}`}>
        <h2>Ближайшие платежи</h2>
        {failed ? (
          <p className="mt-4 text-muted-foreground">
            График обзора недоступен. Полный список обязательств можно открыть
            отдельно.
          </p>
        ) : upcoming.length ? (
          upcoming.map((p) => (
            <Link
              key={p.scheduledPaymentId}
              to="/payments"
              className="grace-row mt-2"
            >
              <span className="min-w-0 flex-1">
                <strong className="block break-words">
                  {p.obligationName}
                </strong>
                <small className="mt-1 flex flex-wrap items-center gap-2 text-muted-foreground">
                  {p.timing === 'due_soon' && <StatusBadge tone="warning">{dueSoonText(p.daysUntilDue)}</StatusBadge>}
                  {dateText(p.dueOn)}
                </small>
              </span>
              <Money valueMinor={p.remainingMinor} currency={p.currency} className="max-w-[50%] text-right" />
            </Link>
          ))
        ) : (
          <p className="mt-4 text-sm text-muted-foreground">
            {pending ? 'Загружаем…' : 'До выбранной даты платежей нет.'}
          </p>
        )}
        <div className="grace-actions">
          <Button variant="ghost" asChild>
            <Link to="/payments">
              Все платежи
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>
      </section>
      <section className="min-w-0 rounded-2xl border border-border bg-card p-6 lg:col-start-1 lg:row-start-2">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2>Последние операции</h2>
          <Button variant="ghost" asChild>
            <Link to="/operations">
              Все операции
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>
        {events.slice(0, 3).map((e) => {
          const p = e.postings[0],
            a = accounts.find((a) => a.id === p?.accountId)
          return (
            <Link to="/operations" key={e.id} className="grace-row mt-1">
              <span className="min-w-0 flex-1">
                <strong className="block break-words">
                  {e.merchant || e.description || 'Операция'}
                </strong>
                <small className="mt-1 block text-muted-foreground">
                  {eventDateText(e)} ·{' '}
                  {a ? accountLabel(a) : (p?.accountName ?? 'Не указан счёт')}
                </small>
              </span>
              {p && (
                <span className="max-w-[50%] text-right">
                  <Money
                    valueMinor={p.amountMinor}
                    currency={p.currency}
                    direction={e.kind === 'internal_transfer' ? undefined : p.direction === 'credit' ? 'credit' : 'debit'}
                  />
                  {e.kind === 'internal_transfer' && (
                    <small className="block text-muted-foreground">
                      Перевод
                    </small>
                  )}
                </span>
              )}
            </Link>
          )
        })}
        {!events.length && (
          <p className="mt-4 text-muted-foreground">Операций пока нет.</p>
        )}
      </section>
      <section className={`min-w-0 rounded-2xl border border-border bg-card p-6 lg:col-start-2 ${late.length ? 'lg:row-start-3' : 'lg:row-start-2'}`}>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2>Счета и карты</h2>
          <Button variant="ghost" asChild>
            <Link to="/accounts">
              Подробнее
              <ArrowRight className="size-4" />
            </Link>
          </Button>
        </div>
        {accounts.map((a) => (
          <Link key={a.id} to="/accounts" className="grace-row mt-1">
            <AccountIcon type={a.type} />
            <span className="min-w-0 flex-1">
              <strong className="block break-words">{accountLabel(a)}</strong>
              <small className="mt-1 block text-muted-foreground">
                {a.type === 'cash'
                  ? 'Наличные'
                  : a.type === 'credit'
                    ? 'Баланс кредитного счёта'
                    : 'Баланс счёта'}
              </small>
            </span>
            <Money valueMinor={a.balanceMinor} currency={a.currency} className="max-w-[48%] text-right" />
          </Link>
        ))}
      </section>
    </div>
  )
}
