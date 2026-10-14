import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SubscriptionsPage } from './SubscriptionsPage'

const { get, post, patch, toastError } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), toastError: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: post, PATCH: patch } }))
vi.mock('sonner', () => ({ toast: { error: toastError, success: vi.fn() } }))

const categories = [
  { id: 'services', name: 'Сервисы', kind: 'expense', createdAt: '', icon: 'film', hidden: false, operationCount: 0 },
  { id: 'old', name: 'Скрытая', kind: 'expense', createdAt: '', icon: null, hidden: true, operationCount: 0 },
  { id: 'salary', name: 'Зарплата', kind: 'income', createdAt: '', icon: null, hidden: false, operationCount: 0 },
]
const subscription = { id: 's1', name: 'Кинотеатр', amountMinor: 39900, currency: 'RUB', period: 'monthly', startsOn: '2026-10-01', endsOn: null, accountId: null, paymentInstrumentId: null, categoryId: 'services', paused: false, reminders: false, revision: 2, nextDueOn: '2026-11-01', expenseIds: [] }
const legacy = { ...subscription, id: 's2', name: 'Старая подписка', categoryId: null }

function show() {
  render(<MemoryRouter><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><SubscriptionsPage /></QueryClientProvider></MemoryRouter>)
}

beforeEach(() => {
  get.mockReset(); post.mockReset(); patch.mockReset(); toastError.mockReset()
  get.mockImplementation(async (path: string) => ({ data: { items: path === '/api/v1/subscriptions' ? [subscription, legacy] : path === '/api/v1/categories' ? categories : [] } }))
  post.mockResolvedValue({ data: subscription })
  patch.mockResolvedValue({ data: subscription })
})

describe('SPEC-0047 subscription form', () => {
  it('shows the category on the card', async () => {
    show()
    const card = (await screen.findByRole('heading', { name: 'Кинотеатр' })).closest('div')!
    expect(within(card).getByText('Сервисы')).toBeVisible()
    expect(within(screen.getByRole('heading', { name: 'Старая подписка' }).closest('div')!).getByText('Без категории')).toBeVisible()
  })

  it('chooses currency from a list with RUB by default and requires an expense category', async () => {
    show()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Добавить подписку' }))
    const dialog = screen.getByRole('dialog', { name: 'Новая подписка' })
    const currency = within(dialog).getByLabelText('Валюта')
    expect(currency.tagName).toBe('BUTTON')
    expect(currency).toHaveTextContent('Российский рубль · RUB')
    await user.type(within(dialog).getByLabelText('Название'), 'Музыка')
    await user.type(within(dialog).getByLabelText('Сумма'), '199')
    await user.type(within(dialog).getByLabelText('Первое списание'), '2026-10-20')
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    expect(post).not.toHaveBeenCalled()
    await waitFor(() => expect(toastError).toHaveBeenCalledWith('Выберите категорию расхода.'))

    await user.click(within(dialog).getByLabelText('Категория'))
    const options = screen.getAllByRole('option').map(o => o.textContent)
    expect(options).toContain('Сервисы')
    expect(options).not.toContain('Скрытая')
    expect(options).not.toContain('Зарплата')
    await user.click(screen.getByRole('option', { name: 'Сервисы' }))
    await user.click(currency)
    await user.click(screen.getByRole('option', { name: 'Доллар США · USD' }))
    await user.click(within(dialog).getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body).toMatchObject({ name: 'Музыка', amountMinor: 19900, currency: 'USD', categoryId: 'services' })
  })

  it('keeps the category when pausing', async () => {
    show()
    const user = userEvent.setup()
    const card = (await screen.findByRole('heading', { name: 'Кинотеатр' })).closest('[data-slot="card"]') ?? document.body
    await user.click(within(card as HTMLElement).getAllByRole('button', { name: 'Приостановить' })[0])
    await waitFor(() => expect(patch).toHaveBeenCalledOnce())
    expect(patch.mock.calls[0][1].body).toMatchObject({ categoryId: 'services', paused: true, revision: 2 })
  })
})
