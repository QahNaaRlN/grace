import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { FinancialHealth } from './FinancialHealth'
import type { FinancialEvent } from '@/shared/api/types'

describe('SPEC-0021 explicit salary data', () => {
  it('retains general income in operations without classifying it as salary', () => {
    const income: FinancialEvent = {
      id: 'income',
      kind: 'income_received',
      source: 'manual',
      merchant: 'Доход от аренды',
      description: 'Аренда',
      occurredAt: '2026-10-06T10:00:00+05:00',
      createdAt: '2026-10-06T10:00:00+05:00',
      postings: [
        {
          id: 'posting',
          accountId: 'account',
          accountName: 'Счёт',
          direction: 'credit',
          amountMinor: 120000,
          currency: 'RUB',
        },
      ],
    }
    render(
      <MemoryRouter>
        <FinancialHealth
          accounts={[]}
          categories={[]}
          events={[income]}
          obligations={[]}
          receipts={[]}
        />
      </MemoryRouter>,
    )
    const salary = screen
      .getByText('Зарплата', { selector: 'summary' })
      .closest('details')!
    expect(
      within(salary).queryByText('Доход от аренды'),
    ).not.toBeInTheDocument()
    expect(within(salary).getByText('Сведения не указаны')).toBeInTheDocument()
    const operations = screen
      .getByText('Операции', { selector: 'summary' })
      .closest('details')!
    expect(
      within(operations).getByText('Доход от аренды', { selector: 'summary' }),
    ).toBeInTheDocument()
  })
})
