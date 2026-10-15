import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'

import { renderPage, serveLedger } from '@/test/render-page'
import { AnalyticsPage } from './AnalyticsPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

beforeEach(() =>
  serveLedger(get, { '/api/v1/analytics/period-summary': { from: '', to: '', currencies: [] } }),
)

it('shows the full statistics and folds the detailed calculations', async () => {
  renderPage(<AnalyticsPage />, '/analytics')
  expect(screen.getByRole('heading', { name: 'Статистика', level: 1 })).toBeVisible()
  expect(await screen.findByRole('region', { name: 'Статистика' })).toBeVisible()
  expect(screen.getByLabelText('Группировка')).toBeVisible()
  await userEvent.click(
    screen.getByRole('button', { name: 'Подробные расчёты остатков и платежей' }),
  )
  expect(await screen.findByText('Сводка периода')).toBeVisible()
  expect(screen.getByText('Доступно после обязательств')).toBeVisible()
  expect(await screen.findByText('За период нет финансовых данных.')).toBeVisible()
})
