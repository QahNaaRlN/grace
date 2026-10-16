import { useState, type FormEvent } from 'react'

import { graceModelLabels, type CreditCardTerms, type GraceModel } from '@/entities/credit-card'
import { today } from '@/shared/lib/date'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { minorToInput, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import { Textarea } from '@/shared/ui/textarea'
import { uploadCardDocument, useSaveCardTerms, type CardTermsInput } from '../api/use-credit-card'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}
// Percent with up to two decimals ("61,9") ↔ basis points (6190).
const percent = (bp: number | null | undefined) => (bp == null ? '' : minorToInput(bp))
const money = (minor: number | null | undefined) => (minor == null ? '' : minorToInput(minor))
const whole = (value: string) => (/^\d+$/.test(value.trim()) ? Number(value.trim()) : null)

export function CardTermsDialog({
  accountId,
  currency,
  version,
  onClose,
}: {
  accountId: string
  currency: string
  /** Edits this version; without it a new version is created. */
  version?: CreditCardTerms
  onClose: () => void
}) {
  const [key] = useState(createIdempotencyKey)
  const save = useSaveCardTerms(accountId, key)
  const [form, setForm] = useState({
    effectiveFrom: version?.effectiveFrom ?? today(),
    limit: money(version?.limitMinor),
    purchaseRate: percent(version?.purchaseRateBasisPoints),
    cashRate: percent(version?.cashRateBasisPoints),
    graceModel: (version?.graceModel ?? 'statement') as GraceModel,
    statementDay: version?.statementDay?.toString() ?? '',
    paymentDays: version?.paymentDays?.toString() ?? '',
    graceDays: version?.graceDays?.toString() ?? '',
    minPercent: percent(version?.minPercentBasisPoints),
    minAmount: money(version?.minAmountMinor),
    minRounding: money(version?.minRoundingMinor),
    minPercentIsCap: version?.minPercentIsCap ?? false,
    minIncludesInterest: version?.minIncludesInterest ?? false,
    note: version?.note ?? '',
  })
  const [file, setFile] = useState<File | null>(null)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const set = (patch: Partial<typeof form>) => setForm((current) => ({ ...current, ...patch }))
  const optional = (text: string) => (text.trim() === '' ? null : parseMinor(text))

  function body(): CardTermsInput | string {
    const statement = form.graceModel === 'statement'
    const statementDay = whole(form.statementDay)
    const paymentDays = whole(form.paymentDays)
    const graceDays = whole(form.graceDays)
    if (!form.effectiveFrom) return 'Укажите дату, с которой действуют условия.'
    if (statement && (!statementDay || statementDay > 31 || !paymentDays))
      return 'Для модели «от выписки» укажите день выписки (1–31) и число дней на оплату.'
    if (form.graceModel === 'from_first_purchase' && !graceDays)
      return 'Укажите длительность льготного периода в днях.'
    const fields = [
      form.limit,
      form.purchaseRate,
      form.cashRate,
      form.minPercent,
      form.minAmount,
      form.minRounding,
    ]
    if (fields.some((text) => text.trim() !== '' && parseMinor(text) === null))
      return 'Суммы и проценты — числа, не больше двух знаков после запятой.'
    return {
      effectiveFrom: form.effectiveFrom,
      limitMinor: optional(form.limit),
      purchaseRateBasisPoints: optional(form.purchaseRate),
      cashRateBasisPoints: optional(form.cashRate),
      graceModel: form.graceModel,
      statementDay: statementDay ?? null,
      paymentDays: paymentDays ?? null,
      graceDays: graceDays ?? null,
      minPercentBasisPoints: optional(form.minPercent),
      minAmountMinor: optional(form.minAmount),
      minRoundingMinor: optional(form.minRounding),
      minPercentIsCap: form.minPercentIsCap,
      minIncludesInterest: form.minIncludesInterest,
      documentId: version?.documentId ?? null,
      note: form.note.trim(),
      revision: version?.revision ?? 0,
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    const input = body()
    if (typeof input === 'string') return setError(input)
    if (pending) return
    setPending(true)
    setError('')
    try {
      if (file) input.documentId = (await uploadCardDocument(accountId, file)).id
      const result = await save.mutateAsync({ termsId: version?.id, body: input })
      if (!result.data) return setError(errorText(result.error, 'Не удалось сохранить условия.'))
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось сохранить условия.')
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !pending && onClose()}>
      <DialogContent className="max-h-dvh overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{version ? 'Изменить условия карты' : 'Условия карты'}</DialogTitle>
          <DialogDescription>
            Перенесите условия из тарифа банка. Неизвестное оставьте пустым — Grace не будет считать
            его нулём.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4">
          <Field label="Действуют с">
            <Input
              type="date"
              value={form.effectiveFrom}
              onChange={(e) => set({ effectiveFrom: e.target.value })}
            />
          </Field>
          <Field label="Кредитный лимит">
            <MoneyInput
              value={form.limit}
              currency={currency}
              onChange={(e) => set({ limit: e.target.value })}
            />
          </Field>
          <Field label="Льготный период">
            <ChoiceSelect
              value={form.graceModel}
              onValueChange={(value) => set({ graceModel: value as GraceModel })}
            >
              {Object.entries(graceModelLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </ChoiceSelect>
          </Field>
          {form.graceModel !== 'from_first_purchase' && form.graceModel !== 'unknown' && (
            <div className="grid grid-cols-2 gap-3">
              <Field label="День выписки" hint="1–31">
                <Input
                  inputMode="numeric"
                  value={form.statementDay}
                  onChange={(e) => set({ statementDay: e.target.value })}
                />
              </Field>
              <Field label="Дней на оплату" hint="После выписки">
                <Input
                  inputMode="numeric"
                  value={form.paymentDays}
                  onChange={(e) => set({ paymentDays: e.target.value })}
                />
              </Field>
            </div>
          )}
          {form.graceModel === 'from_first_purchase' && (
            <Field label="Длительность, дней" hint="С первой покупки после полного погашения">
              <Input
                inputMode="numeric"
                value={form.graceDays}
                onChange={(e) => set({ graceDays: e.target.value })}
              />
            </Field>
          )}
          <div className="grid grid-cols-2 gap-3">
            <Field label="Ставка на покупки, % годовых">
              <Input
                inputMode="decimal"
                value={form.purchaseRate}
                onChange={(e) => set({ purchaseRate: e.target.value })}
              />
            </Field>
            <Field label="Ставка на наличные, % годовых">
              <Input
                inputMode="decimal"
                value={form.cashRate}
                onChange={(e) => set({ cashRate: e.target.value })}
              />
            </Field>
          </div>
          <fieldset className="space-y-3">
            <legend className="text-sm font-semibold">Минимальный платёж</legend>
            <div className="grid grid-cols-3 gap-3">
              <Field label="% от долга">
                <Input
                  inputMode="decimal"
                  value={form.minPercent}
                  onChange={(e) => set({ minPercent: e.target.value })}
                />
              </Field>
              <Field label="Не меньше">
                <MoneyInput
                  value={form.minAmount}
                  currency={currency}
                  onChange={(e) => set({ minAmount: e.target.value })}
                />
              </Field>
              <Field label="Округлять до">
                <MoneyInput
                  value={form.minRounding}
                  currency={currency}
                  onChange={(e) => set({ minRounding: e.target.value })}
                />
              </Field>
            </div>
            <Label className="flex items-center gap-2 font-normal">
              <Checkbox
                checked={form.minPercentIsCap}
                onCheckedChange={(checked) => set({ minPercentIsCap: checked })}
              />
              Процент — верхняя граница, точную сумму банк указывает в выписке
            </Label>
            <Label className="flex items-center gap-2 font-normal">
              <Checkbox
                checked={form.minIncludesInterest}
                onCheckedChange={(checked) => set({ minIncludesInterest: checked })}
              />
              Плюс начисленные проценты
            </Label>
          </fieldset>
          <Field
            label="PDF тарифа"
            hint={
              version?.documentId
                ? 'Уже приложен; новый файл заменит ссылку'
                : 'Необязательно, до 16 МБ'
            }
          >
            <Input
              type="file"
              accept="application/pdf"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </Field>
          <Field label="Примечание">
            <Textarea
              value={form.note}
              maxLength={500}
              onChange={(e) => set({ note: e.target.value })}
            />
          </Field>
          {error && (
            <Notice tone="danger" title="Условия не сохранены">
              {error}
            </Notice>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
              Отмена
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? 'Сохраняем…' : 'Сохранить'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
