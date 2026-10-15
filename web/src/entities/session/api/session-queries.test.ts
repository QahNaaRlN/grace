import { beforeEach, expect, it, vi } from 'vitest'

import { setCSRFToken } from '@/shared/api'
import { loadSession } from './session-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))
vi.mock('@/shared/api/csrf', { spy: true })

beforeEach(() => {
  get.mockReset()
  vi.mocked(setCSRFToken).mockClear()
})

it('treats 401 as no session and an unknown failure as an error', async () => {
  get.mockResolvedValueOnce({ response: new Response(null, { status: 401 }) })
  await expect(loadSession()).resolves.toBeNull()
  get.mockResolvedValueOnce({ response: new Response(null, { status: 500 }) })
  await expect(loadSession()).rejects.toThrow('Проверка доступа временно недоступна')
  expect(setCSRFToken).not.toHaveBeenCalled()
})

it('hands the CSRF token of a valid session to the API client', async () => {
  get.mockResolvedValueOnce({
    data: { user: { id: 'u' }, csrfToken: 'token' },
    response: new Response(null, { status: 200 }),
  })
  await expect(loadSession()).resolves.toMatchObject({ user: { id: 'u' } })
  expect(setCSRFToken).toHaveBeenCalledWith('token')
})
