import { render as renderElement, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { queryHarness } from '@/test/query-harness'
import { ReceiptPhoto } from './ReceiptPhoto'

const render = (element: React.ReactElement) =>
  renderElement(element, { wrapper: queryHarness().wrapper })

const mocked = vi.hoisted(() => ({ GET: vi.fn(), POST: vi.fn(), PATCH: vi.fn(), DELETE: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: mocked }))
vi.mock('@/shared/ui/private-source-image', () => ({
  PrivateSourceImage: () => <div>Приватный источник</div>,
}))
const fields = {
  seller: 'Synthetic store',
  issuedAt: null,
  issuedDate: '2026-09-12',
  currency: 'RUB',
  total: '180.00',
  fiscalDocumentNumber: null,
  items: [],
  accountId: '',
  categoryId: null,
  action: '',
  existingEventId: '',
  candidateReason: '',
}
const draft = {
  id: 'synthetic-draft',
  revision: 1,
  status: 'uploaded',
  fields,
  extractions: [],
  warnings: [],
  photoStatus: 'stored',
  expiresAt: '2026-10-10',
  receiptId: null,
  eventId: null,
}
beforeEach(() => {
  vi.resetAllMocks()
  history.replaceState(null, '', '/')
  mocked.GET.mockImplementation(async (path: string) => ({
    data:
      path === '/api/assistant/capabilities'
        ? {
            receipt: {
              enabled: true,
              model: 'fixture',
              provider: 'synthetic',
              consentGranted: false,
              policyHash: 'synthetic-policy',
              dataDescription: 'Фото без метаданных.',
              privacy: 'Тестовый провайдер.',
            },
          }
        : { items: [] },
  }))
})
describe('receipt photo confirmation boundary', () => {
  it('keeps original recognition text separate from local and saved edits', async () => {
    const originalText = 'SYNTHETIC SHOP\n07:08:09\nComplimentary 2 x 0.00 = 0.00'
    const extractions = [
      {
        fields: { ...fields, issuedAt: '07:08:09' },
        sourceText: originalText,
        model: 'fixture',
        provider: 'fixture',
        costMicros: '100',
      },
    ]
    mocked.POST.mockResolvedValueOnce({ data: { ...draft, status: 'extracted', extractions } })
    mocked.PATCH.mockResolvedValue({
      data: {
        ...draft,
        revision: 2,
        status: 'reviewed',
        fields: { ...fields, seller: 'Owner edit' },
        extractions,
      },
    })
    mocked.POST.mockResolvedValue({
      data: {
        id: 'preview',
        hash: 'hash',
        revision: 2,
        action: 'create',
        amountMinor: '18000',
        accountId: '',
        candidates: [],
        warnings: [],
      },
    })
    render(<ReceiptPhoto />)
    await userEvent.upload(
      await screen.findByLabelText('Фото или PDF чека'),
      new File(['synthetic'], 'receipt.jpg', { type: 'image/jpeg' }),
    )
    const summary = await screen.findByText('Исходный текст распознавания')
    await userEvent.click(summary)
    const original = screen.getByText(
      (_, node) => node?.tagName === 'P' && node.textContent === originalText,
    )
    expect(screen.getByLabelText('Дата чека')).toHaveValue('2026-09-12')
    expect(screen.queryByText(/время неизвестно/)).not.toBeInTheDocument()
    await userEvent.clear(screen.getByLabelText('Магазин'))
    await userEvent.type(screen.getByLabelText('Магазин'), 'Owner edit')
    expect(original.textContent).toBe(originalText)
    await userEvent.click(screen.getByRole('button', { name: 'Проверить и показать результат' }))
    await waitFor(() => expect(mocked.PATCH).toHaveBeenCalledOnce())
    await screen.findByRole('button', { name: 'Подтвердить расход' })
    expect(
      screen.getByText((_, node) => node?.tagName === 'P' && node.textContent === originalText)
        .textContent,
    ).toBe(originalText)
    expect(mocked.PATCH.mock.calls[0][1].body.fields.seller).toBe('Owner edit')
    expect(
      mocked.POST.mock.calls.some(
        ([path]) => path.includes('/extract') || path.includes('/confirmations'),
      ),
    ).toBe(false)
  })

  it('does not turn a foreign or unknown currency into RUB when the amount is edited', async () => {
    mocked.POST.mockResolvedValue({ data: { ...draft, fields: { ...fields, currency: 'USD' } } })
    render(<ReceiptPhoto />)
    await userEvent.upload(
      await screen.findByLabelText('Фото или PDF чека'),
      new File(['synthetic'], 'receipt.jpg', { type: 'image/jpeg' }),
    )
    const currency = await screen.findByLabelText('Валюта чека')
    expect(currency).toHaveTextContent('USD')
    const total = screen.getByLabelText('Итог')
    await userEvent.clear(total)
    await userEvent.type(total, '10.00')
    expect(currency).toHaveTextContent('USD')
    await userEvent.click(currency)
    await userEvent.click(screen.getByRole('option', { name: 'Российский рубль · RUB' }))
    expect(currency).toHaveTextContent('RUB')
  })
  it('uploads without inference or finance publication and keeps recognition behind separate consent', async () => {
    mocked.POST.mockResolvedValue({ data: draft })
    render(<ReceiptPhoto />)
    const upload = await screen.findByLabelText('Фото или PDF чека')
    await userEvent.upload(
      upload,
      new File(['synthetic-image'], 'receipt.jpg', { type: 'image/jpeg' }),
    )
    await screen.findByText('Черновик — финансовые данные ещё не изменены.')
    expect(mocked.POST).toHaveBeenCalledTimes(1)
    expect(mocked.POST.mock.calls[0][0]).toBe('/api/receipt-drafts')
    expect(
      screen.queryByRole('button', { name: 'Передать страницы и распознать чек' }),
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Подготовить чтение фото или PDF' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Подтвердить расход' })).not.toBeInTheDocument()
  })
  it('does not grant broad consent before selecting the actual document model', async () => {
    render(<ReceiptPhoto />)
    await screen.findByLabelText('Фото или PDF чека')
    expect(
      screen.queryByRole('button', { name: 'Разрешить передачу фото для распознавания' }),
    ).not.toBeInTheDocument()
    await waitFor(() => expect(mocked.POST).not.toHaveBeenCalled())
  })
  it('keeps upload and manual fields usable when no model is eligible', async () => {
    mocked.GET.mockResolvedValue({
      data: { receipt: { enabled: false, consentGranted: false }, items: [] },
    })
    mocked.POST.mockResolvedValue({ data: draft })
    render(<ReceiptPhoto />)
    await userEvent.upload(
      await screen.findByLabelText('Фото или PDF чека'),
      new File(['synthetic'], 'receipt.jpg', { type: 'image/jpeg' }),
    )
    expect(await screen.findByLabelText('Магазин')).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Подготовить чтение фото или PDF' })).toBeDisabled()
    expect(mocked.POST).toHaveBeenCalledOnce()
  })
})
