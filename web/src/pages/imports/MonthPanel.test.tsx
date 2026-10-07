import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { expect, it, vi } from 'vitest'
import { MonthPanel } from './MonthPanel'

const client = vi.hoisted(() => ({ GET: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: client }))

it('shows server month totals and unknown dates without publishing or hiding rows beyond a page', async () => {
  client.GET.mockReset()
  const row = { recordId: 'unknown', sheet: 'Операции', row: 305, description: 'Дата требует уточнения', sourceDate: '', sourceAmountMinor: null, sourceCurrency: 'RUB', action: 'unresolved', published: false, targetKind: '', differences: ['unresolved'], reason: '' }
  client.GET.mockResolvedValue({ data: { complete: false, pendingRows: 191, outsidePeriod: 3, unknownDateRows: [row], rows: Array.from({ length: 191 }, (_, i) => ({ ...row, recordId: String(i), row: i + 5 })), totals: [{ currency: 'RUB', sourceRows: 191, sourceMinor: null, preparedMinor: 76275, confirmedMinor: 0 }] } })
  const open = vi.fn()
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter initialEntries={['/imports?batch=batch&month=2026-09']}><MonthPanel batchId="batch" onOpenRecord={open} /></MemoryRouter></QueryClientProvider>)
  expect(await screen.findByText('Перенос месяца ещё не завершён')).toBeInTheDocument()
  expect(screen.getByText(/Строк месяца: 191/)).toBeInTheDocument()
  expect(screen.getByText('В таблице: Неизвестно')).toBeInTheDocument()
  expect(screen.getByText(/Подготовлено:.*762,75/)).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Операции, строка 305: Дата требует уточнения' }))
  expect(open).toHaveBeenCalledWith('unknown')
  expect(client.GET).toHaveBeenCalledWith('/api/imports/batches/{batchId}/month', { params: { path: { batchId: 'batch' }, query: { month: '2026-09' } } })
})

it('waits for an explicit month and shows report errors without claiming completion', async () => {
  client.GET.mockReset().mockResolvedValue({ error: { code: 'invalid_month' } })
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter><MonthPanel batchId="batch" onOpenRecord={vi.fn()} /></MemoryRouter></QueryClientProvider>)
  expect(client.GET).not.toHaveBeenCalled()
  fireEvent.change(screen.getByLabelText('Месяц для сверки'), { target: { value: '2026-09' } })
  await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Не удалось загрузить сверку'))
  expect(screen.queryByText('Все строки месяца подтверждены')).not.toBeInTheDocument()
})
