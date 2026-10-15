import { screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { renderPage, serveLedger } from '@/test/render-page'
import { HealthPage } from './HealthPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

beforeEach(() => serveLedger(get))

it('shows the financial health of the ledger without create actions', async () => {
  renderPage(<HealthPage />, '/health')
  expect(screen.getByRole('heading', { name: 'Финансовое здоровье', level: 1 })).toBeVisible()
  expect(
    screen.getByText('Ближайшие обязательства, доступные деньги и качество данных.'),
  ).toBeVisible()
  expect(await screen.findByText('Полнота истории не подтверждена')).toBeVisible()
  expect(screen.queryByRole('button', { name: /Добавить/ })).not.toBeInTheDocument()
})
