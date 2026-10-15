import { useMutation, useQueryClient } from '@tanstack/react-query'

import { statisticsQueries } from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import type { Category } from '@/shared/api'
import { apiClient } from '@/shared/api'
import type { CategoryIconName } from '@/shared/lib/category-icons'

// A category name and icon show up in the list, in operations and in statistics.
function useRefreshCategories() {
  const queryClient = useQueryClient()
  return () =>
    Promise.all(
      [categoryQueries.all(), financialEventQueries.all(), statisticsQueries.all()].map(
        (queryKey) => queryClient.invalidateQueries({ queryKey }),
      ),
    )
}

/** Creates a personal category (`category` is null) or renames and re-icons an existing one. */
export function useSaveCategory(category: Category | null) {
  const refresh = useRefreshCategories()
  return useMutation({
    mutationFn: async ({ name, icon }: { name: string; icon: CategoryIconName | null }) => {
      const result =
        category === null
          ? await apiClient.POST('/api/v1/categories', {
              body: { name, ...(icon ? { icon } : {}) },
            })
          : await apiClient.PATCH('/api/v1/categories/{categoryId}', {
              params: { path: { categoryId: category.id } },
              body: { name, icon: icon ?? '' },
            })
      if (!result.data) throw new Error(result.error?.message ?? 'Не удалось сохранить категорию.')
    },
    onSuccess: refresh,
  })
}

/**
 * Deletes a personal category and moves its operations to `moveTo`. The count shown to the
 * user is checked against a fresh preview, and Go checks it again inside the deletion.
 */
export function useDeleteCategory(category: Category) {
  const refresh = useRefreshCategories()
  return useMutation({
    mutationFn: async (moveTo: string) => {
      const count = category.operationCount
      const preview = await apiClient.POST('/api/v1/categories/{categoryId}/delete-preview', {
        params: { path: { categoryId: category.id } },
      })
      if (!preview.data) throw new Error(preview.error?.message ?? 'Категория недоступна.')
      if (preview.data.operations !== count)
        throw new Error('Количество операций изменилось. Обновите страницу.')
      const result = await apiClient.DELETE('/api/v1/categories/{categoryId}', {
        params: {
          path: { categoryId: category.id },
          query: { expectedOperations: count, ...(moveTo ? { moveTo } : {}) },
        },
      })
      if (!result.data) throw new Error(result.error?.message ?? 'Не удалось удалить категорию.')
    },
    onSuccess: refresh,
  })
}

/** Hides a shared category from new-operation choices or shows it again. */
export function useToggleCategoryHidden() {
  const refresh = useRefreshCategories()
  return useMutation({
    mutationFn: (category: Category) => {
      const params = { path: { categoryId: category.id } }
      return category.hidden
        ? apiClient.DELETE('/api/v1/categories/{categoryId}/hidden', { params })
        : apiClient.PUT('/api/v1/categories/{categoryId}/hidden', { params })
    },
    onSuccess: async ({ data }) => {
      if (data) await refresh()
    },
  })
}
