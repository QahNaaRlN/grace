import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ImportsPage } from './ImportsPage'

const client = vi.hoisted(() => ({ GET: vi.fn(), PATCH: vi.fn(), POST: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: client }))

const batch = {
  id: 'b1', revision: 4, status: 'reviewing', createdAt: '2026-10-13T09:20:00Z', payloadHash: 'h', source: { sourceId: 'Финансовое здоровье' },
  fileName: 'Операции сентябрь.csv', format: 'csv', periodFrom: '2026-09-01', periodTo: '2026-09-28',
  total: 500, reviewed: 412, needsReview: 12, counts: { unresolved: 88, create: 400, exclude: 12 }, domainCounts: { transaction: 500, budget: 3 },
  lastConfirmation: { id: 'c1', at: '2026-10-13T10:00:00Z', createdEvents: 191, outcomes: 200 },
}
const row = (n: number, patch: object = {}) => ({
  id: `r${n}`, published: false, decision: { action: 'unresolved' }, candidates: [], history: [], eventId: null, questions: n === 5 ? 2 : 0,
  source: { sheet: 'Операции', row: n, domain: 'transaction', raw: {}, issues: [], normalized: { kind: 'purchase', amountMinor: 100000 + n, currency: 'RUB', description: `Покупка ${n}`, date: { precision: 'date', local: '2026-09-02' } } },
  ...patch,
})

function show(url = '/imports?batch=b1') {
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter initialEntries={[url]}><ImportsPage /></MemoryRouter></QueryClientProvider>)
}

beforeEach(() => {
  client.GET.mockReset().mockImplementation(async (path: string, init?: { params?: { query?: { cursor?: string } } }) => {
    if (path === '/api/imports/batches') return { data: { items: [batch] } }
    if (path === '/api/imports/batches/{batchId}') return { data: batch }
    if (path.endsWith('/next-unresolved')) return { data: { record: row(9), remaining: 87 } }
    if (path.endsWith('/records/{recordId}')) return { data: row(5) }
    if (path.endsWith('/records')) return { data: { items: Array.from({ length: 20 }, (_, i) => row(Number(init?.params?.query?.cursor ?? 0) + i + 5)), nextCursor: '100', total: 500 } }
    if (path.endsWith('/month')) return { data: { rows: [], unknownDateRows: [], totals: [], pendingRows: 0, outsidePeriod: 0, complete: false } }
    return { data: { items: [] } }
  })
  client.POST.mockReset()
  client.PATCH.mockReset()
})

describe('SPEC-0044 import history and review at scale', () => {
  it('shows file, upload moment, period, progress and the last confirmation in the history', async () => {
    show()
    const history = await screen.findByRole('region', { name: 'История импортов' })
    const entry = await within(history).findByRole('button', { name: /Операции сентябрь\.csv/ })
    expect(entry).toHaveTextContent('Таблица CSV · Финансовое здоровье · загружено 13 окт., 14:20')
    expect(entry).toHaveTextContent('1 сент. 2026 г. – 28 сент. 2026 г.')
    expect(entry).toHaveTextContent('Проверено 412 из 500 · с вопросами 12')
    expect(entry).toHaveTextContent('Подтверждено 13 окт., 15:00 · создано операций: 191')
    expect(entry).toHaveTextContent('На проверке')
  })

  it('pages, sorts, searches and filters rows on the server', async () => {
    show()
    expect(await screen.findByRole('navigation', { name: 'Страницы строк' })).toHaveTextContent('Строки 1–20 из 500')
    expect(screen.getByLabelText('Прогресс проверки')).toHaveAttribute('value', '412')
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Вперёд' }))
    await waitFor(() => expect(client.GET.mock.calls.at(-1)?.[1].params.query).toMatchObject({ cursor: '100', limit: 100, sort: 'row', domain: 'transaction' }))
    await user.click(screen.getByRole('button', { name: 'С вопросами 12' }))
    await user.type(screen.getByLabelText('Поиск по описанию'), 'кофе')
    await user.click(screen.getByRole('button', { name: 'Найти' }))
    await waitFor(() => expect(client.GET.mock.calls.at(-1)?.[1].params.query).toMatchObject({ status: 'needs_review', q: 'кофе' }))
    expect(client.GET.mock.calls.at(-1)?.[1].params.query.cursor).toBeUndefined()
    await user.click(screen.getByRole('button', { name: 'Первичные документы · 3' }))
    await waitFor(() => expect(client.GET.mock.calls.at(-1)?.[1].params.query).toMatchObject({ domain: 'unsupported' }))
  })

  it('previews every row of the filter, not only the page', async () => {
    client.POST.mockResolvedValue({ data: { id: 'p', hash: 'x', newEvents: 0, links: 0, exclusions: 0, lines: [], changes: [], blockers: [], expiresAt: '2099-01-01T00:00:00Z' } })
    show()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Выбрать все по фильтру (500)' }))
    await user.click(screen.getByRole('button', { name: 'Предварительный просмотр (500)' }))
    await waitFor(() => expect(client.POST).toHaveBeenCalledOnce())
    expect(client.POST.mock.calls[0][1].body).toEqual({ expectedRevision: 4, filter: { domain: 'transaction' } })
  })

  it('explains the 1 000-row limit for a filter selection', async () => {
    client.GET.mockImplementation(async (path: string) => {
      if (path.endsWith('/records')) return { data: { items: [row(5)], nextCursor: '100', total: 4200 } }
      if (path === '/api/imports/batches/{batchId}') return { data: batch }
      return { data: { items: [] } }
    })
    show()
    expect(await screen.findByRole('button', { name: 'Выбрать все по фильтру (4200)' })).toBeDisabled()
    expect(screen.getByText(/не больше 1 000 строк/)).toBeVisible()
  })

  it('opens the next unresolved row after a decision is saved', async () => {
    client.PATCH.mockResolvedValue({ data: { ...batch, revision: 5 } })
    show()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Операции, строка 5' }))
    const sheet = await screen.findByRole('dialog', { name: 'Операции, строка 5' })
    await user.click(within(sheet).getByLabelText('Решение'))
    await user.click(screen.getByRole('option', { name: 'Исключить' }))
    await user.type(within(sheet).getByLabelText('Почему принято это решение'), 'Дубль')
    await user.click(within(sheet).getByRole('button', { name: 'Сохранить решение' }))
    await waitFor(() => expect(client.GET.mock.calls.some(([path, init]) => path.endsWith('/next-unresolved') && init.params.query.after === 'r5')).toBe(true))
    await waitFor(() => expect(client.GET.mock.calls.some(([path, init]) => path.endsWith('/records/{recordId}') && init.params.path.recordId === 'r9')).toBe(true))
  })

  it('moves focus between rows with the arrow keys', async () => {
    show()
    const first = await screen.findByRole('button', { name: 'Операции, строка 5' })
    first.focus()
    const user = userEvent.setup()
    await user.keyboard('{ArrowDown}')
    expect(screen.getByRole('button', { name: 'Операции, строка 6' })).toHaveFocus()
    await user.keyboard('{ArrowUp}')
    expect(first).toHaveFocus()
  })
})
