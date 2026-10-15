import { render, screen, within } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import type { CashPosition } from '@/shared/api'
import { Overview } from './Overview'

const account = {
  id: 'a',
  name: 'Счёт',
  type: 'debit' as const,
  currency: 'RUB',
  balanceReconciled: true,
  revision: 1,
  hasOperations: false,
  lastReconciliation: null,
  lastImportConfirmedAt: null,
  balanceMinor: 100000,
  createdAt: '2026-10-06T00:00:00Z',
  paymentInstruments: [{ id: 'c', name: 'Карта', bank: 'Банк', last4: '0012' }],
}
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

describe('SPEC-0020', () => {
  it('hides stale calculated amounts after analytics fails but keeps accounts visible', () => {
    wrap(
      <Overview
        cash={{
          asOf: '2026-10-06',
          through: '2026-10-31',
          currencies: [
            {
              currency: 'RUB',
              liquidBalanceMinor: 100000,
              overdueObligationsMinor: 0,
              upcomingObligationsMinor: 0,
              obligationsThroughMinor: 0,
              availableAfterObligationsMinor: 99999999,
              scheduledPayments: [],
            },
          ],
        }}
        pending={false}
        failed
        events={[]}
        accounts={[account]}
        retry={() => {}}
      />,
    )
    expect(screen.getByText('Расчёт недоступен')).toBeVisible()
    expect(screen.queryByText(/999\s999/)).not.toBeInTheDocument()
    expect(screen.getByText('Банк *0012')).toBeVisible()
  })
})

describe('SPEC-0028 unreconciled accounts', () => {
  it('displays unknown cash and account balances without converting them to zero', () => {
    wrap(
      <Overview
        cash={{
          asOf: '2026-10-08',
          through: '2026-10-31',
          currencies: [
            {
              currency: 'RUB',
              liquidBalanceMinor: null,
              availableAfterObligationsMinor: null,
              obligationsThroughMinor: 0,
              overdueObligationsMinor: 0,
              upcomingObligationsMinor: 0,
              scheduledPayments: [],
            },
          ],
        }}
        pending={false}
        failed={false}
        events={[]}
        accounts={[{ ...account, balanceMinor: null, balanceReconciled: false }]}
        retry={() => {}}
      />,
    )
    expect(screen.getAllByText('Не сверён').length).toBeGreaterThanOrEqual(2)
  })
})

describe('SPEC-0046 payment due soon', () => {
  it('marks upcoming payments due within 3 days on the overview', () => {
    wrap(
      <Overview
        cash={cash}
        pending={false}
        failed={false}
        events={[]}
        accounts={[]}
        retry={() => {}}
      />,
    )
    const nearest = screen.getByRole('heading', { name: 'Ближайшие платежи' }).parentElement!
    expect(within(nearest).getByRole('link', { name: /Кредит A/ })).toHaveTextContent('Завтра')
    expect(within(nearest).getByRole('link', { name: /Кредит B/ })).toHaveTextContent('Через 3 дня')
    const later = within(nearest).getByRole('link', { name: /Кредит C/ })
    expect(later).not.toHaveTextContent(/Через|Завтра|Сегодня/)
    expect(within(nearest).getByText('Завтра').closest('[data-tone]')).toHaveAttribute(
      'data-tone',
      'warning',
    )
  })
})
