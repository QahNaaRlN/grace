import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { LoanAllocationSection } from './LoanAllocationSection'

const { get, put, del } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), del: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, PUT: put, DELETE: del } }))

const allocation = {
  financialEventId: 'e',
  obligationId: 'loan',
  obligationName: 'Кредит наличными',
  currency: 'RUB',
  amountMinor: 2_066_185,
  schedule: {
    settlementId: 's',
    scheduledPaymentId: 'p',
    dueOn: '2026-03-15',
    amountMinor: 1_066_185,
    parts: { principalMinor: 964_408, interestMinor: 101_777, otherMinor: 0 },
    source: 'schedule',
    revision: 0,
  },
  prepayment: {
    id: 'pre',
    financialEventId: 'e',
    paidOn: '2026-03-15',
    amountMinor: 1_000_000,
    principalMinor: 1_000_000,
    interestMinor: 0,
    otherMinor: 0,
    revision: 1,
  },
  unallocatedMinor: 0,
}
function show() {
  return render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <LoanAllocationSection eventId="e" />
    </QueryClientProvider>,
  )
}
beforeEach(() => {
  get.mockReset()
  put.mockReset()
  del.mockReset()
})

describe('SPEC-0053 allocation of a debt payment', () => {
  it('shows nothing for an operation without a loan', async () => {
    get.mockResolvedValue({ response: { status: 404 }, error: { message: 'Не найдено' } })
    const { container } = show()
    await waitFor(() => expect(get).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the split by the schedule and the prepayment, and saves the owner split', async () => {
    get.mockResolvedValue({ response: { status: 200 }, data: allocation })
    put.mockResolvedValue({
      data: { ...allocation, schedule: { ...allocation.schedule, source: 'owner', revision: 1 } },
    })
    show()
    expect(await screen.findByText(/Кредит наличными/)).toBeVisible()
    expect(screen.getByText(/по графику/)).toBeVisible()
    expect(screen.getByText(/Досрочное погашение/)).toBeVisible()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Указать как в выписке' }))
    await user.clear(screen.getByLabelText('Тело'))
    await user.type(screen.getByLabelText('Тело'), '9000')
    await user.clear(screen.getByLabelText('Проценты'))
    await user.type(screen.getByLabelText('Проценты'), '1661,85')
    await user.click(screen.getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(put).toHaveBeenCalled())
    expect(put.mock.calls[0][1].body).toEqual({
      principalMinor: 900_000,
      interestMinor: 166_185,
      otherMinor: 0,
      revision: 0,
    })
    expect(await screen.findByText(/указано вами/)).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Вернуть по графику' }))
    await waitFor(() => expect(del).toHaveBeenCalled())
    expect(del.mock.calls[0][1].params.query).toEqual({ revision: 1 })
  })
})
