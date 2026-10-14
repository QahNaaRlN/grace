import { useId, useRef, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Link2 } from 'lucide-react'
import { toast } from 'sonner'

import { apiClient } from '@/shared/api/client'
import type { FinancialEvent, Obligation, ScheduledPayment } from '@/shared/api/types'
import { dueSoonText, formatMoney } from '@/shared/lib/finance-display'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { Money } from '@/shared/ui/money'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'
import { StatusBadge, type Tone } from '@/shared/ui/status'
import { groupSchedule, VISIBLE_ATTENTION } from '../model/group-schedule'

const statusLabels: Record<string, string> = { planned: 'Запланирован', overdue: 'Просрочен', partially_paid: 'Частично оплачен', paid: 'Оплачен' }
// SPEC-0035: a payment state is a word, an icon and a tone; yellow means «скоро» only (SPEC-0046).
const statusTones: Record<string, Tone> = { planned: 'info', overdue: 'danger', partially_paid: 'info', paid: 'success' }
const day = (iso: string) => new Date(`${iso}T00:00:00Z`).toLocaleDateString('ru-RU', { timeZone: 'UTC' })
function plural(n: number, one: string, few: string, many: string) {
  const d = n % 10, h = n % 100
  return d === 1 && h !== 11 ? one : d >= 2 && d <= 4 && (h < 12 || h > 14) ? few : many
}
const payments = (n: number) => `${n} ${plural(n, 'платёж', 'платежа', 'платежей')}`

type Linking = { events: FinancialEvent[]; usedEventIDs: Set<string> }

// SPEC-0039: what needs action first; paid history and far payments folded
// behind «Показать все N». Hidden rows are not mounted until opened.
export function ObligationSchedule({ obligation, events, usedEventIDs }: { obligation: Obligation; events: FinancialEvent[]; usedEventIDs: Set<string> }) {
  const groups = groupSchedule(obligation.payments)
  const [linkingId, setLinkingId] = useState('')
  const summary = useRef<HTMLParagraphElement>(null)
  const linking: Linking = { events, usedEventIDs }
  const debtPayments = events.filter(event => event.kind === 'debt_payment' && !usedEventIDs.has(event.id) && event.postings.length === 1 &&
    event.postings[0]?.currency === obligation.currency && event.postings[0]?.direction === 'debit')
  const unpaid = obligation.payments.filter(p => p.remainingMinor > 0)
  const paidCount = obligation.payments.length - unpaid.length
  const overdue = obligation.payments.filter(p => p.isOverdue).length
  const next = (groups.short ?? groups.nearest).find(p => p.remainingMinor > 0 && !p.isOverdue)
  const row = (payment: ScheduledPayment) => (
    <PaymentRow key={payment.id} obligation={obligation} payment={payment} linking={linking}
      open={linkingId === payment.id} onToggle={() => setLinkingId(linkingId === payment.id ? '' : payment.id)}
      onLinked={() => { setLinkingId(''); summary.current?.focus() }} />
  )

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <p ref={summary} tabIndex={-1} className="text-muted-foreground outline-none">
          Оплачено {paidCount} из {payments(obligation.payments.length)}{next ? ` · следующий ${day(next.dueOn)}` : ''}
        </p>
        {overdue > 0 && <StatusBadge tone="danger">Просрочено {overdue}</StatusBadge>}
      </div>
      {unpaid.length > 0 && debtPayments.length === 0 && (
        <p className="text-sm text-muted-foreground">Сначала создайте подходящий платёж по обязательству.</p>
      )}
      {obligation.status === 'completed' ? (
        <Folded title="Оплачено" label={`Оплачено ${payments(paidCount)}`} items={obligation.payments} render={row} />
      ) : groups.short ? (
        <ul className="divide-y divide-border rounded-md bg-muted/40">{groups.short.map(row)}</ul>
      ) : <>
        {groups.attention.length > 0 && (
          <Section title="Требуют внимания">
            <ul className="divide-y divide-border rounded-md bg-muted/40">{groups.attention.slice(0, VISIBLE_ATTENTION).map(row)}</ul>
            {groups.attention.length > VISIBLE_ATTENTION && (
              <Folded label={`Ещё ${groups.attention.length - VISIBLE_ATTENTION} требуют внимания`} items={groups.attention.slice(VISIBLE_ATTENTION)} render={row} />
            )}
          </Section>
        )}
        {groups.nearest.length > 0 && (
          <Section title="Ближайшие"><ul className="divide-y divide-border rounded-md bg-muted/40">{groups.nearest.map(row)}</ul></Section>
        )}
        {groups.later.length > 0 && (
          <Folded title="Дальше" label={`Ещё ${payments(groups.later.length)} до ${day(groups.later.at(-1)!.dueOn)}`} items={groups.later} render={row} />
        )}
        {groups.paid.length > 0 && <Folded title="Оплачено" label={`Оплачено ${payments(groups.paid.length)}`} items={groups.paid} render={row} />}
      </>}
    </div>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  const id = useId()
  return <section aria-labelledby={id} className="space-y-2"><h4 id={id} className="text-sm font-semibold">{title}</h4>{children}</section>
}

