import createClient from 'openapi-fetch'

import { env } from '@/shared/config/env'
import type { paths } from '@/shared/api/generated/schema'
import { csrfToken, setCSRFToken } from '@/shared/model/auth-session'

export const apiClient = createClient<paths>({
  baseUrl: env.VITE_API_BASE_URL,
  credentials: 'include',
})

apiClient.use({
  onRequest({ request }) {
    if (!['GET', 'HEAD', 'OPTIONS'].includes(request.method) && csrfToken) {
      request.headers.set('X-CSRF-Token', csrfToken)
    }
    return request
  },
  onResponse({ request, response }) {
    if (response.status === 401 && !new URL(request.url).pathname.endsWith('/auth/me')) {
      setCSRFToken('')
      window.dispatchEvent(new Event('grace:unauthenticated'))
    }
    return response
  },
})
