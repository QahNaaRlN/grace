import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { vi } from 'vitest'

/**
 * A fresh query client for hook tests that records every invalidated key (SPEC-0048:
 * mutation hooks own their invalidation, and tests check which keys they refresh).
 */
export function queryHarness() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  const invalidated = () => invalidate.mock.calls.map(([filters]) => filters?.queryKey)
  return { client, wrapper, invalidated }
}
