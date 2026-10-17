import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'
import { queryHarness } from '@/test/query-harness'
import type { InstallmentAllocation, InstallmentPlan } from '@/entities/installment'
import { InstallmentAllocationSection } from './InstallmentAllocationSection'
const { get, put } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, PUT: put } }))
const plan: InstallmentPlan = {
  id: 'plan',
  obligationId: 'obligation',
  accountId: 'account',
  name: 'Synthetic BNPL',
  mode: 'standalone',
  provider: 'Synthetic',
  contractNumber: '',
  currency: 'RUB',
  purchaseEventId: 'purchase',
  openedOn: '2026-10-01',
  observedOn: '2026-10-01',
  principalMinor: 1200000,
  initialRemainingMinor: 1200000,
  paidMinor: 0,
  remainingMinor: 1200000,
  accountDebtMinor: 1200000,
  differenceMinor: 0,
  historical: false,
  reminders: false,
  revision: 1,
  drafts: [],
  payments: [
    {
      id: 'row',
      position: 1,
      dueOn: '2026-10-01',
      amountMinor: 1200000,
      paidMinor: 0,
      remainingMinor: 1200000,
    },
  ],
}
const allocation: InstallmentAllocation = {
  accountId: 'account',
  currency: 'RUB',
  eventId: 'event',
  revision: 0,
  creditMinor: 300000,
  unallocatedMinor: 300000,
  items: [],
  history: [],
}
beforeEach(() => {
  get.mockReset()
  put.mockReset()
  get.mockImplementation((path: string) =>
    Promise.resolve({
      data: path.includes('installment-allocation')
        ? allocation
        : {
            items: [
              plan,
              { ...plan, id: 'other-plan', accountId: 'other-account', name: 'Other account plan' },
            ],
          },
    }),
  )
})
it('keeps entered allocation on conflict and only offers plans belonging to this transfer account', async () => {
  put.mockResolvedValue({ error: { message: 'Данные изменились' } })
  const user = userEvent.setup()
  const { wrapper } = queryHarness()
  render(<InstallmentAllocationSection eventId="event" />, { wrapper })
  await user.click(await screen.findByText('Распределение по рассрочкам'))
  const amount = screen.getByLabelText(/осталось/)
  await user.type(amount, '3000')
  await user.click(screen.getByRole('button', { name: 'Подтвердить распределение' }))
  expect(await screen.findByText('Данные изменились')).toBeVisible()
  expect(amount).toHaveValue('3 000')
  expect(screen.queryByText('Other account plan')).not.toBeInTheDocument()
  expect(put.mock.calls[0][1].body.items).toEqual([{ paymentId: 'row', amountMinor: 300000 }])
})
