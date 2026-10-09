import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ReceiptPhoto } from './ReceiptPhoto'

const mocked = vi.hoisted(() => ({ GET: vi.fn(), POST: vi.fn(), PATCH: vi.fn(), DELETE: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: mocked }))
vi.mock('@/shared/model/auth-session', () => ({ csrfToken: 'synthetic-csrf' }))
const fields = { seller: 'Synthetic store', issuedAt: null, issuedDate: '2026-09-12', currency: 'RUB', total: '180.00', fiscalDocumentNumber: null, items: [], accountId: '', categoryId: null, action: '', existingEventId: '', candidateReason: '' }
const draft = { id: 'synthetic-draft', revision: 1, status: 'uploaded', fields, extractions: [], warnings: [], photoStatus: 'stored', expiresAt: '2026-10-10', receiptId: null, eventId: null }
beforeEach(() => {
  vi.resetAllMocks()
  history.replaceState(null, '', '/')
  mocked.GET.mockImplementation(async (path: string) => ({ data: path === '/api/assistant/capabilities' ? { receipt: { enabled: true, model: 'fixture', provider: 'synthetic', consentGranted: false, policyHash: 'synthetic-policy', dataDescription: 'Фото без метаданных.', privacy: 'Тестовый провайдер.' } } : { items: [] } }))
})
describe('receipt photo confirmation boundary', () => {
  it('does not turn a foreign or unknown currency into RUB when the amount is edited', async () => {
    mocked.POST.mockResolvedValue({ data: { ...draft, fields: { ...fields, currency: 'USD' } } })
    render(<ReceiptPhoto />)
    await userEvent.upload(await screen.findByLabelText('Фото чека'), new File(['synthetic'], 'receipt.jpg', { type: 'image/jpeg' }))
    const currency = await screen.findByLabelText('Валюта чека')
    expect(currency).toHaveValue('USD')
    const total = screen.getByLabelText('Итог')
    await userEvent.clear(total); await userEvent.type(total, '10.00')
    expect(currency).toHaveValue('USD')
    await userEvent.selectOptions(currency, 'RUB')
    expect(currency).toHaveValue('RUB')
  })
  it('uploads without inference or finance publication and keeps recognition behind separate consent', async () => {
    mocked.POST.mockResolvedValue({ data: draft })
    render(<ReceiptPhoto />)
    const upload = await screen.findByLabelText('Фото чека')
    await userEvent.upload(upload, new File(['synthetic-image'], 'receipt.jpg', { type: 'image/jpeg' }))
    await screen.findByText('Черновик — финансовые данные ещё не изменены.')
    expect(mocked.POST).toHaveBeenCalledTimes(1)
    expect(mocked.POST.mock.calls[0][0]).toBe('/api/receipt-drafts')
    expect(screen.getByRole('button', { name: 'Распознать чек' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Подтвердить расход' })).not.toBeInTheDocument()
  })
  it('requires an explicit consent action naming the receipt scope', async () => {
    mocked.POST.mockResolvedValue({ data: { granted: true } })
    render(<ReceiptPhoto />)
    await userEvent.click(await screen.findByRole('button', { name: 'Разрешить передачу фото для распознавания' }))
    await waitFor(() => expect(mocked.POST).toHaveBeenCalledWith('/api/assistant/consents', expect.objectContaining({ body: { scope: 'receipt', policyHash: 'synthetic-policy' } })))
    expect(mocked.POST).toHaveBeenCalledTimes(1)
  })
})
