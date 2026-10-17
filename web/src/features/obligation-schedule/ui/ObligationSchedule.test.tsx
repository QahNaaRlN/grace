import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import type { FinancialEvent, Obligation, ScheduledPayment } from '@/shared/api'
import { ObligationSchedule } from './ObligationSchedule'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const iso = (i: number) => {
  const d = new Date(Date.UTC(2025, i, 15))
  return d.toISOString().slice(0, 10)
}
// AC1: 240 monthly payments from 15.01.2025, asOf 2026-10-13: 19 paid,
// 15.08.2026 partially paid, 15.09.2026 overdue.
function mortgage(): Obligation {
  const payments: ScheduledPayment[] = Array.from({ length: 240 }, (_, i) => {
    const base = {
      id: `p${i + 1}`,
      position: i + 1,
      dueOn: iso(i),
      amountMinor: 5000000,
      principalMinor: null,
      interestMinor: null,
      otherMinor: null,
      partsSource: null,
      paidMinor: 0,
      remainingMinor: 5000000,
      status: 'planned' as const,
      isOverdue: false,
      dueSoon: false,
      daysUntilDue: 10,
      settlements: [],
    }
    if (i < 19) return { ...base, status: 'paid' as const, paidMinor: 5000000, remainingMinor: 0 }
    if (i === 19)
      return {
        ...base,
        status: 'partially_paid' as const,
        paidMinor: 1000000,
        remainingMinor: 4000000,
        isOverdue: true,
      }
    if (i === 20) return { ...base, status: 'overdue' as const, isOverdue: true }
    return base
  })
  return {
    id: 'm',
    name: 'Ипотека',
    kind: 'loan',
    currency: 'RUB',
    totalMinor: 1_200_000_000,
    paidMinor: 96_000_000,
    remainingMinor: 1_104_000_000,
    status: 'active',
    createdAt: '2025-01-01T00:00:00Z',
    payments,
  }
}
const debt = (id: string, amountMinor: number): FinancialEvent =>
  ({
    id,
    kind: 'debt_payment',
    source: 'manual',
    merchant: 'Банк',
    description: '',
    occurredAt: '2026-10-01T10:00:00Z',
    occurredDate: null,
    datePrecision: 'instant',
    createdAt: '',
    categoryId: null,
    categoryName: null,
    categoryIcon: null,
    originalEventId: null,
    revision: 1,
    editedByUser: false,
    voidedAt: null,
    voidReason: null,
    postings: [
      {
        id: `${id}-p`,
        accountId: 'a',
        accountName: 'Счёт',
        direction: 'debit',
        amountMinor,
        currency: 'RUB',
      },
    ],
  }) as FinancialEvent

function show(obligation: Obligation, events: FinancialEvent[] = []) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ObligationSchedule obligation={obligation} events={events} usedEventIDs={new Set()} />
    </QueryClientProvider>,
  )
}
const rows = () => screen.queryAllByRole('listitem')

beforeEach(() => post.mockReset())

