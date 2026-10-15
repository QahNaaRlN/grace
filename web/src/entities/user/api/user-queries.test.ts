import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery } from '@/test/fetch-query'
import { authMethodQueries, userQueries } from './user-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps the profile and the sign-in methods under separate roots', () => {
  expect(userQueries.profile().queryKey).toEqual(userQueries.all())
  expect(authMethodQueries.list().queryKey).toEqual(authMethodQueries.all())
  expect(userQueries.all()).not.toEqual(authMethodQueries.all())
})

it('reads the sign-in methods and explains when they are unavailable', async () => {
  get.mockResolvedValueOnce({ data: { password: true } })
  await expect(fetchQuery(authMethodQueries.list())).resolves.toEqual({ password: true })
  expect(get).toHaveBeenCalledWith('/api/auth/methods')
  get.mockResolvedValueOnce({ error: {} })
  await expect(fetchQuery(authMethodQueries.list())).rejects.toThrow(
    'Способы входа временно недоступны.',
  )
})
