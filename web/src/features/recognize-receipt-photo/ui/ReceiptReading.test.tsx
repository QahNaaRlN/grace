import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'
import type { components } from '@/shared/api/generated/schema'
import { ReceiptReading } from './ReceiptReading'
const api = vi.hoisted(() => ({ prepare: vi.fn(), extract: vi.fn(), consent: vi.fn() }))
vi.mock('../api/receipt-drafts', () => ({
  prepareReceiptReading: api.prepare,
  extractReceipt: api.extract,
  setReceiptConsent: api.consent,
}))
vi.mock('@/shared/ui/private-source-image', () => ({
  PrivateSourceImage: ({ label }: { label: string }) => <div>{label}</div>,
}))
const draft: components['schemas']['ReceiptDraft'] = {
  id: 'synthetic',
  revision: 1,
  status: 'uploaded',
  photoStatus: 'stored',
  pages: [{ number: 1, width: 100, height: 100, sha256: 'a'.repeat(64), contentType: 'image/png' }],
  fields: {
    seller: '',
    issuedAt: null,
    issuedDate: null,
    currency: '',
    total: '',
    fiscalDocumentNumber: null,
    items: [],
    accountId: '',
    categoryId: null,
    action: '',
    existingEventId: '',
    candidateReason: '',
  },
  extractions: [],
  warnings: [],
  expiresAt: '2026-10-26',
  receiptId: null,
  eventId: null,
}
const plan: components['schemas']['ReceiptReadingPlan'] = {
  revision: 1,
  hash: 'plan-v1',
  sourceSHA256: 'source',
  model: 'fixture/vision',
  provider: 'Fixture EU',
  profile: 'document_reading',
  groupRevision: 'fixture-v1',
  maximumMicros: 10000,
  consentPolicyHash: 'policy-v1',
  pages: draft.pages!,
}
beforeEach(() => {
  vi.resetAllMocks()
  api.prepare.mockResolvedValue(plan)
  api.extract.mockResolvedValue({ ...draft, revision: 3, status: 'reviewing' })
  api.consent.mockResolvedValue(undefined)
})
it('selects the model and ceiling before explicit consent; keeps source visible after one reading', async () => {
  const changed = vi.fn()
  render(<ReceiptReading draft={draft} busy={false} onChanged={changed} onBusyChange={vi.fn()} />)
  expect(screen.getByText('Страница 1 чека')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Подготовить чтение фото или PDF' }))
  await screen.findByText(/fixture\/vision/)
  const read = screen.getByRole('button', { name: 'Передать страницы и распознать чек' })
  expect(read).toBeDisabled()
  expect(api.extract).not.toHaveBeenCalled()
  expect(api.consent).not.toHaveBeenCalled()
  await userEvent.click(screen.getByRole('checkbox'))
  await userEvent.click(read)
  await waitFor(() => expect(changed).toHaveBeenCalledOnce())
  expect(api.consent).toHaveBeenCalledWith('policy-v1', true, 1)
  expect(api.extract).toHaveBeenCalledOnce()
  expect(screen.getByText('Страница 1 чека')).toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Передать страницы и распознать чек' }),
  ).not.toBeInTheDocument()
})
it('a revised draft invalidates a prepared reading without sending it', async () => {
  const props = { busy: false, onChanged: vi.fn(), onBusyChange: vi.fn() }
  const ui = render(<ReceiptReading draft={draft} {...props} />)
  await userEvent.click(screen.getByRole('button', { name: 'Подготовить чтение фото или PDF' }))
  await screen.findByText(/fixture\/vision/)
  await userEvent.click(screen.getByRole('checkbox'))
  ui.rerender(<ReceiptReading draft={{ ...draft, revision: 2 }} {...props} />)
  expect(screen.getByRole('button', { name: 'Передать страницы и распознать чек' })).toBeDisabled()
  expect(api.extract).not.toHaveBeenCalled()
})
