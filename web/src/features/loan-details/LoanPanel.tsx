import { csrfToken } from '@/shared/model/auth-session'
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Download, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { rowKey, type ScheduleRow } from '@/features/schedule-builder/model/schedule-rows'
import { ScheduleBuilder } from '@/features/schedule-builder/ui/ScheduleBuilder'
import { apiClient } from '@/shared/api/client'
import type { components } from '@/shared/api/generated/schema'
import { minorToInput, parseMinor } from '@/shared/lib/finance-display'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Label } from '@/shared/ui/label'
import { Checkbox } from '@/shared/ui/checkbox'
import { Disclosure } from '@/shared/ui/disclosure'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/shared/ui/dialog'
import { LoadingSkeleton } from '@/shared/ui/skeleton'

type Draft = components['schemas']['PaymentScheduleDraft']
type Loan = components['schemas']['LoanDetails']
export function LoanPanel({ obligationId }: { obligationId: string }) {
  const query = useQueryClient()
  const key = ['loan-details', obligationId]
  const [draft, setDraft] = useState<Draft | null>(null)
  const [confirm, setConfirm] = useState(false)
  const details = useQuery({
    queryKey: key,
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/loans/{obligationId}', {
        params: { path: { obligationId } },
      })
      if (error) throw new Error(error.message)
      return data!
    },
  })
  const refresh = () => query.invalidateQueries({ queryKey: key })
  const copy = useMutation({
    mutationFn: async () => {
      const { data, error } = await apiClient.POST('/api/v1/loans/{obligationId}/schedule-drafts', {
        params: { header: { 'X-CSRF-Token': csrfToken }, path: { obligationId } },
      })
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (d) => {
      setDraft(d)
      void refresh()
    },
    onError: (e) => toast.error(e.message),
  })
  const upload = useMutation({
    mutationFn: async (file: File) => {
      if (file.size > 16 * 1024 * 1024) throw new Error('Файл должен быть не больше 16 МиБ.')
      const { error } = await apiClient.POST('/api/v1/loans/{obligationId}/documents', {
        params: { header: { 'X-CSRF-Token': csrfToken }, path: { obligationId } },
        body: { file: '' },
        bodySerializer: () => {
          const form = new FormData()
          form.append('file', file)
          return form
        },
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      toast.success('Оригинал документа сохранён')
      void refresh()
    },
    onError: (e) => toast.error(e.message),
  })
  const apply = useMutation({
    mutationFn: async () => {
      if (!draft) return
      const { error } = await apiClient.POST(
        '/api/v1/loans/{obligationId}/schedule-drafts/{draftId}/apply',
        {
          params: {
            header: { 'X-CSRF-Token': csrfToken },
            path: { obligationId, draftId: draft.id },
          },
          body: { revision: draft.revision, confirm: true },
        },
      )
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      setConfirm(false)
      setDraft(null)
      void refresh()
      void query.invalidateQueries({ queryKey: ['obligations'] })
      void query.invalidateQueries({ queryKey: ['cash-position'] })
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
  return (
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
              onSaved={() => void refresh()}
            />
          )
        )}
        <section className="space-y-3">
          <h4 className="font-medium">Документы договора</h4>
          <p className="text-sm text-muted-foreground">
            PDF до 16 МиБ. Каждая загрузка сохраняется новой версией. Оригиналы доступны только вам;
            общий лимит — 100 МиБ.
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
                onSaved={(d) => {
                  setDraft(d)
                  void refresh()
                }}
                onApply={() => setConfirm(true)}
              />
            )}
          </DialogContent>
        </Dialog>
        <Dialog open={confirm} onOpenChange={setConfirm}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Применить новый график?</DialogTitle>
              <DialogDescription>
                Неоплаченные строки будут заменены сохранённым черновиком. История старого графика и
                все оплаты сохранятся. Если данные уже изменились, Grace отклонит замену.
              </DialogDescription>
            </DialogHeader>
            <Button disabled={apply.isPending} onClick={() => apply.mutate()}>
              Подтвердить замену
            </Button>
            <Button variant="outline" onClick={() => setConfirm(false)}>
              Отмена
            </Button>
          </DialogContent>
        </Dialog>
      </div>
    </Disclosure>
  )
}
function LoanForm({
  initial,
  obligationId,
  onSaved,
}: {
  initial: Loan
  obligationId: string
  onSaved: () => void
}) {
  const [reminders, setReminders] = useState(initial.reminders)
  const save = useMutation({
    mutationFn: async (form: FormData) => {
      const principal = String(form.get('principal'))
      const rate = String(form.get('rate'))
      const amount = principal ? parseMinor(principal) : null
      const basis = rate ? parseMinor(rate) : null
      if (
        (principal && (amount === null || amount <= 0)) ||
        (rate && (basis === null || basis < 0))
      )
        throw new Error('Проверьте сумму и ставку: не больше двух знаков после запятой.')
      const { error } = await apiClient.PATCH('/api/v1/loans/{obligationId}', {
        params: { header: { 'X-CSRF-Token': csrfToken }, path: { obligationId } },
        body: {
          lender: String(form.get('lender')),
          agreementNumber: String(form.get('number')),
          agreementOn: String(form.get('date')) || null,
          principalMinor: amount,
          annualRateBasisPoints: basis,
          reminders,
          revision: initial.revision,
        },
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      toast.success('Сведения о кредите сохранены')
      onSaved()
    },
    onError: (e) => toast.error(e.message),
  })
  return (
    <form
      className="space-y-4"
      onSubmit={(e) => {
        e.preventDefault()
        save.mutate(new FormData(e.currentTarget))
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        {[
          ['lender', 'Кредитор', initial.lender],
          ['number', 'Номер договора', initial.agreementNumber],
          ['date', 'Дата договора', initial.agreementOn ?? ''],
          ['principal', 'Сумма выдачи · необязательно', minorToInput(initial.principalMinor)],
          [
            'rate',
            'Годовая ставка, % · необязательно',
            minorToInput(initial.annualRateBasisPoints),
          ],
        ].map(([name, label, value]) => (
          <div className="space-y-2" key={name}>
            <Label htmlFor={`${obligationId}-${name}`}>{label}</Label>
            {name === 'principal' ? (
              <MoneyInput
                id={`${obligationId}-${name}`}
                name={name}
                currency="RUB"
                defaultValue={value}
              />
            ) : (
              <Input
                id={`${obligationId}-${name}`}
                name={name}
                type={name === 'date' ? 'date' : 'text'}
                inputMode={name === 'rate' ? 'decimal' : undefined}
                defaultValue={value}
                maxLength={160}
              />
            )}
          </div>
        ))}
      </div>
      <p className="text-xs text-muted-foreground">
        Ставка и сумма выдачи — справочные данные. График задаётся по договору; Grace не
        рассчитывает условия банка автоматически.
      </p>
      <label className="flex min-h-11 items-start gap-3 text-sm">
        <Checkbox checked={reminders} onCheckedChange={(v) => setReminders(v === true)} />
        <span>Напоминать по подтверждённой почте за 3 и 1 день</span>
      </label>
      <Button variant="outline" disabled={save.isPending}>
        Сохранить сведения
      </Button>
    </form>
  )
}
function ScheduleEditor({
  draft,
  obligationId,
  onSaved,
  onApply,
}: {
  draft: Draft
  obligationId: string
  onSaved: (d: Draft) => void
  onApply: () => void
}) {
  // SPEC-0037: the same three ways to fill a schedule as a new obligation; paid rows stay locked.
  const [rows, setRows] = useState<ScheduleRow[]>(() =>
    draft.payments.map((p) => ({
      key: rowKey(),
      dueOn: p.dueOn,
      amount: minorToInput(p.amountMinor),
      locked: p.paidMinor > 0,
      sourceId: p.sourceId,
      paidMinor: p.paidMinor,
    })),
  )
  const [dirty, setDirty] = useState(false)
  const save = useMutation({
    mutationFn: async () => {
      const payments = rows.map((p) => {
        const amountMinor = parseMinor(p.amount)
        if (amountMinor === null || amountMinor <= 0)
          throw new Error('Укажите положительную сумму каждого платежа.')
        return {
          sourceId: p.sourceId ?? '',
          dueOn: p.dueOn,
          paidMinor: p.paidMinor ?? 0,
          amountMinor,
        }
      })
      const { data, error } = await apiClient.PATCH(
        '/api/v1/loans/{obligationId}/schedule-drafts/{draftId}',
        {
          params: {
            header: { 'X-CSRF-Token': csrfToken },
            path: { obligationId, draftId: draft.id },
          },
          body: { revision: draft.revision, payments },
        },
      )
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (d) => {
      onSaved(d)
      toast.success('Черновик сохранён')
    },
    onError: (e) => toast.error(e.message),
  })
  return (
    <div className="space-y-4">
      <ScheduleBuilder
        rows={rows}
        onChange={(next) => {
          setDirty(true)
          setRows(next)
        }}
        currency="RUB"
        idPrefix={`draft-${draft.id}`}
        initialMode="manual"
      />
      <div className="flex flex-wrap gap-2">
        <Button disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
          Сохранить черновик
        </Button>
        <Button
          variant="outline"
          className="h-auto min-h-11 max-w-full whitespace-normal py-2"
          disabled={dirty || save.isPending}
          onClick={onApply}
        >
          Применить сохранённый график
        </Button>
      </div>
      {dirty && (
        <p className="text-sm text-muted-foreground">
          Сначала сохраните изменения. Даты должны идти по порядку.
        </p>
      )}
    </div>
  )
}
