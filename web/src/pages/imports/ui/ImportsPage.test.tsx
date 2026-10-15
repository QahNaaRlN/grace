import { QueryClient, QueryClientProvider, type QueryKey } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import { importBatchQueries } from '@/entities/import-batch'
import { APP_UTC_OFFSET } from '@/shared/config'
import { ImportsPage } from './ImportsPage'

const client = vi.hoisted(() => ({ GET: vi.fn(), PATCH: vi.fn(), POST: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: client }))

it.each([false, true])(
  'keeps home list caches compatible when reviewing imports (warm cache: %s)',
  async (warm) => {
    const row = {
      id: 'cache-row',
      source: {
        sheet: 'Операции',
        row: 8,
        domain: 'transaction',
        raw: {},
        normalized: {
          kind: 'purchase',
          amountMinor: 12000,
          currency: 'RUB',
          date: { precision: 'date', local: '2026-09-02' },
        },
        issues: [],
      },
      decision: { action: 'unresolved' },
      candidates: [],
      history: [],
      published: false,
    }
    const batch = { id: 'cache-batch', revision: 1, counts: {}, source: { sourceId: 'synthetic' } }
    const lists = {
      accounts: [{ id: 'account', name: 'Счёт для проверки', currency: 'RUB' }],
      categories: [{ id: 'category', name: 'Категория для проверки' }],
      'financial-events': [
        {
          id: 'event',
          kind: 'purchase',
          amountMinor: 12000,
          currency: 'RUB',
          description: 'Операция для проверки',
          occurredDate: '2026-09-02',
        },
      ],
    }
    client.GET.mockReset().mockImplementation(async (path: string) => {
      if (path.endsWith('/records/{recordId}')) return { data: row }
      if (path.endsWith('/records')) return { data: { items: [row] } }
      if (path === '/api/imports/batches/{batchId}') return { data: batch }
      const key = path.split('/').at(-1) as keyof typeof lists
      return { data: { items: lists[key] ?? [] } }
    })
    // The list caches the home screens read; import options must never overwrite them.
    const keys: Record<keyof typeof lists, QueryKey> = {
      accounts: accountQueries.list().queryKey,
      categories: categoryQueries.list().queryKey,
      'financial-events': financialEventQueries.list().queryKey,
    }
    const cache = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    })
    if (warm)
      for (const [name, items] of Object.entries(lists))
        cache.setQueryData(keys[name as keyof typeof keys], items)
    render(
      <QueryClientProvider client={cache}>
        <MemoryRouter initialEntries={['/imports?batch=cache-batch']}>
          <ImportsPage />
        </MemoryRouter>
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Операции, строка 8' }))
    await user.click(await screen.findByLabelText('Решение'))
    await user.click(screen.getByRole('option', { name: 'Новая операция' }))
    await user.click(screen.getByLabelText('Счёт операции'))
    await user.click(await screen.findByRole('option', { name: 'Счёт для проверки (RUB)' }))
    await user.click(screen.getByLabelText('Категория'))
    await user.click(await screen.findByRole('option', { name: 'Категория для проверки' }))
    await user.click(screen.getByLabelText('Решение'))
    await user.click(screen.getByRole('option', { name: 'Связать с операцией' }))
    await user.click(screen.getByLabelText('Существующая операция'))
    await user.click(await screen.findByRole('option', { name: /Операция для проверки/ }))
    for (const [name, items] of Object.entries(lists)) {
      const queryKey = keys[name as keyof typeof keys]
      if (warm) expect(cache.getQueryData(queryKey)).toEqual(items)
      else expect(cache.getQueryData(queryKey)).toBeUndefined()
      const loaded = await cache.fetchQuery({ queryKey, queryFn: async () => items })
      expect(loaded.map((item) => item.id)).toEqual(items.map((item) => item.id))
    }
  },
)

