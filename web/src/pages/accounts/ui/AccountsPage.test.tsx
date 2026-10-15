import { screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { renderPage, serveLedger } from '@/test/render-page'
import { AccountsPage } from './AccountsPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

beforeEach(() => serveLedger(get))

it('lists the accounts and always offers a new account (SPEC-0036)', async () => {
  renderPage(<AccountsPage />, '/accounts')
  expect(screen.getByRole('heading', { name: 'Счета и карты', level: 1 })).toBeVisible()
  expect(await screen.findByRole('button', { name: /Счёт/ })).toBeVisible()
  expect(screen.getByRole('button', { name: 'Добавить счёт' })).toBeVisible()
  expect(screen.queryByRole('button', { name: 'Добавить операцию' })).not.toBeInTheDocument()
})
