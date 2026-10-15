import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CreateAccountForm } from './CreateAccountForm'

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
  it('preserves leading zeros when creating the first card', async () => {
    const user = userEvent.setup()
    post.mockResolvedValue({ data: account })
    wrap(<CreateAccountForm onSaved={() => {}} />)
    await user.type(screen.getByLabelText('Банк'), 'Банк')
    await user.type(screen.getByLabelText('Последние четыре цифры карты'), '0012')
    await user.type(screen.getByLabelText('Текущий остаток'), '100,01')
    await user.click(screen.getByRole('button', { name: 'Проверить данные' }))
    expect(post).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Создать счёт' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body.card.last4).toBe('0012')
    expect(post.mock.calls[0][1].body.openingBalanceMinor).toBe(10001)
  })
})

describe('SPEC-0028 unreconciled accounts', () => {
  it('saves unknown opening balance explicitly and requires confirmation', async () => {
    const user = userEvent.setup()
    post.mockResolvedValue({ data: { ...account, balanceMinor: null, balanceReconciled: false } })
    wrap(<CreateAccountForm onSaved={() => {}} />)
    await user.click(screen.getByLabelText('Есть карта'))
    await user.type(screen.getByLabelText('Название'), 'История')
    await user.click(screen.getByLabelText('Остаток пока не сверён'))
    await user.click(screen.getByRole('button', { name: 'Проверить данные' }))
    expect(screen.getByText('Не сверён')).toBeVisible()
    expect(post).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Создать счёт' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body).toMatchObject({
      openingBalanceMinor: null,
      balanceReconciled: false,
    })
  })
})
