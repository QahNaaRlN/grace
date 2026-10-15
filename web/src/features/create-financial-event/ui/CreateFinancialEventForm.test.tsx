import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CreateFinancialEventForm } from './CreateFinancialEventForm'

const { post, get } = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, GET: get } }))
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
beforeEach(() => {
  post.mockReset()
  get.mockReset()
})
describe('SPEC-0020', () => {
  it('does not write before confirmation and reuses the key after a failed save', async () => {
    const user = userEvent.setup()
    post
      .mockResolvedValueOnce({ error: { message: 'Повторите' } })
      .mockResolvedValueOnce({ data: { id: 'event' } })
    wrap(
      <CreateFinancialEventForm
        accounts={[account]}
        categories={[
          {
            id: 'cat',
            name: 'Продукты',
            kind: 'expense',
            icon: null,
            hidden: false,
            operationCount: 0,
            createdAt: '2026-10-01T00:00:00Z',
          },
        ]}
      />,
    )
    await user.type(screen.getByLabelText('Сумма'), '1200,50')
    expect(screen.getByLabelText('Сумма')).toHaveValue('1 200,50')
    await user.type(screen.getByLabelText('Описание / комментарий'), 'Пример')
    await user.click(screen.getByRole('button', { name: 'Посмотреть подтверждение' }))
    expect(post).not.toHaveBeenCalled()
    expect(screen.getByText('Банк *0012', { selector: 'dd' })).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Подтвердить и сохранить' }))
    await screen.findByText('Повторите')
    await user.click(screen.getByRole('button', { name: 'Подтвердить и сохранить' }))
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
    expect(post.mock.calls[0][1].body.amountMinor).toBe(120050)
    expect(post.mock.calls[0][1].params.header['Idempotency-Key']).toBe(
      post.mock.calls[1][1].params.header['Idempotency-Key'],
    )
  })
})
