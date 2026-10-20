import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it } from 'vitest'

import type { components } from '@/shared/api/generated/schema'
import { statementFixture } from '../index.testing'
import { StatementReconciliation } from './Reconciliation'
import { MappingCoverage } from './MappingCoverage'

it('displays server-selected and computed totals and opens source context without inventing missing totals', async () => {
  const layout = statementFixture().layout!
  layout.pages.push({ number: 2, width: 600, height: 800 })
  layout.spans[0] = { ...layout.spans[0], page: 2, text: '543,21' }
  const span = layout.spans[0]
  const check = { status: 'not_evaluated', reasons: [] }
  const review: components['schemas']['StatementReview'] = {
    rows: [],
    header: {
      openingBookDate: null,
      closingBookDate: null,
      openingBookBalance: null,
      closingBookBalance: null,
      availableBalanceObserved: null,
      debitTurnover: { amountMinor: 54321, currency: 'RUB' },
      creditTurnover: null,
      evidence: [{ role: 'debitTurnover', literal: '543,21', spans: [span] }],
    },
    reconciliation: {
      turnoverCheck: { status: 'mismatch', reasons: [] },
      bookBalanceCheck: check,
      rowCoverage: check,
      ledgerBalanceCheck: check,
      financialCoverage: check,
      debitMinor: 65432,
      creditMinor: null,
      blockers: ['printed_turnover_mismatch'],
      warnings: [],
    },
  }
  render(<StatementReconciliation review={review} layout={layout} />)
  expect(screen.getByText(/Выбранный итог: Не выбран или не распознан/)).toBeInTheDocument()
  expect(screen.getByText(/Сумма строк: Не рассчитана/)).toBeInTheDocument()
  expect(screen.getByText('543,21 ₽')).toBeInTheDocument()
  expect(screen.getByText('654,32 ₽')).toBeInTheDocument()
  await userEvent.setup().click(screen.getByRole('button', { name: 'Показать источник списаний' }))
  expect(screen.getByLabelText('Текст страницы 2')).toHaveTextContent('543,21')
})

it('distinguishes legacy classification from exact omissions and paginates every source fragment', async () => {
  const layout = statementFixture().layout!
  layout.spans = Array.from({ length: 25 }, (_, i) => ({
    ...layout.spans[0],
    id: `missing-${i}`,
    text: `fragment-${i}`,
  }))
  const mapping: components['schemas']['StatementDraftMapping'] = {
    version: 'statement-review-mapping.v1',
    revision: 1,
    status: 'reconstructed_partial',
    proposal: null,
    unmappedSpanIds: null,
    unclassifiedSpanIds: layout.spans.map((s) => s.id),
    issues: [],
  }
  const { rerender } = render(<MappingCoverage layout={layout} mapping={mapping} />)
  expect(screen.getByText('Неклассифицированных фрагментов: 25')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Стр. 1: fragment-24' })).not.toBeInTheDocument()
  const user = userEvent.setup()
  await user.click(screen.getByRole('button', { name: 'Следующие фрагменты' }))
  expect(screen.getByRole('button', { name: 'Стр. 1: fragment-24' })).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Стр. 1: fragment-24' }))
  expect(screen.getByLabelText('Текст страницы 1')).toHaveTextContent('fragment-24')
  rerender(
    <MappingCoverage
      layout={layout}
      mapping={{
        ...mapping,
        status: 'complete',
        unmappedSpanIds: ['missing-24'],
        unclassifiedSpanIds: [],
      }}
    />,
  )
  expect(
    screen.queryByText(/Служебная разметка старого черновика не сохранена/),
  ).not.toBeInTheDocument()
  expect(screen.getByText('Непривязанных фрагментов: 1')).toBeInTheDocument()
})
