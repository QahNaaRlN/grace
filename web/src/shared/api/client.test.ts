import { afterEach, beforeEach, expect, it, vi } from 'vitest'

vi.mock('@/shared/config', () => ({ env: { VITE_API_BASE_URL: 'http://grace.test' } }))

const fetchMock = vi.fn<(request: Request) => Promise<Response>>()

beforeEach(() => {
  vi.resetModules()
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})
afterEach(() => vi.unstubAllGlobals())

async function load() {
  const { apiClient } = await import('./client')
  const csrf = await import('./csrf')
  return { apiClient, csrf }
}

it('adds the session CSRF token to mutations and not to reads', async () => {
  const { apiClient, csrf } = await load()
  csrf.setCSRFToken('synthetic-csrf')
  fetchMock.mockResolvedValue(new Response(null, { status: 204 }))

  await apiClient.POST('/api/auth/logout')
  await apiClient.GET('/api/v1/accounts')

  const [mutation, read] = fetchMock.mock.calls.map(([request]) => request)
  expect(mutation.method).toBe('POST')
  expect(mutation.headers.get('X-CSRF-Token')).toBe('synthetic-csrf')
  expect(read.headers.has('X-CSRF-Token')).toBe(false)
})

it('clears the token and signals sign-out on a 401 outside /auth/me', async () => {
  const { apiClient, csrf } = await load()
  csrf.setCSRFToken('synthetic-csrf')
  const signedOut = vi.fn()
  window.addEventListener('grace:unauthenticated', signedOut)
  fetchMock.mockResolvedValue(new Response(null, { status: 401 }))

  await apiClient.GET('/api/v1/accounts')

  expect(csrf.csrfToken).toBe('')
  expect(signedOut).toHaveBeenCalledOnce()
  window.removeEventListener('grace:unauthenticated', signedOut)
})
