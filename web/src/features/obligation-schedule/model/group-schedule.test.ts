import { describe, expect, it } from 'vitest'
import type { ScheduledPayment } from '@/shared/api/types'
import { groupSchedule } from './group-schedule'

function payment(position: number, patch: Partial<ScheduledPayment> = {}): ScheduledPayment {
  return { id: `p${position}`, position, dueOn: '2026-01-01', amountMinor: 100, paidMinor: 0, remainingMinor: 100, status: 'planned', isOverdue: false, dueSoon: false, daysUntilDue: 10, settlements: [], ...patch }
}
const paid = (position: number) => payment(position, { status: 'paid', paidMinor: 100, remainingMinor: 0 })
function sizes(payments: ScheduledPayment[]) {
  const g = groupSchedule(payments)
  const all = [...(g.short ?? []), ...g.attention, ...g.nearest, ...g.later, ...g.paid]
  return { g, all }
}

describe('SPEC-0039 groupSchedule', () => {
  it.each([
    ['all paid', Array.from({ length: 12 }, (_, i) => paid(i + 1))],
    ['all future', Array.from({ length: 12 }, (_, i) => payment(i + 1))],
    ['partially paid and overdue', [...Array.from({ length: 4 }, (_, i) => paid(i + 1)), payment(5, { status: 'partially_paid', paidMinor: 40, remainingMinor: 60, isOverdue: true }), payment(6, { status: 'overdue', isOverdue: true }), ...Array.from({ length: 8 }, (_, i) => payment(i + 7))]],
    ['unknown status', [...Array.from({ length: 8 }, (_, i) => payment(i + 1)), payment(9, { status: 'mystery' as ScheduledPayment['status'] })]],
    ['one row', [payment(1)]],
    ['600 rows', Array.from({ length: 600 }, (_, i) => (i < 30 ? paid(i + 1) : payment(i + 1)))],
  ])('puts every payment into exactly one group, in position order: %s', (_name, payments) => {
    const { all } = sizes([...payments].reverse())
    expect(all).toHaveLength(payments.length)
    expect(new Set(all.map(p => p.id)).size).toBe(payments.length)
    const { g } = sizes(payments)
    for (const group of [g.short ?? [], g.attention, g.nearest, g.later, g.paid]) {
      expect(group.map(p => p.position)).toEqual([...group.map(p => p.position)].sort((a, b) => a - b))
    }
  })

  it('shows a short schedule whole', () => {
    const { g } = sizes([paid(1), payment(2, { isOverdue: true, status: 'overdue' }), payment(3)])
    expect(g.short?.map(p => p.id)).toEqual(['p1', 'p2', 'p3'])
  })

  it('keeps overdue, partially paid and unknown visible; three nearest; the rest collapsed', () => {
    const payments = [...Array.from({ length: 19 }, (_, i) => paid(i + 1)),
      payment(20, { status: 'partially_paid', paidMinor: 50, remainingMinor: 50, isOverdue: true }),
      payment(21, { status: 'overdue', isOverdue: true }),
      ...Array.from({ length: 219 }, (_, i) => payment(i + 22))]
    const { g } = sizes(payments)
    expect(g.short).toBeNull()
    expect(g.attention.map(p => p.position)).toEqual([20, 21])
    expect(g.nearest.map(p => p.position)).toEqual([22, 23, 24])
    expect(g.later).toHaveLength(216)
    expect(g.paid).toHaveLength(19)
  })

  it('does not touch money', () => {
    const payments = Array.from({ length: 10 }, (_, i) => payment(i + 1, { amountMinor: 9007199254740991, remainingMinor: 9007199254740991 }))
    const { g } = sizes(payments)
    expect(g.nearest[0]).toBe(payments[0])
  })
})
