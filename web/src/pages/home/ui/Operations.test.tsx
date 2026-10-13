import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account, Category, FinancialEvent } from '@/shared/api/types'
import { Operations } from './Operations'

const { get, patch, post } = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, PATCH: patch, POST: post } }))

const account: Account = { id: 'card', name: 'Карта', type: 'debit', currency: 'RUB', balanceReconciled: true, balanceMinor: 0, createdAt: '2026-10-01T00:00:00Z', paymentInstruments: [{ id: 'c', name: 'Карта', bank: 'Т-Банк', last4: '4821' }] }
const categories: Category[] = [{ id: 'food', name: 'Продукты', kind: 'expense', createdAt: '', icon: 'shopping-cart', hidden: false, operationCount: 1 }]
function event(id: string, patch: Partial<FinancialEvent>): FinancialEvent {
  return { id, kind: 'purchase', source: 'manual', merchant: '', description: '', occurredAt: '2026-10-13T09:15:00Z', occurredDate: null, datePrecision: 'instant', createdAt: '2026-10-13T09:15:00Z', categoryId: 'food', categoryName: 'Продукты', categoryIcon: 'shopping-cart', originalEventId: null, revision: 1, editedByUser: false, voidedAt: null, voidReason: null,
    postings: [{ id: `${id}-p`, accountId: 'card', accountName: 'Карта', direction: 'debit', amountMinor: 120000, currency: 'RUB' }], ...patch } as FinancialEvent
}
const page1 = {
  items: [
    event('a', { merchant: 'Пятёрочка' }),
    event('b', { kind: 'income_received', categoryId: null, categoryName: null, categoryIcon: null, description: 'Зарплата', postings: [{ id: 'b-p', accountId: 'card', accountName: 'Карта', direction: 'credit', amountMinor: 6400000, currency: 'RUB' }] }),
    event('c', { merchant: 'Кофейня', occurredAt: null, occurredDate: '2026-10-12', datePrecision: 'date' }),
  ],
  nextCursor: 'next',
  dayTotals: [
    { date: '2026-10-13', currency: 'RUB', inflowMinor: 6400000, outflowMinor: 120000 },
    { date: '2026-10-12', currency: 'RUB', inflowMinor: 0, outflowMinor: 120000 },
  ],
}
const page2 = { items: [event('d', { merchant: 'Аптека', occurredAt: '2026-10-11T07:00:00Z' })], nextCursor: null, dayTotals: [{ date: '2026-10-11', currency: 'RUB', inflowMinor: 0, outflowMinor: 120000 }] }

function show(url = '/operations') {
  render(<MemoryRouter initialEntries={[url]}><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <Operations categories={categories} accounts={[account]} receipts={[]} />
  </QueryClientProvider></MemoryRouter>)
}

beforeEach(() => {
  get.mockReset(); patch.mockReset(); post.mockReset()
  get.mockImplementation(async (_path: string, init: { params: { query: { cursor?: string } } }) => ({ data: init.params.query.cursor ? page2 : page1 }))
})

describe('SPEC-0043 operations list', () => {
  it('groups by day with server totals, shows category icons and signed amounts', async () => {
    show()
    const today = await screen.findByRole('region', { name: /13 октября/ })
    expect(within(today).getByRole('heading')).toHaveTextContent('13 октября, вторник')
    expect(within(today).getByText(/−1\s200,00\s₽ · \+64\s000,00\s₽/)).toBeVisible()
    const row = within(today).getByRole('button', { name: /Пятёрочка/ })
    expect(row).toHaveTextContent('Продукты · Т-Банк *4821')
    expect(row.querySelector('.lucide-shopping-cart')).not.toBeNull()
    expect(within(today).getByRole('button', { name: /Зарплата/ })).toHaveTextContent('+64')
    const yesterday = screen.getByRole('region', { name: /12 октября/ })
    expect(within(yesterday).getByRole('button', { name: /Кофейня/ })).toHaveTextContent('время неизвестно')
    expect(get.mock.calls[0][1].params.query).toMatchObject({ limit: 50 })
  })

  it('loads the next page by cursor', async () => {
    show()
    await userEvent.click(await screen.findByRole('button', { name: 'Показать ещё' }))
    expect(await screen.findByRole('button', { name: /Аптека/ })).toBeVisible()
    expect(get.mock.calls[1][1].params.query.cursor).toBe('next')
    expect(screen.queryByRole('button', { name: 'Показать ещё' })).not.toBeInTheDocument()
    expect(screen.getByText('Показано операций: 4')).toBeVisible()
  })

  it('opens a sheet with only the facts that exist', async () => {
    show()
    await userEvent.click(await screen.findByRole('button', { name: /Пятёрочка/ }))
    const sheet = screen.getByRole('dialog')
    expect(within(sheet).getByText('Продавец')).toBeVisible()
    expect(within(sheet).queryByText('Описание')).not.toBeInTheDocument()
    expect(within(sheet).queryByText(/Не указано/)).not.toBeInTheDocument()
    expect(within(sheet).getByText('Ручной ввод')).toBeVisible()
    expect(within(sheet).getByRole('button', { name: 'Добавить состав' })).toBeVisible()
  })

  it('sends search and filters to the server from the URL', async () => {
    show('/operations?q=кофе&account=card&kind=purchase&from=2026-10-01')
    await screen.findByRole('button', { name: /Пятёрочка/ })
    expect(get.mock.calls[0][1].params.query).toMatchObject({ q: 'кофе', accountId: 'card', kind: 'purchase', from: '2026-10-01' })
    expect(screen.getByRole('button', { name: 'Сбросить фильтры' })).toBeVisible()
  })

  it('says when nothing matches the filters', async () => {
    get.mockResolvedValue({ data: { items: [], nextCursor: null, dayTotals: [] } })
    show('/operations?q=несуществующее')
    expect(await screen.findByText('По этим условиям операций нет.')).toBeVisible()
    await waitFor(() => expect(get).toHaveBeenCalledOnce())
  })
})

