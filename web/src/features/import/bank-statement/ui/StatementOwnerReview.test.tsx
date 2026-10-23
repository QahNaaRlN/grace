import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'

import { bankStatementQueries } from '@/entities/bank-statement'
import { statementFixture } from '@/entities/bank-statement/index.testing'
import { StatementOwnerReview } from './StatementOwnerReview'

const { patch, get } = vi.hoisted(() => ({ patch: vi.fn(), get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { PATCH: patch, GET: get } }))
beforeEach(() => {
  patch.mockReset()
  get.mockReset()
})
const rolesLabel = 'Проверены роли дат, валютных колонок, оборотов и остатков'
function surface(fixture = statementFixture()) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 30_000 } },
  })
  const detail = bankStatementQueries.detail(fixture.id)
  client.setQueryData(detail.queryKey, fixture)
  function Surface() {
    const source = useQuery({ ...detail, staleTime: Infinity })
    return source.data ? <StatementOwnerReview statement={source.data} /> : null
  }
  render(
    <QueryClientProvider client={client}>
      <Surface />
    </QueryClientProvider>,
  )
  return { client, detail }
}
it('requires a fresh owner review after materialization resets server roles and pages', async () => {
  const fixture = { ...statementFixture(), rolesConfirmed: true, reviewedPages: [1] }
  const { client, detail } = surface(fixture)
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Основание проверки'), 'Previous source review')
  const regrouped = { ...fixture, revision: 2, rolesConfirmed: false, reviewedPages: [] }
  act(() => client.setQueryData(detail.queryKey, regrouped))

  const save = screen.getByRole('button', { name: 'Сохранить проверку' })
  await waitFor(() => expect(save).toBeDisabled())
  expect(screen.getByRole('status')).toHaveTextContent('предыдущей версии')
  await user.click(save)
  expect(patch).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: 'Загрузить актуальную проверку' }))
  expect(screen.getByRole('checkbox', { name: rolesLabel })).not.toBeChecked()
  expect(screen.getByRole('checkbox', { name: 'Страница 1' })).not.toBeChecked()
  expect(screen.getByLabelText('Основание проверки')).toHaveValue('')
  expect(save).toBeDisabled()

  const saved = { ...regrouped, revision: 3, rolesConfirmed: true, reviewedPages: [1] }
  patch.mockResolvedValue({ data: saved })
  get.mockResolvedValue({ data: saved })
  await user.click(screen.getByRole('checkbox', { name: rolesLabel }))
  await user.click(screen.getByRole('checkbox', { name: 'Страница 1' }))
  await user.type(screen.getByLabelText('Основание проверки'), 'Fresh review of regrouped source')
  await user.click(save)
  await waitFor(() => expect(patch).toHaveBeenCalledOnce())
  expect(patch.mock.calls[0][1].body).toMatchObject({
    expectedRevision: 2,
    rolesConfirmed: true,
    reviewedPages: [1],
    reason: 'Fresh review of regrouped source',
  })
  await waitFor(() =>
    expect(screen.queryByRole('button', { name: 'Загрузить актуальную проверку' })).toBeNull(),
  )
})
it('preserves edits after a 409 but prevents resubmitting them against the new revision', async () => {
  const fixture = statementFixture()
  const next = { ...fixture, revision: 2 }
  patch.mockResolvedValue({ error: { code: 'revision_conflict', message: 'stale' } })
  get.mockResolvedValue({ data: next })
  surface(fixture)
  const user = userEvent.setup()
  await user.click(screen.getByRole('checkbox', { name: rolesLabel }))
  await user.click(screen.getByRole('checkbox', { name: 'Страница 1' }))
  await user.type(
    screen.getByLabelText('Основание проверки'),
    'Unsaved review retained at conflict',
  )
  const save = screen.getByRole('button', { name: 'Сохранить проверку' })
  await user.click(save)
  expect(await screen.findByRole('alert')).toHaveTextContent('Ввод сохранён')
  expect(get).toHaveBeenCalledOnce()
  expect(screen.getByLabelText('Основание проверки')).toHaveValue(
    'Unsaved review retained at conflict',
  )
  expect(screen.getByRole('checkbox', { name: rolesLabel })).toBeChecked()
  expect(screen.getByRole('checkbox', { name: 'Страница 1' })).toBeChecked()
  expect(save).toBeDisabled()
  await user.click(save)
  expect(patch).toHaveBeenCalledOnce()
  expect(patch.mock.calls[0][1].body.expectedRevision).toBe(1)
})
