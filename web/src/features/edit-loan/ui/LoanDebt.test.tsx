import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fakeSchedulePreview } from '@/entities/payment-schedule/index.testing'
import type { components } from '@/shared/api/generated/schema'
import { LoanPanel } from './LoanPanel'

const { get, post, patch } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: post, PATCH: patch } }))

type Debt = components['schemas']['LoanDebtStatus']
const totals = { principalMinor: 0, interestMinor: 0, otherMinor: 0, unknownMinor: 0 }
function debt(patch: Partial<Debt> = {}): Debt {
  return {
    on: '2026-03-31',
    issuedMinor: 12_000_000,
    principalOutstandingMinor: 8_601_700,
    basis: 'snapshot',
    basisOn: '2026-03-20',
    unallocatedPayments: 0,
    latestSnapshot: {
      id: 's',
      takenOn: '2026-03-20',
      principalMinor: 8_601_700,
      accruedInterestMinor: null,
      payoffMinor: 8_650_000,
      note: '',
      revision: 1,
    },
    gracePrincipalOnSnapshotMinor: 8_601_708,
    discrepancyMinor: 8,
    paid: { ...totals, principalMinor: 3_398_292, interestMinor: 234_078 },
    upcoming: { ...totals, principalMinor: 9_091_708, interestMinor: 500_000 },
    scheduleOutdated: true,
    snapshots: [],
    prepayments: [
      {
        id: 'pre',
        financialEventId: 'e-pre',
        paidOn: '2026-03-20',
        amountMinor: 500_000,
        principalMinor: 490_000,
        interestMinor: 10_000,
        otherMinor: 0,
        revision: 1,
      },
    ],
    ...patch,
  }
}
function loan(d: Debt, extra: object = {}) {
  return {
    lender: '',
    agreementNumber: '',
    agreementOn: '2026-01-15',
    principalMinor: 12_000_000,
    annualRateBasisPoints: 1200,
    repaymentType: 'annuity',
    reminders: false,
    revision: 1,
    documents: [],
    drafts: [],
    debt: d,
    ...extra,
  }
}
function renderPanel() {
  return render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <LoanPanel obligationId="loan" currency="RUB" />
    </QueryClientProvider>,
  )
}
const draft = {
  id: 'copy',
  revision: 1,
  appliedAt: null,
  payments: [
    { sourceId: 'paid', dueOn: '2026-02-15', amountMinor: 1_066_185, paidMinor: 1_066_185 },
    { sourceId: 'future', dueOn: '2026-03-15', amountMinor: 1_066_185, paidMinor: 0 },
  ],
}
beforeEach(() => {
  get.mockReset()
  post.mockReset()
  patch.mockReset()
})

describe('SPEC-0053 loan debt', () => {
  it('shows Grace and bank principal, the discrepancy, outdated schedule and prepayments', async () => {
    get.mockResolvedValue({ data: loan(debt()) })
    renderPanel()
    const section = await screen.findByRole('region', { name: 'Долг по кредиту' })
    expect(
      within(section).getByText(/Данные банка на .* минус тело платежей после этой даты/),
    ).toBeVisible()
    expect(within(section).getByText(/Банк на .*полное погашение/)).toBeVisible()
    expect(within(section).getByText(/расхождение 0,08/)).toBeVisible()
    expect(within(section).getByText('График устарел')).toBeVisible()
    expect(within(section).getByText('Досрочные погашения')).toBeVisible()
    expect(within(section).getByRole('button', { name: 'Досрочное погашение' })).toBeDisabled()
  })

  it('says the debt is unknown instead of showing zero', async () => {
    get.mockResolvedValue({
      data: loan(
        debt({
          principalOutstandingMinor: null,
          basis: 'unknown',
          basisOn: null,
          latestSnapshot: null,
          discrepancyMinor: null,
          gracePrincipalOnSnapshotMinor: null,
          scheduleOutdated: false,
          prepayments: [],
        }),
      ),
    })
    renderPanel()
    const section = await screen.findByRole('region', { name: 'Долг по кредиту' })
    expect(within(section).getByText('неизвестно')).toBeVisible()
    expect(within(section).getByText(/Укажите сумму выдачи/)).toBeVisible()
    expect(within(section).queryByText('График устарел')).toBeNull()
  })

  it('saves the bank data with an idempotency key', async () => {
    get.mockResolvedValue({ data: loan(debt()) })
    post.mockResolvedValue({ data: debt() })
    renderPanel()
    await userEvent.click(await screen.findByRole('button', { name: 'Ввести данные из банка' }))
    await userEvent.type(screen.getByLabelText('Дата'), '2026-03-20')
    await userEvent.type(screen.getByLabelText('Остаток основного долга'), '86017')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(post).toHaveBeenCalled())
    const [path, init] = post.mock.calls[0]
    expect(path).toBe('/api/v1/loans/{obligationId}/debt-snapshots')
    expect(init.params.header['Idempotency-Key']).toBeTruthy()
    expect(init.body).toEqual({
      takenOn: '2026-03-20',
      principalMinor: 8_601_700,
      accruedInterestMinor: null,
      payoffMinor: null,
      note: '',
    })
  })

  it('calculates principal and interest of a saved copy and shows the warnings', async () => {
    get.mockResolvedValue({ data: loan(debt({ scheduleOutdated: false, prepayments: [] })) })
    post.mockImplementation(
      async (path: string, init: { body: Parameters<typeof fakeSchedulePreview>[0] }) => {
        if (path === '/api/v1/obligations/schedule-preview') return fakeSchedulePreview(init.body)
        if (path.endsWith('/calculate-parts'))
          return {
            data: {
              draft: {
                ...draft,
                revision: 2,
                payments: [
                  draft.payments[0],
                  {
                    ...draft.payments[1],
                    principalMinor: 964_408,
                    interestMinor: 101_777,
                    otherMinor: 0,
                    partsSource: 'calculated',
                  },
                ],
              },
              rows: [{ warning: 'payment_below_interest' }, { warning: null }],
              principalTotalMinor: 964_408,
              principalDifferenceMinor: -11_035_592,
            },
          }
        return { data: draft }
      },
    )
    renderPanel()
    await userEvent.click(
      screen.getByRole('button', { name: 'Договор, документы и копии графика' }),
    )
    await userEvent.click(await screen.findByRole('button', { name: 'Скопировать текущий график' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Рассчитать тело и проценты' }))
    await waitFor(() =>
      expect(post).toHaveBeenCalledWith(
        '/api/v1/loans/{obligationId}/schedule-drafts/{draftId}/calculate-parts',
        expect.objectContaining({ body: { revision: 1 } }),
      ),
    )
    const principal = await screen.findByLabelText<HTMLInputElement>('Тело, платёж 2')
    expect(principal.value).toMatch(/^9\s644,08$/)
    expect(screen.getByText('Рассчитано Grace — сверьте с графиком банка')).toBeVisible()
    expect(screen.getByText(/Платёж 1: платёж меньше процентов/)).toBeVisible()
  })
})
