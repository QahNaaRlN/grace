import { screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { renderPage, serveLedger } from '@/test/render-page'
import { OperationsPage } from './OperationsPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

beforeEach(() => serveLedger(get))

it('lists operations and offers a new operation but no account', async () => {
  renderPage(<OperationsPage />, '/operations')
  expect(screen.getByRole('heading', { name: 'Операции', level: 1 })).toBeVisible()
  expect(await screen.findByRole('button', { name: 'Добавить операцию' })).toBeVisible()
  expect(screen.queryByRole('button', { name: /Добавить .*счёт/ })).not.toBeInTheDocument()
  expect(await screen.findByPlaceholderText('Магазин или описание')).toBeVisible()
})
