import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { userQueries } from '@/entities/user'
import { queryHarness } from '@/test/query-harness'
import { useProfileActions } from './use-profile-actions'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const profile = {
  id: 'u',
  displayName: 'Alice',
  firstName: '',
  lastName: '',
  email: null,
  avatarUrl: null,
}

beforeEach(() => post.mockReset())

it('verifies the address, updates the form first and then refetches the user', async () => {
  post.mockResolvedValue({ response: new Response(null, { status: 204 }) })
  const order: string[] = []
  const { wrapper, client } = queryHarness()
  vi.mocked(client.invalidateQueries).mockImplementation(async (filters) => {
    order.push(`invalidate ${JSON.stringify(filters?.queryKey)}`)
  })
  const { result } = renderHook(() => useProfileActions(profile), { wrapper })
  await result.current.verifyEmail('123456', () => order.push('verified'))
  expect(order).toEqual(['verified', `invalidate ${JSON.stringify(userQueries.all())}`])
})

it('keeps the cache when the code is wrong', async () => {
  post.mockResolvedValue({
    response: new Response(null, { status: 400 }),
    error: { message: 'Код неверный' },
  })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useProfileActions(profile), { wrapper })
  await expect(result.current.verifyEmail('000000', () => {})).rejects.toThrow('Код неверный')
  expect(invalidated()).toEqual([])
})

it('puts the saved names into the profile cache instead of refetching', async () => {
  post.mockResolvedValue({ data: { ...profile, displayName: 'Алиса' } })
  const { wrapper, client, invalidated } = queryHarness()
  const { result } = renderHook(() => useProfileActions(profile), { wrapper })
  await result.current.saveNames({ displayName: 'Алиса', firstName: '', lastName: '' })
  expect(client.getQueryData(userQueries.profile().queryKey)).toMatchObject({
    displayName: 'Алиса',
  })
  expect(invalidated()).toEqual([])
})
