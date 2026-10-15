import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProfilePanel } from './ProfilePanel'
import { sessionQueries } from '@/entities/session'
const mocks = vi.hoisted(() => ({
  GET: vi.fn(),
  POST: vi.fn(),
  DELETE: vi.fn(),
}))
vi.mock('@/shared/api/client', () => ({ apiClient: mocks }))
const profile = {
  id: 'user',
  displayName: 'Alias',
  firstName: '',
  lastName: '',
  email: 'person@example.test',
  avatarUrl: null,
}
beforeEach(() => {
  vi.clearAllMocks()
  mocks.GET.mockResolvedValue({ data: profile })
})
describe('personal profile', () => {
  it('saves explicit first and last names without changing the user identity', async () => {
    mocks.POST.mockResolvedValue({
      data: {
        ...profile,
        displayName: 'Name Family',
        firstName: 'Name',
        lastName: 'Family',
      },
    })
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <ProfilePanel />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Имя'), 'Name')
    await user.type(screen.getByLabelText('Фамилия'), 'Family')
    await user.click(screen.getByRole('button', { name: 'Сохранить профиль' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Профиль сохранён')
    expect(mocks.POST.mock.calls[0][1].body).toEqual({
      displayName: 'Name Family',
      firstName: 'Name',
      lastName: 'Family',
    })
    expect(mocks.POST.mock.calls[0][1].body).not.toHaveProperty('id')
    expect(mocks.POST.mock.calls[0][1].body).not.toHaveProperty('role')
  })

  it('SPEC-0036: confirms inside the personal card, saves once and updates the session without /me', async () => {
    const saved = { ...profile, displayName: 'Number One' }
    mocks.POST.mockResolvedValue({ data: saved })
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    client.setQueryData(sessionQueries.all(), {
      user: { id: 'user', name: 'Alias', role: 'owner', avatarUrl: null },
      csrfToken: 't',
      mode: 'email',
    })
    render(
      <QueryClientProvider client={client}>
        <ProfilePanel />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    const save = await screen.findByRole('button', { name: 'Сохранить профиль' })
    expect(save).toBeDisabled()
    const display = screen.getByLabelText('Отображаемое имя')
    await user.clear(display)
    await user.type(display, 'Number One')
    await user.dblClick(save)
    const status = await screen.findByRole('status')
    expect(status).toHaveTextContent('Профиль сохранён')
    expect(status.closest('section')).toHaveTextContent('Личные данные')
    expect(mocks.POST).toHaveBeenCalledOnce()
    expect(mocks.GET.mock.calls.map((call) => call[0])).not.toContain('/api/auth/me')
    expect(client.getQueryData<{ user: { name: string } }>(sessionQueries.all())?.user.name).toBe(
      'Number One',
    )
    expect(save).toBeDisabled()
  })

  it('SPEC-0036: shows e-mail errors in the e-mail card', async () => {
    mocks.POST.mockResolvedValue({ error: { message: 'Слишком часто' } })
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <ProfilePanel />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Новая почта'), 'new@example.test')
    await user.click(screen.getByRole('button', { name: 'Получить код' }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Слишком часто')
    expect(alert.closest('section')).toHaveTextContent('Почта и вход')
  })
})
