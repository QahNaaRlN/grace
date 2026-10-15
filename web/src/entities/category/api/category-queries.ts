import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

export const categoryQueries = {
  all: () => ['categories'] as const,
  list: () =>
    queryOptions({
      queryKey: categoryQueries.all(),
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/v1/categories')
        if (!data) throw new Error('Не удалось загрузить категории')
        return data.items
      },
    }),
  importOptions: () =>
    queryOptions({
      queryKey: [...categoryQueries.all(), 'import-options'] as const,
      queryFn: async () => (await apiClient.GET('/api/v1/categories')).data,
    }),
}
