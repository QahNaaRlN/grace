import type { Location } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { chunkReloadKey, lazyPage } from './lazyPage'

const target = (): Location => ({
  pathname: '/analytics',
  search: '?month=2026-10',
  hash: '',
  state: null,
  key: 'next',
})

function DummyPage() {
  return null
}

// SPEC-0052 AC5: a page chunk that fails to load opens the target URL once; a second
// failure goes to React Router.
describe('SPEC-0052 lazyPage', () => {
  const assign = vi.fn()

  beforeEach(() => {
    window.sessionStorage.clear()
    vi.stubGlobal('location', { ...window.location, assign })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    assign.mockReset()
  })

  it('returns the page and clears the reload mark after a successful load', async () => {
    window.sessionStorage.setItem(chunkReloadKey, '1')
    const route = await lazyPage(async () => ({ Component: DummyPage }), target)()
    expect(route.Component).toBe(DummyPage)
    expect(window.sessionStorage.getItem(chunkReloadKey)).toBeNull()
    expect(assign).not.toHaveBeenCalled()
  })

  it('opens the target URL once when the chunk fails to load', async () => {
    let settled = false
    void lazyPage(() => Promise.reject(new TypeError('Failed to fetch')), target)().finally(() => {
      settled = true
    })
    await vi.waitFor(() => expect(assign).toHaveBeenCalledWith('/analytics?month=2026-10'))
    expect(window.sessionStorage.getItem(chunkReloadKey)).toBe('1')
    await Promise.resolve()
    expect(settled).toBe(false)
  })

  it('passes the error to React Router when the chunk fails again after the reload', async () => {
    window.sessionStorage.setItem(chunkReloadKey, '1')
    const error = new TypeError('Failed to fetch')
    await expect(lazyPage(() => Promise.reject(error), target)()).rejects.toBe(error)
    expect(assign).not.toHaveBeenCalled()
  })

  it('does not reload without sessionStorage, so it cannot loop', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('denied', 'SecurityError')
    })
    const error = new TypeError('Failed to fetch')
    await expect(lazyPage(() => Promise.reject(error), target)()).rejects.toBe(error)
    expect(assign).not.toHaveBeenCalled()
    vi.restoreAllMocks()
  })
})
