import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { TelegramLoginForm } from './TelegramLoginForm'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const assign = vi.fn()
beforeEach(() => {
  post.mockReset()
  assign.mockReset()
  vi.stubGlobal('location', { ...window.location, assign })
})
afterEach(() => vi.unstubAllGlobals())

it('starts the Telegram flow with the invitation and opens the authorization page', async () => {
  post.mockResolvedValue({ data: { authorizationUrl: 'https://oauth.telegram.org/auth' } })
  render(<TelegramLoginForm onError={() => {}} />)
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Код приглашения (при первом входе)'), ' invite ')
  await user.click(screen.getByRole('button', { name: 'Войти через Telegram' }))
  await waitFor(() => expect(assign).toHaveBeenCalledWith('https://oauth.telegram.org/auth'))
  expect(post).toHaveBeenCalledWith('/api/auth/login', { body: { invitation: 'invite' } })
  expect(screen.getByRole('button', { name: 'Войти через Telegram' })).toBeDisabled()
})

it('explains a failed start and lets the user try again', async () => {
  post.mockResolvedValue({ error: {} })
  const onError = vi.fn()
  render(<TelegramLoginForm onError={onError} />)
  await userEvent.click(screen.getByRole('button', { name: 'Войти через Telegram' }))
  await waitFor(() => expect(onError).toHaveBeenLastCalledWith('Не удалось начать вход.'))
  expect(post).toHaveBeenCalledWith('/api/auth/login', { body: {} })
  expect(screen.getByRole('button', { name: 'Войти через Telegram' })).toBeEnabled()
  expect(assign).not.toHaveBeenCalled()
})