function Folded({ title, label, items, render }: { title?: string; label: string; items: ScheduledPayment[]; render: (p: ScheduledPayment) => ReactNode }) {
  const [open, setOpen] = useState(false)
  const id = useId()
  const toggle = (
    <Button variant="link" aria-expanded={open} aria-controls={id} onClick={() => setOpen(!open)}>
      {open ? 'Свернуть' : `Показать все ${items.length}`}
    </Button>
  )
  const body = (
    <>
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-md bg-muted/60 px-3 text-sm">
        <span>{label}</span>{toggle}
      </div>
      {open && <ul id={id} className="divide-y divide-border rounded-md bg-muted/40">{items.map(render)}</ul>}
      {open && items.length >= 50 && <div className="flex justify-end"><Button variant="link" aria-controls={id} onClick={() => setOpen(false)}>Свернуть</Button></div>}
    </>
  )
  return title ? <Section title={title}>{body}</Section> : <div className="space-y-2">{body}</div>
}

function PaymentRow({ obligation, payment, linking, open, onToggle, onLinked }: {
  obligation: Obligation; payment: ScheduledPayment; linking: Linking; open: boolean; onToggle: () => void; onLinked: () => void
}) {
  const formId = useId()
  const candidates = linking.events.filter(event => {
    const posting = event.postings[0]
    return event.kind === 'debt_payment' && event.postings.length === 1 && !linking.usedEventIDs.has(event.id) &&
      posting?.currency === obligation.currency && posting.direction === 'debit' && posting.amountMinor <= payment.remainingMinor
  })
  return (
    <li className="space-y-2 px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
        <span className="font-medium">{day(payment.dueOn)}</span>
        <span className="text-muted-foreground">осталось <Money valueMinor={payment.remainingMinor} currency={obligation.currency} size="sm" /></span>
        <span className="ml-auto flex min-w-0 max-w-full flex-wrap items-center gap-2">
          {payment.dueSoon && <StatusBadge tone="warning">{dueSoonText(payment.daysUntilDue)}</StatusBadge>}
          <StatusBadge className="h-auto min-h-6 shrink whitespace-normal py-0.5" tone={payment.isOverdue ? 'danger' : statusTones[payment.status] ?? 'unknown'}>
            {statusLabels[payment.status] ?? 'Статус неизвестен'}{payment.isOverdue && payment.status === 'partially_paid' ? ' · просрочен' : ''}
          </StatusBadge>
        </span>
        {payment.remainingMinor > 0 && candidates.length > 0 && (
          <Button variant="outline" className="w-full sm:w-auto" aria-expanded={open} aria-controls={formId} onClick={onToggle}>
            <Link2 aria-hidden="true" />Связать платёж
          </Button>
        )}
      </div>
      {open && <div id={formId}><SettlementForm obligation={obligation} payment={payment} candidates={candidates} onLinked={onLinked} /></div>}
    </li>
  )
}

function SettlementForm({ obligation, payment, candidates, onLinked }: { obligation: Obligation; payment: ScheduledPayment; candidates: FinancialEvent[]; onLinked: () => void }) {
  const queryClient = useQueryClient()
  const [eventID, setEventID] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  async function settle() {
    if (!eventID || isSubmitting) return
    setIsSubmitting(true)
    const result = await apiClient.POST('/api/v1/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements', {
      params: { path: { obligationId: obligation.id, paymentId: payment.id }, header: { 'Idempotency-Key': createIdempotencyKey(), 'X-CSRF-Token': csrfToken } },
      body: { financialEventId: eventID },
    })
    setIsSubmitting(false)
    if (result.error) {
      toast.error(result.error.message)
      return
    }
    await Promise.all([queryClient.invalidateQueries({ queryKey: ['obligations'] }), queryClient.invalidateQueries({ queryKey: ['cash-position'] })])
    toast.success('Фактический платёж связан с графиком')
    onLinked()
  }
  return (
    <div className="flex flex-wrap gap-2">
      <Select value={eventID} onValueChange={setEventID}>
        <SelectTrigger className="min-w-0 flex-1" aria-label="Фактический платёж"><SelectValue placeholder="Выберите фактический платёж" /></SelectTrigger>
        <SelectContent>
          {candidates.map(event => (
            <SelectItem key={event.id} value={event.id}>
              {event.merchant || event.description || 'Платёж'} · {formatMoney(event.postings[0].amountMinor, event.postings[0].currency)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button className="w-full sm:w-auto" type="button" variant="outline" disabled={!eventID || isSubmitting} onClick={() => void settle()}>
        {isSubmitting ? 'Связываю…' : 'Связать'}
      </Button>
    </div>
  )
}
