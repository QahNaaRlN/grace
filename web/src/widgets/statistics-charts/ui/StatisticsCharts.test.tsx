import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { StatisticsCurrency } from '@/shared/api'
import { StatisticsCharts, StatisticsPanel } from './StatisticsCharts'
const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))
const item: StatisticsCurrency = {
  currency: 'RUB',
  summary: {
    currency: 'RUB',
    openingBalanceMinor: null,
    closingBalanceMinor: null,
    incomeMinor: 50000,
    purchasesMinor: 10000,
    refundsMinor: 2000,
    debtPaymentsMinor: 3000,
    debtPrincipalMinor: 0,
    debtInterestMinor: 0,
    debtOtherMinor: 0,
    debtUnallocatedMinor: 3000,
    netExternalCashFlowMinor: 39000,
    internalTransferVolumeMinor: 999000,
    expenseCategories: [],
  },
  categories: [
    {
      categoryId: null,
      categoryName: 'Без категории',
      purchasesMinor: 10000,
      refundsMinor: 2000,
      netExpenseMinor: 8000,
      shareBasisPoints: 10000,
    },
  ],
  series: [
    {
      from: '2026-09-01',
      to: '2026-10-01',
      incomeMinor: 50000,
      purchasesMinor: 10000,
      refundsMinor: 2000,
      debtPaymentsMinor: 3000,
      debtPrincipalMinor: 0,
      debtInterestMinor: 0,
      debtOtherMinor: 0,
      debtUnallocatedMinor: 3000,
      outflowsMinor: 13000,
    },
    {
      from: '2026-10-01',
      to: '2026-10-10',
      incomeMinor: 0,
      purchasesMinor: 0,
      refundsMinor: 0,
      debtPaymentsMinor: 0,
      debtPrincipalMinor: 0,
      debtInterestMinor: 0,
      debtOtherMinor: 0,
      debtUnallocatedMinor: 0,
      outflowsMinor: 0,
    },
  ],
}
function wrap(element: React.ReactNode) {
  return render(
    <MemoryRouter>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        {element}
      </QueryClientProvider>
    </MemoryRouter>,
  )
}
beforeEach(() => get.mockReset())
describe('SPEC-0030 charts', () => {
  it('shows separate gross purchases/refunds and a readable table with keyboard selection', async () => {
    wrap(<StatisticsCharts item={item} />)
    expect(screen.getByText('Без категории')).toBeVisible()
    expect(screen.getByText(/100%/)).toBeVisible()
    const purchase = screen.getAllByRole('button', { name: /1 сент.*Покупки/ })[0]
    purchase.focus()
    await userEvent.keyboard('{Enter}')
    expect(purchase).toHaveAttribute('aria-pressed', 'true')
    const summaries = screen.getAllByText('Таблица значений')
    await userEvent.click(summaries[0])
    expect(screen.getAllByRole('columnheader', { name: 'Возвраты' })[0]).toBeVisible()
    expect(screen.getAllByRole('columnheader', { name: 'Платежи' })[0]).toBeVisible()
    expect(screen.queryByText(/9.990/)).not.toBeInTheDocument()
  })
  it('uses the authoritative API for grouping and currency, without mixing currencies', async () => {
    get.mockResolvedValue({
      data: {
        from: '2026-09-01',
        to: '2026-10-10',
        granularity: 'month',
        coverage: 'recorded_only',
        currencies: [item, { ...item, currency: 'USD' }],
      },
    })
    wrap(<StatisticsPanel />)
    await screen.findByText('Без категории')
    expect(screen.getByText(/Полнота истории не подтверждена/)).toBeVisible()
    await userEvent.click(screen.getByLabelText('Группировка'))
    await userEvent.click(screen.getByRole('option', { name: 'По неделям' }))
    await waitFor(() =>
      expect(get).toHaveBeenLastCalledWith('/api/v1/analytics/statistics', {
        params: { query: expect.objectContaining({ granularity: 'week' }) },
      }),
    )
    await userEvent.click(screen.getByLabelText('Валюта'))
    await userEvent.click(screen.getByRole('option', { name: 'USD' }))
    expect(screen.getByLabelText('Валюта')).toHaveTextContent('USD')
    expect(screen.getAllByRole('group')[0]).toHaveAccessibleName(/USD/)
  })
  it('handles failure and empty data without inventing a chart', async () => {
    get
      .mockResolvedValueOnce({ error: { message: 'Нет соединения' } })
      .mockResolvedValue({ data: { currencies: [] } })
    wrap(<StatisticsPanel />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Нет соединения')
    await userEvent.click(screen.getByRole('button', { name: 'Повторить загрузку' }))
    expect(await screen.findByText('Пока нет финансовых данных для графиков.')).toBeVisible()
    expect(screen.queryByRole('group')).not.toBeInTheDocument()
  })
  it('rejects unsafe amounts instead of rounding financial values', () => {
    wrap(
      <StatisticsCharts
        item={{
          ...item,
          series: [{ ...item.series[0], purchasesMinor: Number.MAX_SAFE_INTEGER + 1 }],
        }}
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('точного отображения')
    expect(screen.queryByRole('group')).not.toBeInTheDocument()
  })
  it('keeps a zero-only recorded period textual and exposes compact statistics navigation', async () => {
    const zero = {
      ...item,
      categories: [],
      summary: { ...item.summary, purchasesMinor: 0 },
      series: [item.series[1]],
    }
    get.mockResolvedValue({ data: { currencies: [zero] } })
    wrap(<StatisticsPanel compact />)
    expect(await screen.findByText('Нет записанных покупок за этот период.')).toBeVisible()
    expect(screen.getByRole('link', { name: 'Открыть статистику' })).toHaveAttribute(
      'href',
      '/analytics',
    )
    expect(screen.queryByRole('group')).not.toBeInTheDocument()
  })
})

describe('SPEC-0053 loan payments', () => {
  it('splits debt payments into principal, cost of credit and unallocated', () => {
    wrap(
      <StatisticsCharts
        item={{
          ...item,
          summary: {
            ...item.summary,
            debtPrincipalMinor: 2000,
            debtInterestMinor: 700,
            debtOtherMinor: 0,
            debtUnallocatedMinor: 300,
          },
        }}
      />,
    )
    const card = screen
      .getByText('Платежи по кредитам')
      .closest('[data-slot="card"]') as HTMLElement
    expect(within(card).getByText('Основной долг').nextSibling).toHaveTextContent(/20,00/)
    expect(within(card).getByText('Проценты').nextSibling).toHaveTextContent(/7,00/)
    expect(within(card).getByText('Не распределено').nextSibling).toHaveTextContent(/3,00/)
  })
  it('hides the card when there were no debt payments', () => {
    wrap(
      <StatisticsCharts item={{ ...item, summary: { ...item.summary, debtPaymentsMinor: 0 } }} />,
    )
    expect(screen.queryByText('Платежи по кредитам')).not.toBeInTheDocument()
  })
})
