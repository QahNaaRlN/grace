import { apiClient } from '@/shared/api'

/** Starts the Telegram OAuth flow; resolves with the authorization URL to open. */
export async function startTelegramLogin(invitation: string) {
  const { data } = await apiClient.POST('/api/auth/login', {
    body: invitation ? { invitation } : {},
  })
  if (!data) throw new Error('Не удалось начать вход.')
  return data.authorizationUrl
}
