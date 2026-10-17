import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Copy, Download, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { loanQueries } from '@/entities/obligation'
import { apiClient, type FinancialEvent } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Disclosure } from '@/shared/ui/disclosure'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/shared/ui/dialog'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import {
  useApplyLoanSchedule,
  useCopyLoanSchedule,
  useUploadLoanDocument,
} from '../api/loan-mutations'
import { LoanDebt } from './LoanDebt'
import { LoanForm, ScheduleEditor, type Calculation, type Draft } from './LoanForms'

export function LoanPanel({
  obligationId,
  currency = 'RUB',
  events = [],
  usedEventIDs = new Set(),
}: {
  obligationId: string
  currency?: string
  events?: FinancialEvent[]
  usedEventIDs?: Set<string>
}) {
  const [draft, setDraft] = useState<Draft | null>(null)
  const [calculation, setCalculation] = useState<Calculation | null>(null)
  const [confirm, setConfirm] = useState(false)
  const details = useQuery(loanQueries.details(obligationId))
  const copy = useCopyLoanSchedule(obligationId, {
    onSuccess: setDraft,
    onError: (e) => toast.error(e.message),
  })
  const upload = useUploadLoanDocument(obligationId, {
    onSuccess: () => toast.success('Оригинал документа сохранён'),
    onError: (e) => toast.error(e.message),
  })
  const apply = useApplyLoanSchedule(obligationId, {
    onSuccess: () => {
      setConfirm(false)
      setDraft(null)
      toast.success('Новый график применён')
    },
    onError: (e) => toast.error(e.message),
  })
  async function download(document: components['schemas']['LoanDocument']) {
    try {
      const { data, error } = await apiClient.GET(
        '/api/v1/loans/{obligationId}/documents/{documentId}',
        { params: { path: { obligationId, documentId: document.id } }, parseAs: 'blob' },
      )
      if (error) throw new Error(error.message)
      if (!data) return
      const url = URL.createObjectURL(data)
      const a = window.document.createElement('a')
      a.href = url
      a.download = document.filename
      a.click()
      setTimeout(() => URL.revokeObjectURL(url), 10000)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Не удалось скачать документ')
    }
  }
  const prepaid = new Set(details.data?.debt?.prepayments.map((p) => p.financialEventId))
  // Any unlinked debt payment in the loan currency may be an early repayment.
  const candidates = events.filter(
    (event) =>
      event.kind === 'debt_payment' &&
      !event.voidedAt &&
      !usedEventIDs.has(event.id) &&
      !prepaid.has(event.id) &&
      event.postings.length === 1 &&
      event.postings[0]?.currency === currency &&
      event.postings[0]?.direction === 'debit',
  )
  return (
    <>
      {details.data?.debt && (
        <LoanDebt
          obligationId={obligationId}
          currency={currency}
          debt={details.data.debt}
          candidates={candidates}
        />
      )}
      <Disclosure
        title="Договор, документы и копии графика"
        className="rounded-xl border p-4"
        triggerClassName="min-h-11 font-medium"
      >
        <div className="mt-4 space-y-6">
          {details.isPending ? (
            <LoadingSkeleton label="Загружаем кредит…" />
          ) : details.isError ? (
            <div className="space-y-3">
              <p role="alert">{details.error.message}</p>
              <Button variant="outline" onClick={() => void details.refetch()}>
                Повторить
              </Button>
            </div>
          ) : (
            details.data && (
              <LoanForm
                key={details.data.revision}
                initial={details.data}
                obligationId={obligationId}
              />
            )
          )}
          <section className="space-y-3">
            <h4 className="font-medium">Документы договора</h4>
            <p className="text-sm text-muted-foreground">
              PDF до 16 МиБ. Каждая загрузка сохраняется новой версией. Оригиналы доступны только
              вам; общий лимит — 100 МиБ.
            </p>
            <Label htmlFor={`loan-file-${obligationId}`} className="sr-only">
              Загрузить документ договора PDF
            </Label>
            <Input
              id={`loan-file-${obligationId}`}
              type="file"
              accept="application/pdf,.pdf"
              disabled={upload.isPending}
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) upload.mutate(file)
                e.target.value = ''
              }}
            />
            {upload.isPending && (
              <p role="status" className="text-sm">
                Сохраняю документ…
              </p>
            )}
            {details.data?.documents.map((d) => (
              <div
                className="flex items-center justify-between gap-3 rounded-xl bg-muted/50 p-3"
                key={d.id}
              >
                <div className="flex min-w-0 items-center gap-2">
                  <FileText className="size-4 shrink-0" />
                  <span className="truncate text-sm" title={d.filename}>
                    {d.filename} · v{d.version}
                  </span>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label={`Скачать ${d.filename}, версия ${d.version}`}
                  onClick={() => void download(d)}
                >
                  <Download className="size-4" />
                </Button>
              </div>
            ))}
          </section>
          <section className="space-y-3">
            <h4 className="font-medium">Копии графика</h4>
            <p className="text-sm text-muted-foreground">
              Создайте черновик, измените будущие платежи и отдельно подтвердите замену. Оплаченные
              строки сохранятся.
            </p>
            <Button
              variant="outline"
              className="h-auto min-h-11 max-w-full whitespace-normal py-2 text-left"
              disabled={copy.isPending}
              onClick={() => copy.mutate()}
            >
              <Copy className="size-4" />
              Скопировать текущий график
            </Button>
            {details.data?.drafts.map((d) => (
              <div key={d.id} className="flex items-center justify-between gap-3 text-sm">
                <span>
                  {d.appliedAt ? 'Применённая копия' : 'Черновик'} · {d.payments.length} платежей
                </span>
                {!d.appliedAt && (
                  <Button variant="ghost" onClick={() => setDraft(d)}>
                    Редактировать
                  </Button>
                )}
              </div>
            ))}
          </section>
          <Dialog
            open={draft !== null}
            onOpenChange={(open) => {
              if (!open) setDraft(null)
            }}
          >
            <DialogContent className="sm:max-w-2xl">
              <DialogHeader>
                <DialogTitle>Копия графика</DialogTitle>
                <DialogDescription>
                  Сохранение черновика не меняет текущий график и остатки.
                </DialogDescription>
              </DialogHeader>
              {draft && (
                <ScheduleEditor
                  key={`${draft.id}-${draft.revision}`}
                  draft={draft}
                  obligationId={obligationId}
                  onSaved={setDraft}
                  onApply={() => setConfirm(true)}
                  calculation={calculation?.draft.id === draft.id ? calculation : null}
                  onCalculated={setCalculation}
                />
              )}
            </DialogContent>
          </Dialog>
          <Dialog open={confirm} onOpenChange={setConfirm}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Применить новый график?</DialogTitle>
                <DialogDescription>
                  Неоплаченные строки будут заменены сохранённым черновиком. История старого графика
                  и все оплаты сохранятся. Если данные уже изменились, Grace отклонит замену.
                </DialogDescription>
              </DialogHeader>
              <Button disabled={apply.isPending} onClick={() => apply.mutate(draft)}>
                Подтвердить замену
              </Button>
              <Button variant="outline" onClick={() => setConfirm(false)}>
                Отмена
              </Button>
            </DialogContent>
          </Dialog>
        </div>
      </Disclosure>
    </>
  )
}
