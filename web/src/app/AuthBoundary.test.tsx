import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useContext } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { beforeEach, expect, it, vi } from 'vitest'

import { apiClient } from '@/shared/api'
import { AuthContext } from '@/entities/session'
import { setCSRFToken } from '@/shared/api'

import { AuthBoundary } from './AuthBoundary'

vi.mock('@/shared/api/client', () => ({ apiClient: { GET: vi.fn(), POST: vi.fn() } }))
vi.mock('@/shared/api/csrf', { spy: true })
const session = {
  user: { id: 'user', name: 'Alice', role: 'member' },
  csrfToken: 'csrf',
  mode: 'telegram',
}

function Protected() {
  const auth = useContext(AuthContext)
  return (
    <>
      <p>Private finances</p>
      <button onClick={() => void auth?.logout()}>Logout</button>
    </>
  )
}
function mount() {
  const cache = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  cache.setQueryData(['accounts'], ['private-account'])
  render(
    <QueryClientProvider client={cache}>
      <MemoryRouter>
        <Routes>
          <Route
            path="/"
            element={
              <AuthBoundary>
                <Protected />
              </AuthBoundary>
            }
          />
          <Route path="/login" element={<p>Login required</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return cache
}
beforeEach(() => {
  vi.clearAllMocks()
})

it('does not render financial pages without a session and clears cached data', async () => {
  vi.mocked(apiClient.GET).mockResolvedValue({
    response: new Response(null, { status: 401 }),
  } as never)
  const cache = mount()
  expect(await screen.findByText('Login required')).toBeInTheDocument()
  expect(screen.queryByText('Private finances')).not.toBeInTheDocument()
  expect(cache.getQueryData(['accounts'])).toBeUndefined()
})
it('loads CSRF and clears financial cache on logout', async () => {
  vi.mocked(apiClient.GET).mockResolvedValue({ data: session, response: new Response() } as never)
  vi.mocked(apiClient.POST).mockResolvedValue({
    response: new Response(null, { status: 204 }),
  } as never)
  const cache = mount()
  expect(await screen.findByText('Private finances')).toBeInTheDocument()
  expect(setCSRFToken).toHaveBeenLastCalledWith('csrf')
  await userEvent.click(screen.getByText('Logout'))
  expect(await screen.findByText('Login required')).toBeInTheDocument()
  expect(cache.getQueryData(['accounts'])).toBeUndefined()
  expect(setCSRFToken).toHaveBeenLastCalledWith('')
})
it('removes financial content and cache on an API 401 event', async () => {
  vi.mocked(apiClient.GET).mockResolvedValue({ data: session, response: new Response() } as never)
  const cache = mount()
  await screen.findByText('Private finances')
  await act(async () => {
    window.dispatchEvent(new Event('grace:unauthenticated'))
  })
  expect(await screen.findByText('Login required')).toBeInTheDocument()
  expect(cache.getQueryData(['accounts'])).toBeUndefined()
})
it('clears the previous user data when the session identity changes in another tab', async () => {
  vi.mocked(apiClient.GET).mockResolvedValue({ data: session, response: new Response() } as never)
  const cache = mount()
  await screen.findByText('Private finances')
  vi.mocked(apiClient.GET).mockResolvedValue({
    data: { ...session, user: { ...session.user, id: 'another-user' } },
    response: new Response(),
  } as never)
  await act(async () => {
    await cache.refetchQueries({ queryKey: ['auth-session'] })
  })
  expect(await screen.findByText('Login required')).toBeInTheDocument()
  expect(cache.getQueryData(['accounts'])).toBeUndefined()
})
