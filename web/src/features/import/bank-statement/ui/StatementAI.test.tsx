import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'

import { statementFixture, statementPreviewFixture } from '@/entities/bank-statement/index.testing'
import { StatementAI } from './StatementAI'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))
beforeEach(() => {
  post.mockReset()
})
it('prepares visible policy/cost/input first; only explicit consent starts extraction, never financial confirmation', async () => {
  const preview = statementPreviewFixture()
  post.mockImplementation(async (path: string) =>
    path.endsWith('/extraction-previews')
      ? { data: preview }
      : path.endsWith('/extraction-consents')
        ? { data: { id: 'consent-1', previewId: preview.id } }
        : {
            data: {
              id: 'attempt-1',
              statementId: 'statement-1',
              status: 'completed',
              errorCode: null,
              batchId: 'batch-1',
            },
          },
  )
  render(
    <QueryClientProvider client={new QueryClient()}>
      <StatementAI statement={statementFixture()} />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  expect(post).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: 'Подготовить AI-распознавание' }))
  expect(await screen.findByText(/google\/gemini-2.5-flash/)).toHaveTextContent('0.0080')
  expect(post).toHaveBeenCalledTimes(1)
  const send = screen.getByRole('button', { name: 'Передать данные и подготовить черновик' })
  expect(send).toBeDisabled()
  await user.click(screen.getByRole('checkbox', { name: /Разрешаю передачу показанных данных/ }))
  await user.click(send)
  await waitFor(() => expect(post).toHaveBeenCalledTimes(3))
  expect(post.mock.calls.map((call) => call[0])).toEqual([
    '/api/imports/statements/{statementId}/extraction-previews',
    '/api/imports/statements/{statementId}/extraction-consents',
    '/api/imports/statements/{statementId}/extractions',
  ])
  expect(post.mock.calls[2][1].body).toEqual({
    expectedRevision: 1,
    previewId: 'preview-1',
    previewHash: 'preview-hash',
    consentId: 'consent-1',
  })
  expect(await screen.findByRole('status')).toHaveTextContent(
    'Нужна проверка страниц, полей и строк',
  )
})
it('keeps the manual path when statement AI is disabled and never requests consent or inference', async () => {
  post.mockResolvedValue({ error: { code: 'ai_disabled', message: 'disabled' } })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <StatementAI statement={statementFixture()} />
    </QueryClientProvider>,
  )
  await userEvent.click(screen.getByRole('button', { name: 'Подготовить AI-распознавание' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Доступна ручная подготовка строк')
  expect(post).toHaveBeenCalledTimes(1)
})
