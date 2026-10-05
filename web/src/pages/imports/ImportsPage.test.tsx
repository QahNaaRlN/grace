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
