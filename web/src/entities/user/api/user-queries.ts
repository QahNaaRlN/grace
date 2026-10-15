import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

export const userQueries = {
  all: () => ['user-profile'] as const,
  profile: () =>
    queryOptions({
      queryKey: userQueries.all(),
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/auth/profile')
        if (!data) throw new Error('Не удалось загрузить профиль.')
        return data
      },
    }),
}

export const authMethodQueries = {
  all: () => ['auth-methods'] as const,
  list: () =>
    queryOptions({
      queryKey: authMethodQueries.all(),
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/auth/methods')
        if (!data) throw new Error('Способы входа временно недоступны.')
        return data
      },
    }),
}
