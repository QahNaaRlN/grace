import { screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { renderPage, serveLedger } from '@/test/render-page'
import { CategoriesPage } from './CategoriesPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

const category = {
  id: 'c',
  name: 'Продукты',
  kind: 'expense',
  icon: null,
  hidden: false,
  editable: true,
  operationCount: 3,
  createdAt: '2026-10-01T00:00:00Z',
}

beforeEach(() => serveLedger(get, { '/api/v1/categories': { items: [category] } }))

it('shows the categories even with an empty ledger and offers no account', async () => {
  serveLedger(get, {
    '/api/v1/categories': { items: [category] },
    '/api/v1/accounts': { items: [] },
  })
  renderPage(<CategoriesPage />, '/categories')
  expect(screen.getByRole('heading', { name: 'Категории', level: 1 })).toBeVisible()
  expect(await screen.findByText('Продукты')).toBeVisible()
  expect(screen.getByText('Ваша категория · 3 операции')).toBeVisible()
  expect(screen.queryByRole('heading', { name: 'Начните с первого счёта' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /Добавить .*счёт/ })).not.toBeInTheDocument()
})
