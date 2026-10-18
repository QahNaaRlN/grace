import { useMutation, useQueryClient } from '@tanstack/react-query'

import { bankStatementQueries } from '@/entities/bank-statement'
import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

type Edit = components['schemas']['StatementEdit']
type Manual = components['schemas']['StatementManualDraft']
export function useStatementMutations(statementId: string) {
  const cache = useQueryClient()
  const refresh = async () => {
    await Promise.all([
      cache.invalidateQueries({ queryKey: bankStatementQueries.all() }),
      cache.invalidateQueries({ queryKey: importBatchQueries.all() }),
    ])
  }
  const conflict = async (error: { code: string; message: string } | undefined) => {
    if (error?.code === 'revision_conflict') {
      await cache.fetchQuery(bankStatementQueries.detail(statementId))
      throw new Error(
        'Источник изменён другим действием. Ввод сохранён; проверьте актуальное состояние и повторите сохранение.',
      )
    }
  }
  const upload = useMutation({
    mutationFn: async ({
      file,
      accountId,
      key,
    }: {
      file: File
      accountId: string
      key: string
    }) => {
      const form = new FormData()
      form.append('file', file)
      form.append('accountId', accountId)
      const { data, error } = await apiClient.POST('/api/imports/statements/pdf', {
        body: { file: '', accountId },
        bodySerializer: () => form,
        params: { header: { 'Idempotency-Key': key } },
      })
      if (!data) throw new Error(error?.message ?? 'Не удалось сохранить PDF.')
      return data.statement
    },
    onSuccess: refresh,
  })
  const edit = useMutation({
    mutationFn: async (body: Edit) => {
      const { data, error } = await apiClient.PATCH('/api/imports/statements/{statementId}', {
        params: { path: { statementId }, header: { 'Idempotency-Key': crypto.randomUUID() } },
        body,
      })
      if (!data) {
        await conflict(error)
        throw new Error(
          error?.message ?? 'Проверка не сохранена. Обновите состояние, ввод сохранён в форме.',
        )
      }
      return data
    },
    onSuccess: refresh,
  })
  const manual = useMutation({
    mutationFn: async ({ body, key }: { body: Manual; key: string }) => {
      const { data, error } = await apiClient.POST(
        '/api/imports/statements/{statementId}/manual-drafts',
        {
          params: { path: { statementId }, header: { 'Idempotency-Key': key } },
          body,
        },
      )
      if (!data) {
        await conflict(error)
        throw new Error(error?.message ?? 'Черновик не сохранён. Ввод остался в форме.')
      }
      return data
    },
    onSuccess: refresh,
  })
  const remove = useMutation({
    mutationFn: async (expectedRevision: number) => {
      const { data, error } = await apiClient.DELETE(
        '/api/imports/statements/{statementId}/original',
        {
          params: { path: { statementId }, header: { 'Idempotency-Key': crypto.randomUUID() } },
          body: { expectedRevision },
        },
      )
      if (!data) throw new Error(error?.message ?? 'Оригинал не удалён.')
      return data
    },
    onSuccess: refresh,
  })
  return { upload, edit, manual, remove }
}
