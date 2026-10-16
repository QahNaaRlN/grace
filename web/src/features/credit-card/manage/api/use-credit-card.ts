import { useMutation, useQueryClient } from '@tanstack/react-query'

import { creditCardQueries } from '@/entities/credit-card'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type CardTermsInput = components['schemas']['CreditCardTermsInput']
export type CardSnapshotInput = components['schemas']['CreditCardSnapshotInput']

/**
 * Terms and bank snapshots never change the ledger, so only the card itself is
 * refreshed. The answer already carries the fresh card; the cache takes it as is.
 */
function useCardWrite<T>(
  accountId: string,
  send: (body: T) => Promise<{ data?: components['schemas']['CreditCard']; error?: unknown }>,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: send,
    onSuccess: async ({ data }) => {
      if (!data) return
      queryClient.setQueryData(creditCardQueries.detail(accountId).queryKey, data)
      await queryClient.invalidateQueries({ queryKey: creditCardQueries.all() })
    },
  })
}

/** Creates a terms version, or edits one when `termsId` is given (revision must match). */
export function useSaveCardTerms(accountId: string, idempotencyKey: string) {
  return useCardWrite(accountId, ({ termsId, body }: { termsId?: string; body: CardTermsInput }) =>
    termsId
      ? apiClient.PATCH('/api/v1/accounts/{accountId}/credit-card/terms/{termsId}', {
          params: { path: { accountId, termsId } },
          body,
        })
      : apiClient.POST('/api/v1/accounts/{accountId}/credit-card/terms', {
          params: { path: { accountId }, header: { 'Idempotency-Key': idempotencyKey } },
          body,
        }),
  )
}

export function useRecordCardSnapshot(accountId: string, idempotencyKey: string) {
  return useCardWrite(accountId, (body: CardSnapshotInput) =>
    apiClient.POST('/api/v1/accounts/{accountId}/credit-card/snapshots', {
      params: { path: { accountId }, header: { 'Idempotency-Key': idempotencyKey } },
      body,
    }),
  )
}

export function useCardReminders(accountId: string) {
  return useCardWrite(accountId, (reminders: boolean) =>
    apiClient.PUT('/api/v1/accounts/{accountId}/credit-card/settings', {
      params: { path: { accountId } },
      body: { reminders },
    }),
  )
}

/** Stores the terms PDF; the returned id is attached to a version on save. */
export async function uploadCardDocument(accountId: string, file: File) {
  const body = new FormData()
  body.set('file', file)
  const result = await apiClient.POST('/api/v1/accounts/{accountId}/credit-card/documents', {
    params: { path: { accountId } },
    body: { file: file as unknown as string },
    bodySerializer: () => body,
  })
  if (!result.data) throw new Error('Не удалось сохранить PDF. Нужен файл PDF до 16 МБ.')
  return result.data
}
