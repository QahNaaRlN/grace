import { useRef, useState } from 'react'
import type { FinancialEvent, Obligation, ScheduledPayment } from '@/shared/api'
import { StatusBadge } from '@/shared/ui/status'
import { groupSchedule, VISIBLE_ATTENTION } from '../model/group-schedule'
import { day, payments, type Linking } from './schedule-display'
import { Section, Folded, PaymentRow } from './ScheduleRows'

// SPEC-0039: what needs action first; paid history and far payments folded
// behind «Показать все N». Hidden rows are not mounted until opened.
export function ObligationSchedule({
  obligation,
  events,
  usedEventIDs,
}: {
  obligation: Obligation
  events: FinancialEvent[]
  usedEventIDs: Set<string>
}) {
  const groups = groupSchedule(obligation.payments)
  const [linkingId, setLinkingId] = useState('')
  const summary = useRef<HTMLParagraphElement>(null)
  const linking: Linking = { events, usedEventIDs }
  const debtPayments = events.filter(
    (event) =>
      event.kind === 'debt_payment' &&
      !usedEventIDs.has(event.id) &&
      event.postings.length === 1 &&
      event.postings[0]?.currency === obligation.currency &&
      event.postings[0]?.direction === 'debit',
  )
  const unpaid = obligation.payments.filter((p) => p.remainingMinor > 0)
  const paidCount = obligation.payments.length - unpaid.length
  const overdue = obligation.payments.filter((p) => p.isOverdue).length
  const next = (groups.short ?? groups.nearest).find((p) => p.remainingMinor > 0 && !p.isOverdue)
  const row = (payment: ScheduledPayment) => (
    <PaymentRow
      key={payment.id}
      obligation={obligation}
      payment={payment}
      linking={linking}
      open={linkingId === payment.id}
      onToggle={() => setLinkingId(linkingId === payment.id ? '' : payment.id)}
      onLinked={() => {
        setLinkingId('')
        summary.current?.focus()
      }}
    />
  )

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <p ref={summary} tabIndex={-1} className="text-muted-foreground outline-none">
          Оплачено {paidCount} из {payments(obligation.payments.length)}
          {next ? ` · следующий ${day(next.dueOn)}` : ''}
        </p>
        {overdue > 0 && <StatusBadge tone="danger">Просрочено {overdue}</StatusBadge>}
      </div>
      {unpaid.length > 0 && debtPayments.length === 0 && (
        <p className="text-sm text-muted-foreground">
          Сначала создайте подходящий платёж по обязательству.
        </p>
      )}
      {obligation.status === 'completed' ? (
        <Folded
          title="Оплачено"
          label={`Оплачено ${payments(paidCount)}`}
          items={obligation.payments}
          render={row}
        />
      ) : groups.short ? (
        <ul className="divide-y divide-border rounded-md bg-muted/40">{groups.short.map(row)}</ul>
      ) : (
        <>
          {groups.attention.length > 0 && (
            <Section title="Требуют внимания">
              <ul className="divide-y divide-border rounded-md bg-muted/40">
                {groups.attention.slice(0, VISIBLE_ATTENTION).map(row)}
              </ul>
              {groups.attention.length > VISIBLE_ATTENTION && (
                <Folded
                  label={`Ещё ${groups.attention.length - VISIBLE_ATTENTION} требуют внимания`}
                  items={groups.attention.slice(VISIBLE_ATTENTION)}
                  render={row}
                />
              )}
            </Section>
          )}
          {groups.nearest.length > 0 && (
            <Section title="Ближайшие">
              <ul className="divide-y divide-border rounded-md bg-muted/40">
                {groups.nearest.map(row)}
              </ul>
            </Section>
          )}
          {groups.later.length > 0 && (
            <Folded
              title="Дальше"
              label={`Ещё ${payments(groups.later.length)} до ${day(groups.later.at(-1)!.dueOn)}`}
              items={groups.later}
              render={row}
            />
          )}
          {groups.paid.length > 0 && (
            <Folded
              title="Оплачено"
              label={`Оплачено ${payments(groups.paid.length)}`}
              items={groups.paid}
              render={row}
            />
          )}
        </>
      )}
    </div>
  )
}
