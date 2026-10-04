import createClient from 'openapi-fetch'

import { env } from '@/shared/config/env'
import type { paths } from '@/shared/api/generated/schema'

export const apiClient = createClient<paths>({
  baseUrl: env.VITE_API_BASE_URL,
})