describe('SPEC-0041 operation corrections', () => {
  async function openSheet(name: RegExp) {
    show()
    await userEvent.click(await screen.findByRole('button', { name }))
    return screen.getByRole('dialog')
  }

  it('sends only the changed fields with the revision and an idempotency key', async () => {
    patch.mockResolvedValue({ data: event('a', { merchant: 'Пятёрочка', revision: 2, editedByUser: true, postings: [{ id: 'a-p', accountId: 'card', accountName: 'Карта', direction: 'debit', amountMinor: 150000, currency: 'RUB' }] }) })
    const sheet = await openSheet(/Пятёрочка/)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Изменить' }))
    const form = screen.getByRole('dialog', { name: 'Изменить операцию' })
    const amount = within(form).getByLabelText('Сумма')
    await userEvent.clear(amount)
    await userEvent.type(amount, '1500')
    await userEvent.click(within(form).getByRole('button', { name: 'Сохранить изменения' }))
    await waitFor(() => expect(patch).toHaveBeenCalledOnce())
    const [path, init] = patch.mock.calls[0]
    expect(path).toBe('/api/v1/financial-events/{eventId}')
    expect(init.body).toEqual({ revision: 1, amountMinor: 150000 })
    expect(init.params.header['Idempotency-Key']).toBeTruthy()
    expect(await screen.findByText('Изменена вручную')).toBeVisible()
  })

  it('explains which link blocks a money change', async () => {
    patch.mockResolvedValue({ error: { code: 'linked', links: ['возврат по этой покупке'], message: 'Сначала снимите связь: возврат по этой покупке.' } })
    const sheet = await openSheet(/Пятёрочка/)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Изменить' }))
    const form = screen.getByRole('dialog', { name: 'Изменить операцию' })
    const amount = within(form).getByLabelText('Сумма')
    await userEvent.clear(amount)
    await userEvent.type(amount, '900')
    await userEvent.click(within(form).getByRole('button', { name: 'Сохранить изменения' }))
    expect(await within(form).findByText('Сначала снимите связь: возврат по этой покупке.')).toBeVisible()
  })

  it('voids only with a reason and then shows the operation as voided', async () => {
    post.mockResolvedValue({ data: event('a', { merchant: 'Пятёрочка', revision: 2, voidedAt: '2026-10-13T10:00:00Z', voidReason: 'Дубль из выписки' }) })
    const sheet = await openSheet(/Пятёрочка/)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Отменить операцию' }))
    const confirm = screen.getByRole('dialog', { name: 'Отменить операцию?' })
    const submit = within(confirm).getByRole('button', { name: 'Отменить операцию' })
    expect(submit).toBeDisabled()
    await userEvent.type(within(confirm).getByLabelText('Причина'), 'Дубль из выписки')
    await userEvent.click(submit)
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body).toEqual({ revision: 1, reason: 'Дубль из выписки' })
    expect(await screen.findByText('Операция отменена')).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Изменить' })).not.toBeInTheDocument()
  })

  it('lists voided operations on request', async () => {
    show()
    await screen.findByRole('button', { name: /Пятёрочка/ })
    await userEvent.click(screen.getByRole('checkbox', { name: 'Показать отменённые' }))
    await waitFor(() => expect(get.mock.calls.at(-1)?.[1].params.query).toMatchObject({ voided: true }))
  })

  it('shows the history with names instead of identifiers', async () => {
    const edited = event('e', { merchant: 'Исправленная', revision: 3, editedByUser: true })
    get.mockImplementation(async (path: string) => path.endsWith('/history')
      ? { data: { items: [
        { revision: 3, createdAt: '2026-10-13T10:00:00Z', changes: [{ field: 'category', before: null, after: 'food' }] },
        { revision: 2, createdAt: '2026-10-13T09:30:00Z', changes: [{ field: 'amountMinor', before: 100000, after: 120000 }, { field: 'kind', before: 'income_received', after: 'purchase' }] },
      ] } }
      : { data: { items: [edited], nextCursor: null, dayTotals: [] } })
    const sheet = await openSheet(/Исправленная/)
    await userEvent.click(within(sheet).getByRole('button', { name: /История изменений · 2/ }))
    expect(await within(sheet).findByText('Категория: — → Продукты')).toBeVisible()
    expect(within(sheet).getByText(/Сумма: 1\s000,00\s₽ → 1\s200,00\s₽/)).toBeVisible()
    expect(within(sheet).getByText('Вид: Доход → Покупка')).toBeVisible()
  })
})
