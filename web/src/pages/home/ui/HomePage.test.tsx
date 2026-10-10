import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CreateFinancialEventForm } from '@/features/create-financial-event/ui/CreateFinancialEventForm'
import { CreateAccountForm } from '@/features/create-account/ui/CreateAccountForm'
import { Overview } from './Overview'
import { HomePage } from './HomePage'
const { post, get } = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, GET: get } }))
const account = {
  id: 'a',
  name: 'Счёт',
  type: 'debit' as const,
  currency: 'RUB',
  balanceReconciled: true,
  balanceMinor: 100000,
  createdAt: '2026-10-06T00:00:00Z',
  paymentInstruments: [{ id: 'c', name: 'Карта', bank: 'Банк', last4: '0012' }],
}
function wrap(element: React.ReactNode) {
  return render(
    <MemoryRouter>
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
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
  it('offers the first account without showing invented balances', async () => {
    get.mockImplementation(async (path: string) => ({
      data:
        path === '/api/v1/analytics/cash-position'
          ? { asOf: '2026-10-06', through: '2026-10-31', currencies: [] }
          : { items: [] },
    }))
    wrap(<HomePage />)
    expect(
      await screen.findByRole('heading', { name: 'Начните с первого счёта' }),
    ).toBeVisible()
    expect(
      screen.queryByRole('heading', { name: 'После известных платежей' }),
    ).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(
      screen.getAllByRole('button', { name: 'Добавить первый счёт' })[0],
    )
    expect(screen.getByRole('heading', { name: 'Новый счёт' })).toBeVisible()
    expect(post).not.toHaveBeenCalled()
  })

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
            createdAt: '2026-10-01T00:00:00Z',
          },
        ]}
      />,
    )
    await user.type(screen.getByLabelText('Сумма (RUB)'), '1200,50')
    await user.type(screen.getByLabelText('Описание / комментарий'), 'Пример')
    await user.click(
      screen.getByRole('button', { name: 'Посмотреть подтверждение' }),
    )
    expect(post).not.toHaveBeenCalled()
    expect(
      screen.getByText('Банк *0012', { selector: 'dd' }),
    ).toBeVisible()
    await user.click(
      screen.getByRole('button', { name: 'Подтвердить и сохранить' }),
    )
    await screen.findByText('Повторите')
    await user.click(
      screen.getByRole('button', { name: 'Подтвердить и сохранить' }),
    )
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
    expect(post.mock.calls[0][1].body.amountMinor).toBe(120050)
    expect(post.mock.calls[0][1].params.header['Idempotency-Key']).toBe(
      post.mock.calls[1][1].params.header['Idempotency-Key'],
    )
  })
  it('preserves leading zeros when creating the first card', async () => {
    const user = userEvent.setup()
    post.mockResolvedValue({ data: account })
    wrap(<CreateAccountForm onSaved={() => {}} />)
    await user.type(screen.getByLabelText('Банк'), 'Банк')
    await user.type(
      screen.getByLabelText('Последние четыре цифры карты'),
      '0012',
    )
    await user.type(screen.getByLabelText('Текущий остаток, ₽'), '100,01')
    await user.click(screen.getByRole('button', { name: 'Проверить данные' }))
    expect(post).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Создать счёт' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body.card.last4).toBe('0012')
    expect(post.mock.calls[0][1].body.openingBalanceMinor).toBe(10001)
  })
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
 it('saves unknown opening balance explicitly and requires confirmation', async () => {
  const user=userEvent.setup()
  post.mockResolvedValue({data:{...account,balanceMinor:null,balanceReconciled:false}})
  wrap(<CreateAccountForm onSaved={() => {}} />)
  await user.click(screen.getByLabelText('Есть карта'))
  await user.type(screen.getByLabelText('Название'),'История')
  await user.click(screen.getByLabelText('Остаток пока не сверён'))
  await user.click(screen.getByRole('button',{name:'Проверить данные'}))
  expect(screen.getByText('Не сверён')).toBeVisible()
  expect(post).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button',{name:'Создать счёт'}))
  await waitFor(()=>expect(post).toHaveBeenCalledOnce())
  expect(post.mock.calls[0][1].body).toMatchObject({openingBalanceMinor:null,balanceReconciled:false})
 })
 it('displays unknown cash and account balances without converting them to zero', () => {
  wrap(<Overview cash={{asOf:'2026-10-08',through:'2026-10-31',currencies:[{currency:'RUB',liquidBalanceMinor:null,availableAfterObligationsMinor:null,obligationsThroughMinor:0,overdueObligationsMinor:0,upcomingObligationsMinor:0,scheduledPayments:[]}]}} pending={false} failed={false} events={[]} accounts={[{...account,balanceMinor:null,balanceReconciled:false}]} retry={()=>{}} />)
  expect(screen.getAllByText('Не сверён').length).toBeGreaterThanOrEqual(2)
 })
})
