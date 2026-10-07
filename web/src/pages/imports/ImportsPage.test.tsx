import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { ImportsPage } from './ImportsPage'

const client = vi.hoisted(() => ({ GET: vi.fn(), PATCH: vi.fn(), POST: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: client }))

it('keeps a saved decision when batch revision refreshes before record details', async () => {
  let revision = 1
  let detailReads = 0
  let resolveDetails!: (value: { data: unknown }) => void
  const delayedDetails = new Promise<{ data: unknown }>(resolve => { resolveDetails = resolve })
  const source = { sheet: 'Операции', row: 5, domain: 'transaction', raw: {}, normalized: { kind: 'purchase', amountMinor: 1000, currency: 'RUB' }, issues: [] }
  const original = { id: 'record', source, decision: { action: 'unresolved' }, candidates: [], history: [], published: false }
  const saved = { ...original, decision: { action: 'exclude', reason: 'Проверенный дубль' } }
  const batch = () => ({ id: 'batch', revision, counts: {}, domainCounts: {}, source: { sourceId: 'synthetic' } })
  client.GET.mockImplementation(async (path: string) => {
    if (path.endsWith('/records/{recordId}')) return ++detailReads === 1 ? { data: original } : delayedDetails
    if (path.endsWith('/records')) return { data: { items: [original] } }
    if (path === '/api/imports/batches') return { data: { items: [batch()] } }
    if (path === '/api/imports/batches/{batchId}') return { data: batch() }
    return { data: { items: [] } }
  })
  client.PATCH.mockImplementation(async () => { revision = 2; return { data: batch() } })
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={cache}><MemoryRouter initialEntries={['/imports?batch=batch']}><ImportsPage /></MemoryRouter></QueryClientProvider>)
  const user = userEvent.setup()
  await user.click(await screen.findByRole('button', { name: 'Операции, строка 5' }))
  await user.selectOptions(await screen.findByLabelText('Решение'), 'exclude')
  await user.type(screen.getByLabelText('Почему принято это решение'), 'Проверенный дубль')
  await user.click(screen.getByRole('button', { name: 'Сохранить решение' }))
  await waitFor(() => expect(cache.getQueryData<{ revision: number }>(['imports', 'batch', 'batch'])?.revision).toBe(2))
  expect(screen.getByLabelText('Решение')).toHaveValue('exclude')
  await act(async () => { resolveDetails({ data: saved }) })
  await waitFor(() => expect(screen.getByLabelText('Почему принято это решение')).toHaveValue('Проверенный дубль'))
  expect(screen.getByLabelText('Решение')).toHaveValue('exclude')
  expect(client.PATCH).toHaveBeenCalledOnce()
  expect(client.POST).not.toHaveBeenCalled()
})

it('uploads original CSV bytes and waits for separate review and confirmation', async () => {
  client.GET.mockReset().mockImplementation(async (path: string) => path === '/api/imports/batches/{batchId}' ? { data: { id: 'batch-csv', revision: 1, counts: {}, source: { sourceId: 'Финансовое здоровье' } } } : { data: { items: [] } })
  client.POST.mockReset().mockResolvedValue({ data: { id: 'batch-csv' } })
  client.PATCH.mockReset()
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={cache}><MemoryRouter><ImportsPage /></MemoryRouter></QueryClientProvider>)
  const file = new File(['CSV original bytes: 9007199254740993,01'], 'Операции.csv', { type: 'text/csv' })
  const user = userEvent.setup()
  await user.upload(screen.getByLabelText('Таблица CSV или снимок JSON (до 16 MiB)'), file)
  expect(screen.getByLabelText('Название источника')).toHaveValue('Финансовое здоровье')
  await user.click(screen.getByRole('button', { name: 'Сохранить черновик' }))
  await waitFor(() => expect(client.POST).toHaveBeenCalledOnce())
  const [path, options] = client.POST.mock.calls[0]
  expect(path).toBe('/api/imports/csv')
  expect(options.bodySerializer()).toBe(file)
  expect(options.params.query.sourceId).toBe('Финансовое здоровье')
  expect(client.PATCH).not.toHaveBeenCalled()
})

it('keeps the selected month and refreshes financial views after confirmation', async () => {
  client.GET.mockReset().mockImplementation(async (path: string) => {
    if (path.endsWith('/month')) return { data: { rows: [], unknownDateRows: [], totals: [], pendingRows: 0, outsidePeriod: 0, complete: false } }
    if (path.endsWith('/records')) return { data: { items: [{ id: 'row', source: { sheet: 'Операции', row: 5, domain: 'transaction', normalized: {}, issues: [] }, published: false, decision: { action: 'create' } }] } }
    if (path === '/api/imports/batches/{batchId}') return { data: { id: 'batch', revision: 1, counts: {}, source: { sourceId: 'source' } } }
    return { data: { items: [] } }
  })
  client.POST.mockReset().mockImplementation(async (path: string) => path.endsWith('/previews') ? { data: { id: 'preview', hash: 'hash', newEvents: 1, links: 0, exclusions: 0, lines: [], changes: [], blockers: [], expiresAt: '2099-01-01T00:00:00Z' } } : { data: { id: 'receipt', outcomes: [] } })
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const invalidate = vi.spyOn(cache, 'invalidateQueries')
  render(<QueryClientProvider client={cache}><MemoryRouter initialEntries={['/imports?batch=batch&month=2026-09']}><ImportsPage /></MemoryRouter></QueryClientProvider>)
  const user = userEvent.setup()
  await user.click(await screen.findByRole('checkbox'))
  await user.click(screen.getByRole('button', { name: 'Предварительный просмотр (1)' }))
  expect(client.POST).toHaveBeenCalledOnce()
  await user.click(await screen.findByRole('button', { name: 'Подтвердить импорт 1 операций' }))
  await screen.findByText('Подтверждение сохранено')
  expect(screen.getByLabelText('Месяц для сверки')).toHaveValue('2026-09')
  for (const key of ['imports', 'accounts', 'financial-events', 'period-summary', 'cash-position']) expect(invalidate).toHaveBeenCalledWith({ queryKey: [key] })
})