it('saves a date-only decision without inventing a timestamp or publishing it', async () => {
  const row = {
    id: 'calendar-row',
    source: {
      sheet: 'Операции',
      row: 8,
      domain: 'transaction',
      raw: {},
      normalized: {
        kind: 'purchase',
        amountMinor: 12000,
        currency: 'RUB',
        date: { precision: 'date', local: '2026-09-02', utc: null },
      },
      issues: [],
    },
    decision: { action: 'unresolved' },
    candidates: [],
    history: [],
    published: false,
  }
  const batch = { id: 'calendar-batch', revision: 1, counts: {}, source: { sourceId: 'synthetic' } }
  client.GET.mockReset().mockImplementation(async (path: string) => {
    if (path.endsWith('/records/{recordId}')) return { data: row }
    if (path.endsWith('/records')) return { data: { items: [row] } }
    if (path === '/api/imports/batches/{batchId}') return { data: batch }
    return { data: { items: [] } }
  })
  client.PATCH.mockReset().mockResolvedValue({ data: batch })
  client.POST.mockReset()
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={cache}>
      <MemoryRouter initialEntries={['/imports?batch=calendar-batch']}>
        <ImportsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: 'Операции, строка 8' }))
  await user.click(await screen.findByLabelText('Решение'))
  await user.click(screen.getByRole('option', { name: 'Новая операция' }))
  expect(screen.getByLabelText('Известен только день, время неизвестно')).toBeChecked()
  expect(screen.getByLabelText('Дата операции')).toHaveValue('2026-09-02')
  expect(
    screen.queryByLabelText(`Дата и время (с часовым поясом, например ${APP_UTC_OFFSET})`),
  ).not.toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Сохранить решение' }))
  await waitFor(() => expect(client.PATCH).toHaveBeenCalledOnce())
  const decision = client.PATCH.mock.calls[0][1].body.decision
  expect(decision.fields.occurredDate).toBe('2026-09-02')
  expect(decision.fields.occurredAt).toBeUndefined()
  expect(client.POST).not.toHaveBeenCalled()
})

it('keeps a saved decision when batch revision refreshes before record details', async () => {
  client.GET.mockReset()
  client.PATCH.mockReset()
  client.POST.mockReset()
  let revision = 1
  let detailReads = 0
  let resolveDetails!: (value: { data: unknown }) => void
  const delayedDetails = new Promise<{ data: unknown }>((resolve) => {
    resolveDetails = resolve
  })
  const source = {
    sheet: 'Операции',
    row: 5,
    domain: 'transaction',
    raw: {},
    normalized: { kind: 'purchase', amountMinor: 1000, currency: 'RUB' },
    issues: [],
  }
  const original = {
    id: 'record',
    source,
    decision: { action: 'unresolved' },
    candidates: [],
    history: [],
    published: false,
  }
  const saved = { ...original, decision: { action: 'exclude', reason: 'Проверенный дубль' } }
  const batch = () => ({
    id: 'batch',
    revision,
    counts: {},
    domainCounts: {},
    createdAt: '2026-10-13T09:00:00Z',
    source: { sourceId: 'synthetic' },
  })
  client.GET.mockImplementation(async (path: string) => {
    if (path.endsWith('/records/{recordId}'))
      return ++detailReads === 1 ? { data: original } : delayedDetails
    if (path.endsWith('/records')) return { data: { items: [original] } }
    if (path === '/api/imports/batches') return { data: { items: [batch()] } }
    if (path === '/api/imports/batches/{batchId}') return { data: batch() }
    return { data: { items: [] } }
  })
  client.PATCH.mockImplementation(async () => {
    revision = 2
    return { data: batch() }
  })
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={cache}>
      <MemoryRouter initialEntries={['/imports?batch=batch']}>
        <ImportsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: 'Операции, строка 5' }))
  await user.click(await screen.findByLabelText('Решение'))
  await user.click(screen.getByRole('option', { name: 'Исключить' }))
  await user.type(screen.getByLabelText('Почему принято это решение'), 'Проверенный дубль')
  await user.click(screen.getByRole('button', { name: 'Сохранить решение' }))
  await waitFor(() =>
    expect(cache.getQueryData(importBatchQueries.batch('batch').queryKey)?.revision).toBe(2),
  )
  expect(screen.getByLabelText('Решение')).toHaveTextContent('Исключить')
  await act(async () => {
    resolveDetails({ data: saved })
  })
  await waitFor(() =>
    expect(screen.getByLabelText('Почему принято это решение')).toHaveValue('Проверенный дубль'),
  )
  expect(screen.getByLabelText('Решение')).toHaveTextContent('Исключить')
  expect(client.PATCH).toHaveBeenCalledOnce()
  expect(client.POST).not.toHaveBeenCalled()
})

