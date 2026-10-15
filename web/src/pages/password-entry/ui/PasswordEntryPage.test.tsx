import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, expect, it, vi } from 'vitest'

import { renderPage } from '@/test/render-page'
import { PasswordEntryPage } from './PasswordEntryPage'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

beforeEach(() => get.mockReset())

it('titles each view and shows the password form when passwords are enabled', async () => {
  get.mockResolvedValue({ data: { password: true } })
  renderPage(<PasswordEntryPage view="forgot" />, '/forgot-password')
  expect(screen.getByRole('heading', { name: 'Восстановление пароля' })).toBeVisible()
  expect(await screen.findByLabelText('Почта')).toBeVisible()
})

it('explains that the method is off and lets the user retry or go back', async () => {
  get.mockResolvedValue({ data: { password: false } })
  renderPage(<PasswordEntryPage view="register" />, '/register')
  expect(screen.getByRole('heading', { name: 'Регистрация в Grace' })).toBeVisible()
  expect(await screen.findByRole('alert')).toHaveTextContent('Этот способ доступа пока недоступен.')
  expect(screen.getByRole('link', { name: 'Вернуться ко входу' })).toHaveAttribute('href', '/login')
  get.mockClear()
  await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))
  expect(get).toHaveBeenCalledWith('/api/auth/methods')
})
