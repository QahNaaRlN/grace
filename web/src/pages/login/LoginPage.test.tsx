import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { describe, it, vi, expect, beforeEach } from 'vitest'
import { LoginPage } from './LoginPage'
const mocks = vi.hoisted(() => ({ GET: vi.fn(), POST: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: mocks }))
function mount() {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<p>Signed in</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}
beforeEach(() => {
  vi.clearAllMocks()
  mocks.GET.mockImplementation(async (path: string) =>
    path === '/api/auth/methods'
      ? {
          data: {
            mode: 'email',
            email: true,
            telegram: false,
            emailLinkAvailable: true,
          },
        }
      : { response: new Response(null, { status: 401 }) },
  )
})
describe('email login', () => {
  it('keeps the invitation, verifies a leading-zero code, and does not offer Telegram in email mode', async () => {
    mocks.POST.mockResolvedValueOnce({
      data: { expiresInSeconds: 600, resendAfterSeconds: 60 },
    }).mockResolvedValueOnce({
      data: {
        user: { id: 'owner', name: 'User', role: 'admin' },
        csrfToken: 'csrf',
        mode: 'email',
      },
    })
    const user = userEvent.setup()
    mount()
    await user.type(
      await screen.findByLabelText('Почта'),
      'person@example.test',
    )
    await user.type(
      screen.getByLabelText('Код приглашения (при первом входе)'),
      'invite',
    )
    expect(
      screen.queryByRole('button', { name: 'Войти через Telegram' }),
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Получить код' }))
    await user.type(await screen.findByLabelText('Код из письма'), '001234')
    await user.click(screen.getByRole('button', { name: 'Войти' }))
    expect(await screen.findByText('Signed in')).toBeVisible()
    expect(mocks.POST.mock.calls[0][1].body.invitation).toBe('invite')
    expect(mocks.POST.mock.calls[1][1].body.code).toBe('001234')
  })
  it('keeps the verification page on invalid/expired code', async () => {
    mocks.POST.mockResolvedValueOnce({
      data: { expiresInSeconds: 600, resendAfterSeconds: 60 },
    }).mockResolvedValueOnce({ error: { message: 'Код истёк' } })
    const user = userEvent.setup()
    mount()
    await user.type(
      await screen.findByLabelText('Почта'),
      'person@example.test',
    )
    await user.click(screen.getByRole('button', { name: 'Получить код' }))
    await user.type(await screen.findByLabelText('Код из письма'), '001234')
    await user.click(screen.getByRole('button', { name: 'Войти' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Код истёк')
    expect(screen.queryByText('Signed in')).not.toBeInTheDocument()
  })
  it('opens the app immediately for an active session without sending email', async () => {
    mocks.GET.mockImplementation(async (path: string) =>
      path === '/api/auth/methods'
        ? {
            data: {
              mode: 'email',
              email: true,
              telegram: false,
              emailLinkAvailable: true,
            },
          }
        : {
            data: {
              user: { id: 'owner', name: 'User', role: 'admin' },
              csrfToken: 'csrf',
              mode: 'email',
            },
            response: new Response(),
          },
    )
    mount()
    expect(await screen.findByText('Signed in')).toBeVisible()
    await waitFor(() => expect(mocks.POST).not.toHaveBeenCalled())
  })
})
