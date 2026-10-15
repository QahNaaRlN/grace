import createClient from 'openapi-fetch'

import { env } from '@/shared/config'
import type { paths } from './generated/schema'
import { csrfToken, setCSRFToken } from './csrf'

// The middleware below sets X-CSRF-Token on every mutation, so the client type drops
// the header the OpenAPI contract declares for those operations.
type WithoutCsrfHeader<Operation> = Operation extends {
  parameters: { header: infer Header extends { 'X-CSRF-Token': unknown } }
}
  ? Omit<Operation, 'parameters'> & {
      parameters: Omit<Operation['parameters'], 'header'> &
        (keyof Omit<Header, 'X-CSRF-Token'> extends never
          ? { header?: never }
          : { header: Omit<Header, 'X-CSRF-Token'> })
    }
  : Operation
export type ClientPaths = {
  [Path in keyof paths]: { [Method in keyof paths[Path]]: WithoutCsrfHeader<paths[Path][Method]> }
}

export const apiClient = createClient<ClientPaths>({
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
