import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type ReceiptDraft = components['schemas']['ReceiptDraft']
export type ReceiptDraftFields = components['schemas']['ReceiptDraftFields']
export type ReceiptDraftPreview = components['schemas']['ReceiptDraftPreview']
export type AIMode = components['schemas']['AIMode']

/** Whether recognition is offered, plus the accounts and categories the draft can use. */
export function loadReceiptContext() {
  return Promise.all([
    apiClient.GET('/api/assistant/capabilities'),
    apiClient.GET('/api/v1/accounts'),
    apiClient.GET('/api/v1/categories'),
  ])
}

export async function getReceiptDraft(draftId: string) {
  return (await apiClient.GET('/api/receipt-drafts/{draftId}', { params: { path: { draftId } } }))
    .data
}

/** Grants or revokes consent to send receipt photos to the recognition provider. */
export async function setReceiptConsent(policyHash: string, grant: boolean) {
  const result = grant
    ? await apiClient.POST('/api/assistant/consents', { body: { scope: 'receipt', policyHash } })
    : await apiClient.DELETE('/api/assistant/consents/{scope}', {
        params: { path: { scope: 'receipt' } },
      })
  if (result.error) throw new Error('Согласие не сохранено. Обнови страницу.')
}

export async function uploadReceiptPhoto(file: File) {
  const body = new FormData()
  body.set('photo', file)
  // multipart is sent without a JSON Content-Type; the browser supplies its boundary.
  const result = await apiClient.POST('/api/receipt-drafts', {
    body: { photo: file as unknown as string },
    bodySerializer: () => body,
  })
  if (!result.data)
    throw new Error('Не удалось загрузить фото. Используй JPEG, PNG или WebP до 8 МБ.')
  return result.data
}

/** A paid recognition attempt; the same key retries it without a new paid request. */
export async function extractReceipt(draftId: string, key: string) {
  const result = await apiClient.POST('/api/receipt-drafts/{draftId}/extract', {
    params: { path: { draftId }, header: { 'Idempotency-Key': key } },
  })
  if (!result.data)
    throw new Error(
      'Распознавание не завершено. Проверь согласие и лимит. Повтор той же попытки не создаёт новый платный запрос.',
    )
  return result.data
}

/** Saves the edited fields, then asks what a confirmation would write. */
export async function saveReceiptEdits(draft: ReceiptDraft, fields: ReceiptDraftFields) {
  const edited = await apiClient.PATCH('/api/receipt-drafts/{draftId}', {
    params: { path: { draftId: draft.id } },
    body: { revision: draft.revision, fields },
  })
  if (!edited.data)
    throw new Error('Не удалось сохранить правки. Открой актуальный черновик и повтори.')
  return edited.data
}

export async function previewReceipt(draft: ReceiptDraft) {
  const result = await apiClient.POST('/api/receipt-drafts/{draftId}/previews', {
    params: { path: { draftId: draft.id } },
    body: { revision: draft.revision },
  })
  if (!result.data)
    throw new Error('Предварительный просмотр недоступен. Проверь сумму, дату, счёт и все позиции.')
  return result.data
}

export async function confirmReceipt(draftId: string, preview: ReceiptDraftPreview, key: string) {
  const result = await apiClient.POST('/api/receipt-drafts/{draftId}/confirmations', {
    params: { path: { draftId }, header: { 'Idempotency-Key': key } },
    body: { previewId: preview.id, hash: preview.hash },
  })
  if (!result.data)
    throw new Error(
      'Подтверждение не получено. Повтори с тем же просмотром; при изменении книги нужен новый просмотр.',
    )
  return result.data
}

export async function cancelReceipt(draftId: string) {
  const result = await apiClient.DELETE('/api/receipt-drafts/{draftId}', {
    params: { path: { draftId } },
  })
  if (!result.data) throw new Error('Отмена не завершена.')
  return result.data
}
