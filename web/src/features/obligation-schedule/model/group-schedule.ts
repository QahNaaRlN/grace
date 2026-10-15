import type { ScheduledPayment } from '@/shared/api'

// SPEC-0039: presentation grouping of a schedule Go already computed. Only
// status, isOverdue, remainingMinor and position decide the group; React
// never compares dates with «today» and never adds money.

export const NEAREST_COUNT = 3
export const SHORT_SCHEDULE = 6
export const VISIBLE_ATTENTION = 5

const knownStatuses = new Set(['planned', 'overdue', 'partially_paid', 'paid'])

export type ScheduleGroups = {
  /** ≤ SHORT_SCHEDULE payments: shown whole, in position order. */
  short: ScheduledPayment[] | null
  /** Overdue, partially paid or unknown status; never hidden silently. */
  attention: ScheduledPayment[]
  /** The next NEAREST_COUNT unpaid payments. */
  nearest: ScheduledPayment[]
  /** Further unpaid payments, collapsed by default. */
  later: ScheduledPayment[]
  /** Fully paid history, collapsed by default. */
  paid: ScheduledPayment[]
}

export function needsAttention(payment: ScheduledPayment) {
  return (
    payment.isOverdue || payment.status === 'partially_paid' || !knownStatuses.has(payment.status)
  )
}

export function groupSchedule(payments: ScheduledPayment[]): ScheduleGroups {
  const ordered = [...payments].sort((a, b) => a.position - b.position)
  const groups: ScheduleGroups = { short: null, attention: [], nearest: [], later: [], paid: [] }
  if (ordered.length <= SHORT_SCHEDULE) {
    groups.short = ordered
    return groups
  }
  for (const payment of ordered) {
    if (needsAttention(payment)) groups.attention.push(payment)
    else if (payment.remainingMinor > 0 && groups.nearest.length < NEAREST_COUNT)
      groups.nearest.push(payment)
    else if (payment.remainingMinor > 0) groups.later.push(payment)
    else groups.paid.push(payment)
  }
  return groups
}
