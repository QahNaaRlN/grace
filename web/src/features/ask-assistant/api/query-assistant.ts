import { apiClient } from '@/shared/api/client'
import type { AssistantQuery, AssistantResult } from '@/shared/api/types'

export type SubmitAssistant = (
  query: AssistantQuery,
  signal: AbortSignal,
) => Promise<AssistantResult>

export class AssistantRequestError extends Error {
  readonly kind: 'validation' | 'network' | 'malformed'

  constructor(kind: 'validation' | 'network' | 'malformed') {
    super(kind)
    this.kind = kind
    this.name = 'AssistantRequestError'
  }
}

export const queryAssistant: SubmitAssistant = async (query, signal) => {
  let response
  try {
    response = await apiClient.POST('/api/v1/assistant/query', {
      body: query,
      signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new AssistantRequestError('network')
  }

  if (response.data) return response.data

  if (
    response.response.status === 503 &&
    response.error &&
    'status' in response.error &&
    response.error.status === 'unavailable'
  ) {
    return response.error
  }

  if (response.response.status === 422) {
    throw new AssistantRequestError('validation')
  }

  throw new AssistantRequestError('malformed')
}
