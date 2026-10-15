import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { expect, it } from 'vitest'
import { FinancialHealth } from './FinancialHealth'
import type { Account, CashPosition } from '@/shared/api'
it('keeps unknown balances and incomplete history explicit without inventing salary or a score', () => {
  render(
    <MemoryRouter>
      <FinancialHealth
        accounts={[
          {
            id: 'a',
            name: 'Кредитная карта',
            type: 'credit',
            currency: 'RUB',
            balanceMinor: null,
            balanceReconciled: false,
            createdAt: '',
          } as Account,
        ]}
        categories={[]}
        events={[]}
        obligations={[]}
        receipts={[]}
        cash={
          {
            through: '2026-10-31',
            currencies: [
              {
                currency: 'RUB',
                availableAfterObligationsMinor: null,
                liquidBalanceMinor: null,
                obligationsThroughMinor: 50000,
              },
            ],
          } as CashPosition
        }
      />
    </MemoryRouter>,
  )
  expect(screen.getAllByText('Не сверён').length).toBeGreaterThan(0)
  expect(screen.getByText(/Полнота истории не подтверждена/)).toBeInTheDocument()
  expect(screen.getByText(/не считается зарплатой автоматически/)).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Проверить график' })).toHaveAttribute(
    'href',
    '/payments',
  )
})
