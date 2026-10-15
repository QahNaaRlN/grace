import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import type { CashPosition } from '@/shared/api'
import { PaymentsPage } from './PaymentsPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

const payment = (
  id: string,
  dueOn: string,
  timing: 'overdue' | 'due_soon' | 'upcoming',
  daysUntilDue: number,
) => ({
  scheduledPaymentId: id,
  obligationId: 'o',
  obligationName: `Кредит ${id}`,
  dueOn,
  amountMinor: 1000,
  paidMinor: 0,
  remainingMinor: 1000,
  timing,
  daysUntilDue,
})
const cash: CashPosition = {
  asOf: '2026-10-14',
  through: '2026-10-31',
  currencies: [
    {
      currency: 'RUB',
      liquidBalanceMinor: 100000,
      availableAfterObligationsMinor: 97000,
      obligationsThroughMinor: 3000,
      overdueObligationsMinor: 0,
      upcomingObligationsMinor: 3000,
      scheduledPayments: [
        payment('A', '2026-10-15', 'due_soon', 1),
        payment('B', '2026-10-17', 'due_soon', 3),
        payment('C', '2026-10-25', 'upcoming', 11),
      ],
    },
  ],
}

function wrap(element: React.ReactNode, url = '/') {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        {element}
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('SPEC-0046 payment due soon', () => {
  it('shows «скоро» on the payments page and keeps yellow only for it', async () => {
    const scheduled = (id: string, dueOn: string, patch: object) => ({
      id,
      position: 1,
      dueOn,
      amountMinor: 1000,
      paidMinor: 0,
      remainingMinor: 1000,
      status: 'planned',
      isOverdue: false,
      dueSoon: false,
      daysUntilDue: 10,
      settlements: [],
      ...patch,
    })
    get.mockImplementation(async (path: string) => ({
      data:
        path === '/api/v1/analytics/cash-position'
          ? cash
          : path === '/api/v1/obligations'
            ? {
                items: [
                  {
                    id: 'o',
                    name: 'Кредит',
                    kind: 'loan',
                    currency: 'RUB',
                    totalMinor: 3000,
                    paidMinor: 500,
                    remainingMinor: 2500,
                    status: 'active',
                    createdAt: '2026-10-01T00:00:00Z',
                    payments: [
                      scheduled('p1', '2026-10-14', { dueSoon: true, daysUntilDue: 0 }),
                      scheduled('p2', '2026-10-24', {
                        status: 'partially_paid',
                        paidMinor: 500,
                        remainingMinor: 500,
                      }),
                      scheduled('p3', '2026-11-14', {}),
                    ],
                  },
                ],
              }
            : path === '/api/v1/accounts'
              ? {
                  items: [
                    {
                      id: 'a',
                      name: 'Счёт',
                      type: 'debit',
                      currency: 'RUB',
                      balanceMinor: 100000,
                      balanceReconciled: true,
                      createdAt: '2026-10-01T00:00:00Z',
                      revision: 1,
                      hasOperations: false,
                      lastReconciliation: null,
                      lastImportConfirmedAt: null,
                    },
                  ],
                }
              : { items: [] },
    }))
    wrap(<PaymentsPage />, '/payments')
    const today = await screen.findByText('Сегодня')
    expect(today.closest('[data-tone]')).toHaveAttribute('data-tone', 'warning')
    expect(screen.getByText('Частично оплачен').closest('[data-tone]')).toHaveAttribute(
      'data-tone',
      'info',
    )
    expect(screen.getAllByText('Запланирован')).toHaveLength(2)
  })
})
