import { render, screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import type { CreditCard } from '@/entities/credit-card'
import type { Account } from '@/shared/api'
import { queryHarness } from '@/test/query-harness'
import { CreditCardPanel } from './CreditCardPanel'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

const card: Account = {
  id: 'card',
  name: 'Synthetic credit card',
  type: 'credit',
  currency: 'RUB',
  balanceMinor: -1_100_000,
  balanceReconciled: true,
  createdAt: '2026-09-01T00:00:00Z',
  revision: 1,
  hasOperations: true,
  lastReconciliation: null,
  lastImportConfirmedAt: null,
  paymentInstruments: [],
}
const status: CreditCard['status'] = {
  on: '2026-09-28',
  source: 'ledger',
  debtMinor: 1_100_000,
  availableMinor: 8_900_000,
  statementClosedOn: '2026-09-05',
  nextStatementOn: '2026-10-05',
  graceAmountMinor: 900_000,
  graceDueOn: '2026-09-30',
  graceEstimate: false,
  graceLost: false,
  graceAtRisk: true,
  minimumMinor: 140_000,
  minimumDueOn: '2026-09-30',
  minimumIsUpperBound: true,
  overdueMinimumMinor: 0,
  unknown: [],
}
const data: CreditCard = {
  accountId: 'card',
  currency: 'RUB',
  reminders: false,
  activeTermsId: 't',
  terms: [
    {
      id: 't',
      effectiveFrom: '2026-09-01',
      limitMinor: 10_000_000,
      purchaseRateBasisPoints: null,
      cashRateBasisPoints: null,
      graceModel: 'statement',
      statementDay: 5,
      paymentDays: 25,
      graceDays: null,
      minPercentBasisPoints: 1400,
      minAmountMinor: 60_000,
      minRoundingMinor: 10_000,
      minPercentIsCap: true,
      minIncludesInterest: false,
      documentId: null,
      note: '',
      revision: 1,
    },
  ],
  snapshots: [],
  documents: [],
  status,
  bankComparison: null,
}

function show(value: CreditCard) {
  get.mockResolvedValue({ data: value })
  const { wrapper } = queryHarness()
  return render(<CreditCardPanel account={card} accounts={[card]} />, { wrapper })
}

beforeEach(() => get.mockReset())

it('says how much to pay to keep the grace period and warns when the date is near', async () => {
  show(data)
  expect(await screen.findByText(/чтобы не платить/)).toHaveTextContent('9 000')
  expect(screen.getByText('Скоро срок')).toBeInTheDocument()
  expect(screen.getByText(/Минимальный платёж: до/)).toHaveTextContent('1 400')
})

it('shows unknowns instead of zeros when the card has no terms', async () => {
  show({
    ...data,
    activeTermsId: null,
    terms: [],
    status: {
      ...status,
      availableMinor: null,
      graceAmountMinor: null,
      graceDueOn: null,
      minimumMinor: null,
      minimumDueOn: null,
      unknown: ['terms'],
    },
  })
  expect(await screen.findByText('Условия карты не заданы.')).toBeInTheDocument()
  expect(screen.getByText('Неизвестно')).toBeInTheDocument()
  expect(screen.queryByText(/чтобы не платить/)).not.toBeInTheDocument()
  expect(screen.getByRole('checkbox')).toBeDisabled()
})

it('shows both debts when the bank and Grace disagree', async () => {
  show({
    ...data,
    bankComparison: {
      takenOn: '2026-09-25',
      bankDebtMinor: 1_150_000,
      ledgerDebtMinor: 1_100_000,
      differenceMinor: 50_000,
    },
  })
  expect(await screen.findByText('Долг в банке и в Grace расходится')).toBeInTheDocument()
})

it('reports a grace period already lost', async () => {
  show({ ...data, status: { ...status, graceLost: true, graceEstimate: true } })
  expect(await screen.findByText('Льготный период потерян')).toBeInTheDocument()
  expect(screen.getByText(/оценка до выписки/)).toBeInTheDocument()
})
