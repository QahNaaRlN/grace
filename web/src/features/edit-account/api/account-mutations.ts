import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries, type PaymentCard } from '@/entities/account'
import { apiClient } from '@/shared/api'
import type { Account } from '@/shared/api'

type CardFields = { name: string; bank: string; last4: string }
type AccountChange = { revision: number; name: string; type?: Account['type'] }

function useRefreshAccounts() {
  const queryClient = useQueryClient()
  return async ({ data }: { data?: unknown }) => {
    if (data) await queryClient.invalidateQueries({ queryKey: accountQueries.all() })
  }
}

/** Renames an account or changes its type (only while it has no operations). */
export function useUpdateAccount(accountId: string) {
  const onSuccess = useRefreshAccounts()
  return useMutation({
    mutationFn: (body: AccountChange) =>
      apiClient.PATCH('/api/v1/accounts/{accountId}', {
        params: { path: { accountId } },
        body,
      }),
    onSuccess,
  })
}

/** Adds a card to an account or changes an existing one. */
export function useSaveCard(accountId: string, card: PaymentCard | null) {
  const onSuccess = useRefreshAccounts()
  return useMutation({
    mutationFn: (body: CardFields) =>
      card
        ? apiClient.PATCH('/api/v1/accounts/{accountId}/payment-instruments/{cardId}', {
            params: { path: { accountId, cardId: card.id } },
            body,
          })
        : apiClient.POST('/api/v1/accounts/{accountId}/payment-instruments', {
            params: { path: { accountId } },
            body,
          }),
    onSuccess,
  })
}

/** Disables an active card or enables a disabled one. */
export function useToggleCard(accountId: string) {
  const onSuccess = useRefreshAccounts()
  return useMutation({
    mutationFn: (card: PaymentCard) =>
      apiClient.PATCH('/api/v1/accounts/{accountId}/payment-instruments/{cardId}', {
        params: { path: { accountId, cardId: card.id } },
        body: { disabled: !card.disabledAt },
      }),
    onSuccess,
  })
}
