import { useQueryClient } from '@tanstack/react-query'

import { sessionQueries } from '@/entities/session'
import { apiClient, setCSRFToken } from '@/shared/api'

/** Sends a one-time code; resolves with the cooldown before the next request. */
export async function requestEmailCode(email: string, invitation: string) {
  const { data, error } = await apiClient.POST('/api/auth/email/request', {
    body: { email, ...(invitation ? { invitation } : {}) },
  })
  if (!data) throw new Error(error?.message ?? 'Не удалось отправить код.')
  return data.resendAfterSeconds
}

/**
 * Verifies the code and starts a fresh session: nothing cached for a previous user
 * survives, and the new session is put into the cache instead of being refetched.
 */
export function useVerifyEmailCode() {
  const cache = useQueryClient()
  return async (code: string) => {
    const { data, error } = await apiClient.POST('/api/auth/email/verify', { body: { code } })
    if (!data) throw new Error(error?.message ?? 'Не удалось подтвердить код.')
    cache.clear()
    setCSRFToken(data.csrfToken)
    cache.setQueryData(sessionQueries.me().queryKey, data)
  }
}
