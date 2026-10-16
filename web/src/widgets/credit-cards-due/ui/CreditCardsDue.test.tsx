import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, expect, it, vi } from 'vitest'

import type { Account } from '@/shared/api'
import { queryHarness } from '@/test/query-harness'
import { CreditCardsDue } from './CreditCardsDue'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

const account = (id: string, type: Account['type']): Account => ({
  id,
  name: `Synthetic ${id}`,
  type,
  currency: 'RUB',
  balanceMinor: 0,
  balanceReconciled: true,
  createdAt: '2026-09-01T00:00:00Z',
  revision: 1,
  hasOperations: true,
  lastReconciliation: null,
  lastImportConfirmedAt: null,
  paymentInstruments: [],
})
const status = (patch: object) => ({
  data: {
    status: {
      graceAmountMinor: 0,
      graceDueOn: null,
      minimumMinor: 0,
      minimumDueOn: null,
      minimumIsUpperBound: false,
      graceLost: false,
      graceAtRisk: false,
      ...patch,
    },
  },
})

beforeEach(() => get.mockReset())

it('lists cards that need a payment, nearest date first, and skips debit accounts', async () => {
  get.mockImplementation(
    async (_path: string, init: { params: { path: { accountId: string } } }) =>
      init?.params?.path?.accountId === 'later'
        ? status({ graceAmountMinor: 500_000, graceDueOn: '2026-10-30' })
        : status({
            minimumMinor: 60_000,
            minimumDueOn: '2026-10-10',
            minimumIsUpperBound: true,
            graceAtRisk: true,
          }),
  )
  const { wrapper } = queryHarness()
  render(
    <MemoryRouter>
      <CreditCardsDue
        accounts={[
          account('later', 'credit'),
          account('soon', 'credit'),
          account('salary', 'debit'),
        ]}
      />
    </MemoryRouter>,
    { wrapper },
  )
  const items = await screen.findAllByRole('listitem')
  expect(items).toHaveLength(2)
  expect(items[0]).toHaveTextContent(/Synthetic soon.*минимальный платёж не больше 600/)
  expect(items[0]).toHaveTextContent('Скоро срок')
  expect(items[1]).toHaveTextContent(/Synthetic later.*заплатите 5 000.*чтобы не платить проценты/)
  expect(get).toHaveBeenCalledTimes(2)
})

it('renders nothing when no card needs a payment', async () => {
  get.mockResolvedValue(status({}))
  const { wrapper } = queryHarness()
  const { container } = render(
    <MemoryRouter>
      <CreditCardsDue accounts={[account('paid', 'credit')]} />
    </MemoryRouter>,
    { wrapper },
  )
  await vi.waitFor(() => expect(get).toHaveBeenCalled())
  expect(container).toBeEmptyDOMElement()
})
