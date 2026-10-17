import { useId, useState, type ReactNode } from 'react'
import { Link2 } from 'lucide-react'
import { toast } from 'sonner'
import type { FinancialEvent, Obligation, ScheduledPayment } from '@/shared/api'
import { dueSoonText } from '@/shared/lib/finance-display'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Money } from '@/shared/ui/money'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'
import { StatusBadge } from '@/shared/ui/status'
import { useSettlePayment } from '../api/use-settle-payment'
import { statusLabels, statusTones, day, type Linking } from './schedule-display'

export function Section({ title, children }: { title: string; children: ReactNode }) {
  const id = useId()
  return (
    <section aria-labelledby={id} className="space-y-2">
      <h4 id={id} className="text-sm font-semibold">
        {title}
      </h4>
      {children}
    </section>
  )
}

export function Folded({
  title,
  label,
  items,
  render,
}: {
  title?: string
  label: string
  items: ScheduledPayment[]
  render: (p: ScheduledPayment) => ReactNode
}) {
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
        <span>{label}</span>
        {toggle}
      </div>
      {open && (
        <ul id={id} className="divide-y divide-border rounded-md bg-muted/40">
          {items.map(render)}
        </ul>
      )}
      {open && items.length >= 50 && (
        <div className="flex justify-end">
          <Button variant="link" aria-controls={id} onClick={() => setOpen(false)}>
            Свернуть
          </Button>
        </div>
      )}
    </>
  )
  return title ? <Section title={title}>{body}</Section> : <div className="space-y-2">{body}</div>
}

export function PaymentRow({
  obligation,
  payment,
  linking,
  open,
  onToggle,
  onLinked,
}: {
  obligation: Obligation
  payment: ScheduledPayment
  linking: Linking
  open: boolean
  onToggle: () => void
  onLinked: () => void
}) {
  const formId = useId()
  const candidates = linking.events.filter((event) => {
    const posting = event.postings[0]
    return (
      event.kind === 'debt_payment' &&
      event.postings.length === 1 &&
      !linking.usedEventIDs.has(event.id) &&
      posting?.currency === obligation.currency &&
      posting.direction === 'debit' &&
      (obligation.kind === 'loan' || posting.amountMinor <= payment.remainingMinor)
    )
  })
  return (
    <li className="space-y-2 px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
        <span className="font-medium">{day(payment.dueOn)}</span>
        <span className="text-muted-foreground">
          осталось{' '}
          <Money valueMinor={payment.remainingMinor} currency={obligation.currency} size="sm" />
        </span>
        {payment.principalMinor !== null && payment.interestMinor !== null && (
          <span className="text-xs text-muted-foreground">
            тело {formatMoney(payment.principalMinor, obligation.currency)} · проценты{' '}
            {formatMoney(payment.interestMinor, obligation.currency)}
            {payment.otherMinor
              ? ` · прочее ${formatMoney(payment.otherMinor, obligation.currency)}`
              : ''}
            {payment.partsSource === 'calculated' ? ' · рассчитано Grace' : ''}
          </span>
        )}
        <span className="ml-auto flex min-w-0 max-w-full flex-wrap items-center gap-2">
          {payment.dueSoon && (
            <StatusBadge tone="warning">{dueSoonText(payment.daysUntilDue)}</StatusBadge>
          )}
          <StatusBadge
            className="h-auto min-h-6 shrink whitespace-normal py-0.5"
            tone={payment.isOverdue ? 'danger' : (statusTones[payment.status] ?? 'unknown')}
          >
            {statusLabels[payment.status] ?? 'Статус неизвестен'}
            {payment.isOverdue && payment.status === 'partially_paid' ? ' · просрочен' : ''}
          </StatusBadge>
        </span>
        {payment.remainingMinor > 0 && candidates.length > 0 && (
          <Button
            variant="outline"
            className="w-full sm:w-auto"
            aria-expanded={open}
            aria-controls={formId}
            onClick={onToggle}
          >
            <Link2 aria-hidden="true" />
            Связать платёж
          </Button>
        )}
      </div>
      {open && (
        <div id={formId}>
          <SettlementForm
            obligation={obligation}
            payment={payment}
            candidates={candidates}
            onLinked={onLinked}
          />
        </div>
      )}
    </li>
  )
}

export function SettlementForm({
  obligation,
  payment,
  candidates,
  onLinked,
}: {
  obligation: Obligation
  payment: ScheduledPayment
  candidates: FinancialEvent[]
  onLinked: () => void
}) {
  const settlement = useSettlePayment()
  const [eventID, setEventID] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const chosen = candidates.find((event) => event.id === eventID)
  const above = chosen ? chosen.postings[0].amountMinor > payment.remainingMinor : false
  async function settle() {
    if (!eventID || isSubmitting) return
    setIsSubmitting(true)
    const result = await settlement.mutateAsync({
      obligationId: obligation.id,
      paymentId: payment.id,
      financialEventId: eventID,
      prepayRemainder: above,
    })
    setIsSubmitting(false)
    if (result.error) {
      toast.error(result.error.message)
      return
    }
    toast.success(
      above
        ? 'Платёж закрыл строку графика, остальное — досрочное погашение'
        : 'Фактический платёж связан с графиком',
    )
    onLinked()
  }
  return (
    <div className="flex flex-wrap gap-2">
      <Select value={eventID} onValueChange={setEventID}>
        <SelectTrigger className="min-w-0 flex-1" aria-label="Фактический платёж">
          <SelectValue placeholder="Выберите фактический платёж" />
        </SelectTrigger>
        <SelectContent>
          {candidates.map((event) => (
            <SelectItem key={event.id} value={event.id}>
              {event.merchant || event.description || 'Платёж'} ·{' '}
              {formatMoney(event.postings[0].amountMinor, event.postings[0].currency)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button
        className="w-full sm:w-auto"
        type="button"
        variant="outline"
        disabled={!eventID || isSubmitting}
        onClick={() => void settle()}
      >
        {isSubmitting ? 'Связываю…' : above ? 'Связать и погасить досрочно' : 'Связать'}
      </Button>
      {above && (
        <p className="w-full text-sm text-muted-foreground">
          Платёж больше остатка строки: остаток закроет строку, остальное станет досрочным
          погашением кредита. Новый график после погашения скопируйте из банка.
        </p>
      )}
    </div>
  )
}
