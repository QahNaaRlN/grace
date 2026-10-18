import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'

import { bankStatementQueries } from '@/entities/bank-statement'
import { statementFixture } from '@/entities/bank-statement/index.testing'
import { StatementManualDraft } from './StatementManualDraft'

const { post, get } = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, GET: get } }))
beforeEach(() => {
  post.mockReset()
  get.mockReset()
})
it('retains source selection and reason at 409, reads the new revision and retries without invalidating on failure', async () => {
  const fixture = statementFixture()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(bankStatementQueries.detail(fixture.id).queryKey, fixture)
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  get.mockResolvedValue({ data: { ...fixture, revision: 2 } })
  post
    .mockResolvedValueOnce({ error: { code: 'revision_conflict', message: 'stale' } })
    .mockResolvedValueOnce({ data: { ...fixture, revision: 3, batchId: 'batch-1' } })
  function Surface() {
    const source = useQuery({ ...bankStatementQueries.detail(fixture.id), staleTime: Infinity })
    return source.data ? <StatementManualDraft statement={source.data} /> : null
  }
  render(
    <QueryClientProvider client={client}>
      <Surface />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: '19.10.2026' }))
  const reason = screen.getByLabelText('Основание группировки строк и служебных фрагментов')
  await user.type(reason, 'Synthetic source mapping review')
  await user.click(screen.getByRole('button', { name: 'Сохранить черновик без записи операций' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Ввод сохранён')
  expect(reason).toHaveValue('Synthetic source mapping review')
  expect(screen.getByRole('button', { name: '19.10.2026' })).toHaveAttribute('aria-pressed', 'true')
  expect(invalidate).not.toHaveBeenCalled()
  expect(get).toHaveBeenCalledOnce()
  await user.click(screen.getByRole('button', { name: 'Сохранить черновик без записи операций' }))
  await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
  expect(post.mock.calls[0][1].body.expectedRevision).toBe(1)
  expect(post.mock.calls[1][1].body.expectedRevision).toBe(2)
  expect(post.mock.calls[0][1].params.header['Idempotency-Key']).not.toBe(
    post.mock.calls[1][1].params.header['Idempotency-Key'],
  )
})
