import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import type { components } from '@/shared/api/generated/schema'
import { PreviewPanel } from './PreviewPanel'

const preview: components['schemas']['ImportPreview'] = { id: 'p', batchId: 'b', revision: 1, hash: 'hash', expiresAt: '2099-10-06T12:00:00Z', lines: [], changes: [], blockers: [], newEvents: 2, links: 1, exclusions: 0 }
it('requires a separate confirmation click after showing the preview', () => {
  const confirm = vi.fn()
  render(<PreviewPanel preview={preview} pending={false} onConfirm={confirm} />)
  expect(confirm).not.toHaveBeenCalled()
  expect(screen.getByText('Новых операций: 2. Связей: 1. Исключений: 0.')).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Подтвердить импорт 2 операций' }))
  expect(confirm).toHaveBeenCalledOnce()
})
it('cannot confirm server-blocked rows', () => {
  const confirm = vi.fn()
  render(<PreviewPanel preview={{ ...preview, blockers: ['time_confirmation_required'] }} pending={false} onConfirm={confirm} />)
  expect(screen.getByRole('alert')).toHaveTextContent('Уточните время операции и укажите основание.')
  fireEvent.click(screen.getByRole('button', { name: 'Подтвердить импорт 2 операций' }))
  expect(confirm).not.toHaveBeenCalled()
})
it('disables confirmation while a response is pending', () => {
  render(<PreviewPanel preview={{ ...preview, newEvents: 0 }} pending={true} onConfirm={vi.fn()} />)
  expect(screen.getByRole('button', { name: 'Подтвердить решения' })).toBeDisabled()
})
