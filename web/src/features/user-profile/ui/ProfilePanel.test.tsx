import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProfilePanel } from './ProfilePanel'
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
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <ProfilePanel />
      </QueryClientProvider>,
    )
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Имя'), 'Name')
    await user.type(screen.getByLabelText('Фамилия'), 'Family')
    await user.click(screen.getByRole('button', { name: 'Сохранить профиль' }))
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Профиль сохранён',
    )
    expect(mocks.POST.mock.calls[0][1].body).toEqual({
      displayName: 'Name Family',
      firstName: 'Name',
      lastName: 'Family',
    })
    expect(mocks.POST.mock.calls[0][1].body).not.toHaveProperty('id')
    expect(mocks.POST.mock.calls[0][1].body).not.toHaveProperty('role')
  })
})
