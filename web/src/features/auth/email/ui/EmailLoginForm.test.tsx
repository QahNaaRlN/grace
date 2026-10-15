import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { sessionQueries } from '@/entities/session'
import { EmailLoginForm } from './EmailLoginForm'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

function mount(onError = vi.fn()) {
  const cache = new QueryClient()
  cache.setQueryData(['accounts'], ['previous user'])
  render(
    <QueryClientProvider client={cache}>
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<EmailLoginForm onError={onError} />} />
          <Route path="/" element={<p>Обзор</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return { cache, onError }
}

beforeEach(() => post.mockReset())

describe('email sign-in', () => {
  it('sends a code, then signs in with it and starts from an empty cache', async () => {
    post
      .mockResolvedValueOnce({ data: { resendAfterSeconds: 60 } })
      .mockResolvedValueOnce({ data: { user: { id: 'u' }, csrfToken: 'token' } })
    const { cache } = mount()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Почта'), ' me@example.com ')
    await user.click(screen.getByRole('button', { name: 'Получить код' }))
    expect(post.mock.calls[0]).toEqual([
      '/api/auth/email/request',
      { body: { email: 'me@example.com' } },
    ])
    expect(await screen.findByText(/Код отправлен на/)).toBeVisible()
    expect(screen.getByRole('button', { name: 'Повторная отправка через 60 с' })).toBeDisabled()
    await user.type(screen.getByLabelText('Код из письма'), '12a3456')
    await user.click(screen.getByRole('button', { name: 'Войти' }))
    expect(await screen.findByText('Обзор')).toBeVisible()
    expect(post.mock.calls[1][1]).toEqual({ body: { code: '123456' } })
    expect(cache.getQueryData(['accounts'])).toBeUndefined()
    expect(cache.getQueryData(sessionQueries.me().queryKey)).toMatchObject({ user: { id: 'u' } })
  })

  it('reports a refused request through onError and stays on the email step', async () => {
    post.mockResolvedValueOnce({ error: { message: 'Слишком много попыток' } })
    const { onError } = mount()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Почта'), 'me@example.com')
    await user.click(screen.getByRole('button', { name: 'Получить код' }))
    await waitFor(() => expect(onError).toHaveBeenLastCalledWith('Слишком много попыток'))
    expect(screen.getByLabelText('Почта')).toBeVisible()
  })
})
