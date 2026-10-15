import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import {
  importBatchQueries,
  importErrorText,
  nextUnresolvedRecord,
  useImportRefresh,
} from '@/entities/import-batch'
import {
  PreviewPanel,
  previewImport,
  useConfirmImport,
  type ImportPreview,
  type ImportReceipt as Receipt,
} from '@/features/confirm-import'
import { ReviewForm } from '@/features/review-import-record'
import { NewDraftCard } from '@/features/upload-import'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { ImportHistory } from './ImportHistory'
import { ImportReceipt } from './ImportReceipt'
import { MonthPanel } from './MonthPanel'
import { RecordReview, type Selection, type View } from './RecordReview'

const initialView: View = {
  status: '',
  domain: 'transaction',
  q: '',
  sort: 'row',
  cursor: '',
  limit: 100,
}

export function ImportReview() {
  const refreshImports = useImportRefresh()
  const [params, setParams] = useSearchParams()
  const batchId = params.get('batch') ?? ''
  const [view, setView] = useState<View>(initialView)
  const [selection, setSelection] = useState<Selection>({ mode: 'ids', ids: [] })
  const [recordId, setRecordId] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [preview, setPreview] = useState<ImportPreview>()
  const [receipt, setReceipt] = useState<Receipt>()
  const [confirmationKey, setConfirmationKey] = useState('')
  const batch = useQuery({ ...importBatchQueries.batch(batchId), enabled: !!batchId })
  const detail = useQuery({
    ...importBatchQueries.record(batchId, recordId),
    enabled: !!batchId && !!recordId,
  })
  const receiptId = params.get('confirmation') ?? ''
  const savedReceipt = useQuery({
    ...importBatchQueries.receipt(batchId, receiptId),
    enabled: !!batchId && !!receiptId,
  })
  const shownReceipt = receipt || savedReceipt.data
  const confirmation = useConfirmImport(batchId, {
    onConfirmed: (data) => {
      const nextParams = new URLSearchParams(params)
      nextParams.set('batch', batchId)
      nextParams.set('confirmation', data.id)
      setReceipt(data)
      setParams(nextParams)
      setPreview(undefined)
      setSelection({ mode: 'ids', ids: [] })
      setNotice('Решения подтверждены. Оставшиеся строки можно проверить позже.')
    },
    onStale: () => setPreview(undefined),
  })
  const selectedCount = selection.mode === 'filter' ? selection.total : selection.ids.length
  function openBatch(id: string) {
    setParams({ batch: id })
    setView(initialView)
    setSelection({ mode: 'ids', ids: [] })
    setRecordId('')
    setPreview(undefined)
    setReceipt(undefined)
    setError('')
  }
  function changeView(next: Partial<View>) {
    setView((current) => ({ ...current, ...next }))
    setPreview(undefined)
  }
  function changeSelection(next: Selection) {
    setSelection(next)
    setPreview(undefined)
  }
  async function action(run: () => Promise<void>) {
    if (pending) return
    setPending(true)
    setError('')
    try {
      await run()
    } catch (err) {
      setError(importErrorText(err))
    } finally {
      setPending(false)
    }
  }
  // SPEC-0038: the new-draft card owns its own states and messages.
  function draftSaved(id: string) {
    openBatch(id)
    setNotice('')
  }
  // SPEC-0044: walk unresolved rows in sheet order without scrolling.
  async function openNextUnresolved(after?: string) {
    const record = await nextUnresolvedRecord(batchId, after)
    if (record) {
      setRecordId(record.id)
      return true
    }
    setNotice('Нерешённых строк больше нет. Можно подготовить предварительный просмотр.')
    return false
  }
  async function makePreview() {
    if (!batch.data) return
    const filter = { domain: view.domain, status: view.status || undefined, q: view.q || undefined }
    const body =
      selection.mode === 'filter'
        ? { expectedRevision: batch.data.revision, filter }
        : { expectedRevision: batch.data.revision, recordIds: selection.ids }
    const data = await previewImport(batchId, body)
    setPreview(data)
    setReceipt(undefined)
    setConfirmationKey(crypto.randomUUID())
  }
  async function confirm() {
    if (!preview) return
    const { data, stale } = await confirmation.mutateAsync({ preview, key: confirmationKey })
    if (stale)
      throw new Error('Предварительный просмотр устарел. Проверьте строки и получите новый.')
    if (!data)
      throw new Error(
        'Подтверждение не завершено. Можно повторить действие: повторная запись защищена.',
      )
  }
  // Rendered inside the page column, so the sections stay its direct children.
  return (
    <>
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      {notice && <p role="status">{notice}</p>}
      <NewDraftCard onSaved={draftSaved} />
      <ImportHistory selectedId={batchId} onOpen={openBatch} />
      {batchId && batch.isPending && <LoadingSkeleton label="Загружаем черновик…" />}
      {(batch.isError || detail.isError) && (
        <p role="alert">
          Не удалось загрузить данные.{' '}
          <Button variant="outline" onClick={() => void refreshImports.all()}>
            Повторить
          </Button>
        </p>
      )}
      {batch.data && (
        <>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">
              {batch.data.fileName ?? batch.data.source.sourceId}
            </h2>
            <Button
              variant="outline"
              disabled={pending}
              onClick={() =>
                void action(async () => {
                  await openNextUnresolved(recordId)
                })
              }
            >
              Следующая нерешённая
            </Button>
          </div>
          <MonthPanel batchId={batchId} onOpenRecord={setRecordId} />
          <RecordReview
            batchId={batchId}
            batch={batch.data}
            view={view}
            onView={changeView}
            selection={selection}
            onSelection={changeSelection}
            onOpen={setRecordId}
            disabled={pending}
          />
          <Button
            disabled={
              pending || selectedCount === 0 || (selection.mode === 'ids' && selectedCount > 100)
            }
            onClick={() => void action(makePreview)}
          >
            Предварительный просмотр ({selectedCount})
          </Button>
          {preview && (
            <PreviewPanel
              preview={preview}
              pending={pending}
              onConfirm={() => void action(confirm)}
            />
          )}
          {shownReceipt && <ImportReceipt batchId={batchId} receipt={shownReceipt} />}
          {recordId && (
            <Dialog
              open
              onOpenChange={(open) => {
                if (!open) setRecordId('')
              }}
            >
              <DialogContent
                data-motion="menu"
                className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(560px,calc(100%-1rem))] translate-x-0 translate-y-0 overflow-y-auto rounded-r-none sm:max-w-[560px]"
              >
                <DialogHeader>
                  <DialogTitle>
                    {detail.data
                      ? `${detail.data.source.sheet}, строка ${detail.data.source.row}`
                      : 'Строка'}
                  </DialogTitle>
                  <DialogDescription>
                    Решение сохраняется в черновике; финансовый журнал меняется только после
                    предварительного просмотра и подтверждения.
                  </DialogDescription>
                </DialogHeader>
                {detail.isPending ? (
                  <LoadingSkeleton label="Загружаем строку…" />
                ) : (
                  detail.data && (
                    <ReviewForm
                      key={`${detail.data.id}:${JSON.stringify(detail.data.decision)}`}
                      record={detail.data}
                      batchId={batchId}
                      revision={batch.data.revision}
                      onSaved={async () => {
                        const saved = recordId
                        setPreview(undefined)
                        setNotice('Решение сохранено. Финансовый журнал ещё не изменён.')
                        // After a decision, move straight to the next unresolved row.
                        try {
                          await openNextUnresolved(saved)
                        } catch {
                          /* the saved row stays open */
                        }
                      }}
                    />
                  )
                )}
              </DialogContent>
            </Dialog>
          )}
        </>
      )}
    </>
  )
}
