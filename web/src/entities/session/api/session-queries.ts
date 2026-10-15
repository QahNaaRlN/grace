import { queryOptions } from '@tanstack/react-query'

import { apiClient, setCSRFToken } from '@/shared/api'

/** Loads the current session; a 401 means "signed out" and resolves to null. */
export async function loadSession() {
  const { data, response } = await apiClient.GET('/api/auth/me')
  if (response.status === 401) return null
  if (!data) throw new Error('Проверка доступа временно недоступна')
  setCSRFToken(data.csrfToken)
  return data
}

export const sessionQueries = {
  all: () => ['auth-session'] as const,
  me: () => queryOptions({ queryKey: sessionQueries.all(), queryFn: loadSession, retry: false }),
}