describe('SPEC-0039 compact obligation schedule', () => {
  it('shows what needs action and folds the rest of a 240-payment mortgage', () => {
    show(mortgage())
    expect(screen.getByText('Оплачено 19 из 240 платежей · следующий 15.10.2026')).toBeVisible()
    expect(screen.getByText('Просрочено 2')).toBeVisible()
    const attention = screen.getByRole('region', { name: 'Требуют внимания' })
    expect(within(attention).getAllByRole('listitem')).toHaveLength(2)
    expect(within(attention).getByText('Частично оплачен · просрочен')).toBeVisible()
    const nearest = screen.getByRole('region', { name: 'Ближайшие' })
    expect(
      within(nearest)
        .getAllByRole('listitem')
        .map((li) => li.textContent?.slice(0, 10)),
    ).toEqual(['15.10.2026', '15.11.2026', '15.12.2026'])
    expect(screen.getByText('Ещё 216 платежей до 15.12.2044')).toBeVisible()
    expect(screen.getByText('Оплачено 19 платежей')).toBeVisible()
    expect(rows()).toHaveLength(5)
  })

  it('opens exactly the announced rows and keeps focus on the toggle', async () => {
    show(mortgage())
    const user = userEvent.setup()
    const later = within(screen.getByRole('region', { name: 'Дальше' })).getByRole('button', {
      name: 'Показать все 216',
    })
    await user.click(later)
    expect(later).toHaveAttribute('aria-expanded', 'true')
    expect(later).toHaveTextContent('Свернуть')
    expect(later).toHaveFocus()
    expect(rows()).toHaveLength(5 + 216)
    await user.keyboard('{Enter}')
    expect(rows()).toHaveLength(5)
    expect(later).toHaveFocus()
    const paid = within(screen.getByRole('region', { name: 'Оплачено' })).getByRole('button', {
      name: 'Показать все 19',
    })
    paid.focus()
    await user.keyboard(' ')
    expect(rows()).toHaveLength(5 + 19)
    expect(screen.getAllByText('Оплачен').length).toBe(19)
  })

  it('links a payment from the folded part through the same endpoint, hint never repeated', async () => {
    post.mockResolvedValue({ data: {} })
    const { unmount } = show(mortgage())
    expect(
      screen.getAllByText('Сначала создайте подходящий платёж по обязательству.'),
    ).toHaveLength(1)
    expect(screen.queryByRole('button', { name: 'Связать платёж' })).not.toBeInTheDocument()
    unmount()
    show(mortgage(), [debt('e1', 5000000)])
    expect(
      screen.queryByText('Сначала создайте подходящий платёж по обязательству.'),
    ).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Показать все 216' }))
    const far = screen
      .getAllByRole('listitem')
      .find((li) => li.textContent?.startsWith('15.12.2044'))!
    await user.click(within(far).getByRole('button', { name: 'Связать платёж' }))
    await user.click(within(far).getByRole('combobox', { name: 'Фактический платёж' }))
    await user.click(await screen.findByRole('option', { name: /Банк/ }))
    await user.click(within(far).getByRole('button', { name: 'Связать' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    const [path, init] = post.mock.calls[0]
    expect(path).toBe(
      '/api/v1/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements',
    )
    expect(init.params.path).toEqual({ obligationId: 'm', paymentId: 'p240' })
    expect(init.params.header['Idempotency-Key']).toBeTruthy()
    expect(init.body).toEqual({ financialEventId: 'e1' })
    await waitFor(() => expect(screen.getByText(/Оплачено 19 из 240/)).toHaveFocus())
  })

  it('shows a short schedule whole and folds a completed one', () => {
    const short = { ...mortgage(), payments: mortgage().payments.slice(15, 21) }
    const { unmount } = show(short)
    expect(rows()).toHaveLength(6)
    expect(screen.queryByRole('button', { name: /Показать все/ })).not.toBeInTheDocument()
    unmount()
    const done = {
      ...mortgage(),
      status: 'completed' as const,
      payments: mortgage().payments.slice(0, 19),
    }
    show(done)
    expect(screen.getByText('Оплачено 19 платежей')).toBeVisible()
    expect(rows()).toHaveLength(0)
  })

  it('never hides an unknown status and folds attention beyond five', async () => {
    const many = mortgage()
    many.payments = many.payments.map((p, i) =>
      i >= 19 && i < 27
        ? {
            ...p,
            status: 'overdue' as const,
            isOverdue: true,
            paidMinor: 0,
            remainingMinor: 5000000,
          }
        : p,
    )
    many.payments[30] = { ...many.payments[30], status: 'mystery' as ScheduledPayment['status'] }
    show(many)
    const attention = screen.getByRole('region', { name: 'Требуют внимания' })
    expect(within(attention).getAllByRole('listitem')).toHaveLength(5)
    expect(within(attention).getByText('Ещё 4 требуют внимания')).toBeVisible()
    await userEvent.setup().click(within(attention).getByRole('button', { name: 'Показать все 4' }))
    expect(within(attention).getByText('Статус неизвестен')).toBeVisible()
  })
})

describe('SPEC-0053 loan rows', () => {
  it('shows the parts of a row and books a larger payment as row plus prepayment', async () => {
    const loan = mortgage()
    loan.payments[19] = {
      ...loan.payments[19],
      principalMinor: 3_000_000,
      interestMinor: 2_000_000,
      otherMinor: 0,
      partsSource: 'calculated',
    }
    post.mockResolvedValue({ data: {} })
    show(loan, [debt('e9', 6_000_000)])
    const user = userEvent.setup()
    const attention = screen.getByRole('region', { name: 'Требуют внимания' })
    const row = within(attention)
      .getAllByRole('listitem')
      .find((li) => li.textContent?.includes('рассчитано Grace'))!
    expect(within(row).getByText(/тело 30\s000,00.*проценты 20\s000,00/)).toBeVisible()
    await user.click(within(row).getByRole('button', { name: 'Связать платёж' }))
    await user.click(within(row).getByRole('combobox', { name: 'Фактический платёж' }))
    await user.click(await screen.findByRole('option', { name: /Банк/ }))
    expect(within(row).getByText(/остальное станет досрочным погашением/)).toBeVisible()
    await user.click(within(row).getByRole('button', { name: 'Связать и погасить досрочно' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body).toEqual({ financialEventId: 'e9', prepayRemainder: true })
  })
})
