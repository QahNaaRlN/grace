import { useQueryClient } from '@tanstack/react-query'

import { sessionQueries } from '@/entities/session'
import { userQueries } from '@/entities/user'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

type Profile = components['schemas']['UserProfile']
type Names = Pick<Profile, 'displayName' | 'firstName' | 'lastName'>

/**
 * Profile writes. Each action throws an Error with the text to show and owns its cache
 * effects: the POST response updates the session in place instead of refetching /api/auth/me.
 */
export function useProfileActions(profile: Profile) {
  const cache = useQueryClient()
  function update(data: Profile) {
    cache.setQueryData(userQueries.profile().queryKey, data)
    cache.setQueryData(sessionQueries.me().queryKey, (current) =>
      current
        ? {
            ...current,
            user: { ...current.user, name: data.displayName, avatarUrl: data.avatarUrl },
          }
        : current,
    )
  }
  return {
    async saveNames(body: Names) {
      const { data, error } = await apiClient.POST('/api/auth/profile', { body })
      if (!data) throw new Error(error?.message ?? 'Профиль не сохранён.')
      update(data)
      return data
    },
    async uploadAvatar(file: File) {
      if (file.size > 2 * 1024 * 1024) throw new Error('Изображение должно быть не больше 2 МБ.')
      const form = new FormData()
      form.append('avatar', file)
      const { data, error } = await apiClient.POST('/api/auth/profile/avatar', {
        body: { avatar: '' },
        bodySerializer: () => form,
      })
      if (!data) throw new Error(error?.message ?? 'Аватар не сохранён.')
      update(data)
    },
    async removeAvatar() {
      const { response, error } = await apiClient.DELETE('/api/auth/profile/avatar')
      if (!response.ok) throw new Error(error?.message ?? 'Не удалось удалить аватар.')
      update({ ...profile, avatarUrl: null })
    },
    async requestEmail(email: string) {
      const { data, error } = await apiClient.POST('/api/auth/profile/email/request', {
        body: { email },
      })
      if (!data) throw new Error(error?.message ?? 'Не удалось отправить код.')
    },
    /** Verifies the new address; the server ends other sessions, so the user is refetched. */
    async verifyEmail(code: string, onVerified: () => void) {
      const { response, error } = await apiClient.POST('/api/auth/profile/email/verify', {
        body: { code },
      })
      if (!response.ok) throw new Error(error?.message ?? 'Не удалось подтвердить почту.')
      onVerified()
      await cache.invalidateQueries({ queryKey: userQueries.all() })
    },
  }
}
