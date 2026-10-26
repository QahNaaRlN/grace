import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'
import { statementFixture } from '@/entities/bank-statement/index.testing'
import type { components } from '@/shared/api/generated/schema'
import { StatementPlainDraft } from './StatementPlainDraft'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))
vi.mock('./StatementPages', () => ({
  StatementPages: ({ count, onViewed }: { count: number; onViewed: (n: number) => void }) => (
    <button onClick={() => Array.from({ length: count }, (_, i) => onViewed(i + 1))}>
      Просмотреть все страницы
    </button>
  ),
}))
beforeEach(() => post.mockReset())
function fixture(): components['schemas']['BankStatement'] {
  const reading: components['schemas']['PlainStatementReading'] = {
    version: 'statement.plain.0058.v1',
    parseStatus: 'parsed',
    rawAnswer: '<script>text</script>',
    header: {},
    rows: [
      {
        id: 'row-1',
        raw: { operationDate: '19.10.2026 12:34', accountAmount: '-12,34', description: 'Кафе' },
        operationDate: null,
        bookingDate: null,
        accountAmount: null,
        transactionAmount: null,
        description: 'Кафе',
        cardLast4: null,
        issues: [],
      },
    ],
  }
  return {
    ...statementFixture(),
    readingMode: 'plain_review',
    pageCount: 5,
    plain: {
      sourceSHA256: '0'.repeat(64),
      candidate: reading,
      original: reading,
      digest: 'draft-digest',
      reviewedDigest: '',
    },
  }
}
function surface(s = fixture()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrap = (next: typeof s) => (
    <QueryClientProvider client={client}>
      <StatementPlainDraft statement={next} />
    </QueryClientProvider>
  )
  return { ...render(wrap(s)), wrap }
}
it('reviews all pages and current draft with one document confirmation; raw answer stays text', async () => {
  const s = fixture()
  post.mockResolvedValue({ data: { ...s, revision: 2 } })
  surface(s)
  const user = userEvent.setup()
  expect(screen.getAllByRole('checkbox')).toHaveLength(1)
  const review = screen.getByRole('button', { name: 'Сохранить сверку документа' })
  await user.click(screen.getByRole('checkbox'))
  expect(review).toBeDisabled()
  await user.click(screen.getByRole('button', { name: 'Просмотреть все страницы' }))
  await user.click(review)
  await waitFor(() => expect(post).toHaveBeenCalledOnce())
  expect(post.mock.calls[0][0]).toBe('/api/imports/statements/{statementId}/plain-reviews')
  expect(post.mock.calls[0][1].body).toEqual({
    expectedRevision: 1,
    sourceSHA256: s.sha256,
    draftDigest: 'draft-digest',
    reviewedPages: [1, 2, 3, 4, 5],
    rolesConfirmed: true,
  })
  await user.click(screen.getByRole('button', { name: 'Исходный ответ модели' }))
  expect(screen.getByText('<script>text</script>')).toBeInTheDocument()
  expect(document.querySelector('script')).toBeNull()
})
it('keeps an unsaved cell edit after a concurrent source revision and prevents stale saving', async () => {
  const s = fixture()
  const page = surface(s)
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: /1\. 19\.10/ }))
  const description = screen.getByLabelText('Описание, строка 1')
  await user.clear(description)
  await user.type(description, 'Исправленное кафе')
  page.rerender(page.wrap({ ...s, revision: 2 }))
  expect(description).toHaveValue('Исправленное кафе')
  expect(screen.getByRole('button', { name: 'Сохранить правки таблицы' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Сохранить сверку документа' })).toBeDisabled()
  expect(post).not.toHaveBeenCalled()
})
