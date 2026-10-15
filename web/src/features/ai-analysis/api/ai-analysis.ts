import { apiClient } from '@/shared/api'

export function getAICapabilities() {
  return apiClient.GET('/api/assistant/capabilities')
}

export async function getConversationMessages(conversationId: string) {
  const result = await apiClient.GET('/api/assistant/conversations/{conversationId}', {
    params: { path: { conversationId } },
  })
  return result.data?.messages
}

/** Grants consent for the policy shown to the user, or revokes it. */
export async function setAnalysisConsent(policyHash: string, grant: boolean) {
  const result = grant
    ? await apiClient.POST('/api/assistant/consents', { body: { scope: 'analysis', policyHash } })
    : await apiClient.DELETE('/api/assistant/consents/{scope}', {
        params: { path: { scope: 'analysis' } },
      })
  if (result.error)
    throw new Error(
      'Не удалось изменить согласие. Обнови страницу и проверь выбранного провайдера.',
    )
}

export async function createConversation() {
  const created = await apiClient.POST('/api/assistant/conversations')
  if (!created.data) throw new Error('Не удалось открыть диалог.')
  return created.data.id
}

/** Asks one question; a retry with the same key opens the saved answer without a paid call. */
export async function askAnalysis(
  conversationId: string,
  key: string,
  body: { question: string; month: string },
) {
  const result = await apiClient.POST('/api/assistant/conversations/{conversationId}/messages', {
    params: { path: { conversationId }, header: { 'Idempotency-Key': key } },
    body,
  })
  if (!result.data)
    throw new Error(
      'Ответ пока не получен. Можно повторить запрос: сохранённый ответ откроется без повторного платного вызова.',
    )
  return result.data
}

export async function deleteConversation(conversationId: string) {
  const result = await apiClient.DELETE('/api/assistant/conversations/{conversationId}', {
    params: { path: { conversationId } },
  })
  if (result.error) throw new Error('Не удалось удалить разговор.')
}
