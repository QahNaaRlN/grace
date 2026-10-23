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
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 30_000 } },
  })
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
  expect(
    screen.getByRole('button', { name: 'Сохранить черновик без записи операций' }),
  ).toBeDisabled()
  await user.click(
    screen.getByRole('button', {
      name: 'Актуальный черновик проверен — продолжить с моими правками',
    }),
  )
  await user.click(screen.getByRole('button', { name: 'Сохранить черновик без записи операций' }))
  await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
  expect(post.mock.calls[0][1].body.expectedRevision).toBe(1)
  expect(post.mock.calls[1][1].body.expectedRevision).toBe(2)
  expect(post.mock.calls[0][1].params.header['Idempotency-Key']).not.toBe(
    post.mock.calls[1][1].params.header['Idempotency-Key'],
  )
})

it('opens all retained rows and service groups, cancels safely and changes only the chosen header', async () => {
  const fixture = statementFixture()
  const { initial } = await import('../model/manual-fields')
  const proposal = initial()
  proposal.rows[0].operationDate = ['date']
  proposal.rows.push({ ...proposal.rows[0], operationDate: ['date'] })
  proposal.service = [
    { spans: [], reason: 'First retained group' },
    { spans: [], reason: 'Second retained group' },
  ]
  fixture.batchId = 'existing-batch'
  fixture.draftMapping = {
    version: 'statement-review-mapping.v1',
    revision: 1,
    status: 'complete',
    proposal,
    unmappedSpanIds: [],
    unclassifiedSpanIds: [],
    issues: [],
  }
  post.mockResolvedValue({ data: { ...fixture, revision: 2 } })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={client}>
      <StatementManualDraft statement={fixture} />
    </QueryClientProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: 'Исправить группировку до подтверждения' }))
  expect(screen.getByText('Строк в форме: 2. Служебных групп: 2.')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '19.10.2026' })).toHaveAttribute('aria-pressed', 'true')
  await user.click(screen.getByRole('button', { name: 'Отменить исправление' }))
  expect(post).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: 'Исправить группировку до подтверждения' }))
  await user.click(screen.getByLabelText('Заполняемое поле'))
  await user.click(screen.getByRole('option', { name: 'Оборот списаний' }))
  await user.click(screen.getByRole('button', { name: '19.10.2026' }))
  await user.type(
    screen.getByLabelText('Основание группировки строк и служебных фрагментов'),
    'Synthetic targeted field edit',
  )
  await user.click(screen.getByRole('button', { name: 'Сохранить черновик без записи операций' }))
  await waitFor(() => expect(post).toHaveBeenCalledOnce())
  const sent = post.mock.calls[0][1].body.proposal
  expect(sent.rows).toEqual(proposal.rows)
  expect(sent.service).toEqual(proposal.service)
  expect(sent.header).toEqual({ ...proposal.header, debitTurnover: ['date'] })
})

it('shows the legacy warning and prevents replacing an unavailable existing draft with a blank row', async () => {
  const fixture = statementFixture()
  fixture.batchId = 'existing-batch'
  const client = new QueryClient()
  const { rerender } = render(
    <QueryClientProvider client={client}>
      <StatementManualDraft statement={fixture} />
    </QueryClientProvider>,
  )
  expect(
    screen.getByRole('button', { name: 'Исправить группировку до подтверждения' }),
  ).toBeDisabled()
  const { initial } = await import('../model/manual-fields')
  fixture.draftMapping = {
    version: 'statement-review-mapping.v1',
    revision: 1,
    status: 'reconstructed_partial',
    proposal: initial(),
    unmappedSpanIds: null,
    unclassifiedSpanIds: ['date'],
    issues: ['legacy_service_mapping_missing'],
  }
  rerender(
    <QueryClientProvider client={client}>
      <StatementManualDraft statement={fixture} />
    </QueryClientProvider>,
  )
  await userEvent
    .setup()
    .click(screen.getByRole('button', { name: 'Исправить группировку до подтверждения' }))
  expect(screen.getByText(/Служебная разметка старого черновика не сохранена/)).toBeInTheDocument()
})
