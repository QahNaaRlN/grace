import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { importBatchQueries } from '@/entities/import-batch'
import { queryHarness } from '@/test/query-harness'
import { useCreateImportDraft } from './create-import-draft'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const file = new File(['{}'], 'snapshot.json', { type: 'application/json' })

beforeEach(() => post.mockReset())

it('saves the original bytes with the file key and reloads the drafts', async () => {
  post.mockResolvedValue({ data: { id: 'batch' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateImportDraft(), { wrapper })
  const id = await result.current.mutateAsync({
    file,
    kind: 'json',
    sourceId: '',
    idempotencyKey: 'key',
  })
  expect(id).toBe('batch')
  expect(post.mock.calls[0][1].bodySerializer()).toBe(file)
  expect(post.mock.calls[0][1].params.header).toEqual({ 'Idempotency-Key': 'key' })
  expect(invalidated()).toEqual([importBatchQueries.all()])
})

it('explains an unreadable CSV and keeps the cache', async () => {
  post.mockResolvedValue({ error: { code: 'invalid_csv' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateImportDraft(), { wrapper })
  await expect(
    result.current.mutateAsync({ file, kind: 'csv', sourceId: 'Лист', idempotencyKey: 'key' }),
  ).rejects.toThrow('Не удалось прочитать CSV')
  expect(invalidated()).toEqual([])
})