it('uploads original CSV bytes and waits for separate review and confirmation', async () => {
  client.GET.mockReset().mockImplementation(async (path: string) =>
    path === '/api/imports/batches/{batchId}'
      ? {
          data: {
            id: 'batch-csv',
            revision: 1,
            counts: {},
            source: { sourceId: 'Финансовое здоровье' },
          },
        }
      : { data: { items: [] } },
  )
  client.POST.mockReset().mockResolvedValue({ data: { id: 'batch-csv' } })
  client.PATCH.mockReset()
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={cache}>
      <MemoryRouter>
        <ImportsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  const file = new File(['CSV original bytes: 9007199254740993,01'], 'Операции.csv', {
    type: 'text/csv',
  })
  const user = userEvent.setup()
  await user.upload(screen.getByLabelText('Таблица CSV или снимок JSON (до 16 MiB)'), file)
  expect(screen.getByLabelText('Название источника')).toHaveValue('Финансовое здоровье')
  await user.click(screen.getByRole('button', { name: 'Сохранить черновик' }))
  await waitFor(() => expect(client.POST).toHaveBeenCalledOnce())
  const [path, options] = client.POST.mock.calls[0]
  expect(path).toBe('/api/imports/csv')
  expect(options.bodySerializer()).toBe(file)
  expect(options.params.query.sourceId).toBe('Финансовое здоровье')
  expect(options.params.query.fileName).toBe('Операции.csv')
  expect(client.PATCH).not.toHaveBeenCalled()
})

it('keeps the selected month and refreshes financial views after confirmation', async () => {
  client.GET.mockReset().mockImplementation(async (path: string) => {
    if (path.endsWith('/month'))
      return {
        data: {
          rows: [],
          unknownDateRows: [],
          totals: [],
          pendingRows: 0,
          outsidePeriod: 0,
          complete: false,
        },
      }
    if (path.endsWith('/records'))
      return {
        data: {
          items: [
            {
              id: 'row',
              source: {
                sheet: 'Операции',
                row: 5,
                domain: 'transaction',
                normalized: {},
                issues: [],
              },
              published: false,
              decision: { action: 'create' },
            },
          ],
        },
      }
    if (path === '/api/imports/batches/{batchId}')
      return { data: { id: 'batch', revision: 1, counts: {}, source: { sourceId: 'source' } } }
    return { data: { items: [] } }
  })
  client.POST.mockReset().mockImplementation(async (path: string) =>
    path.endsWith('/previews')
      ? {
          data: {
            id: 'preview',
            hash: 'hash',
            newEvents: 1,
            links: 0,
            exclusions: 0,
            lines: [],
            changes: [],
            blockers: [],
            expiresAt: '2099-01-01T00:00:00Z',
          },
        }
      : { data: { id: 'receipt', outcomes: [] } },
  )
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const invalidate = vi.spyOn(cache, 'invalidateQueries')
  render(
    <QueryClientProvider client={cache}>
      <MemoryRouter initialEntries={['/imports?batch=batch&month=2026-09']}>
        <ImportsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.click(await screen.findByRole('checkbox'))
  await user.click(screen.getByRole('button', { name: 'Предварительный просмотр (1)' }))
  expect(client.POST).toHaveBeenCalledOnce()
  await user.click(await screen.findByRole('button', { name: 'Подтвердить импорт 1 операций' }))
  await screen.findByText('Подтверждение сохранено')
  expect(screen.getByLabelText('Месяц для сверки')).toHaveValue('2026-09')
  for (const queryKey of [
    importBatchQueries.all(),
    accountQueries.all(),
    financialEventQueries.all(),
    periodSummaryQueries.all(),
    cashPositionQueries.all(),
  ])
    expect(invalidate).toHaveBeenCalledWith({ queryKey })
})
