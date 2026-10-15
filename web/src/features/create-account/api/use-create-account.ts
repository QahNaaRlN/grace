import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries } from '@/entities/analytics-report'
import { apiClient } from '@/shared/api'
import type { paths } from '@/shared/api/generated/schema'

export type NewAccount =
  paths['/api/v1/accounts']['post']['requestBody']['content']['application/json']

/** Creates an account; after a success accounts and the cash position are refreshed. */
export function useCreateAccount() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: NewAccount) => apiClient.POST('/api/v1/accounts', { body }),
    onSuccess: async ({ data }) => {
      if (!data) return
      await queryClient.invalidateQueries({ queryKey: accountQueries.all() })
      await queryClient.invalidateQueries({ queryKey: cashPositionQueries.all() })
    },
  })
}
