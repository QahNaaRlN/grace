import { useState, type FormEvent } from 'react'

import { reconciliationOutcomeLabels } from '@/entities/account'
import type { Account } from '@/shared/api'
import { fromLocalDateTimeInput, localDateTimeInput } from '@/shared/lib/date'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { formatMoney, parseMinor } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Notice } from '@/shared/ui/status'
import {
  previewReconciliation,
  useConfirmReconciliation,
  type ReconciliationPreview,
} from '../api/reconcile-account'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

export function ReconcileDialog({ account, onClose }: { account: Account; onClose: () => void }) {
  const [key] = useState(createIdempotencyKey)
  const confirmation = useConfirmReconciliation(account.id, key)
  const [stated, setStated] = useState('')
  const [moment, setMoment] = useState(() => localDateTimeInput(new Date().toISOString()))
  const [preview, setPreview] = useState<ReconciliationPreview | null>(null)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState('')

  function input() {
    const statedMinor = parseMinor(stated)
    if (statedMinor === null) {
      setError('Введите остаток из банка: число, не больше двух знаков после запятой.')
      return null
    }
    if (!moment) {
      setError('Укажите дату и время остатка.')
      return null
    }
    return { asOf: fromLocalDateTimeInput(moment), statedMinor }
  }

  async function check(submit: FormEvent) {
    submit.preventDefault()
    const body = input()
    if (!body || pending) return
    setPending(true)
    setError('')
    const result = await previewReconciliation(account.id, body)
    setPending(false)
    if (!result.data) return setError(errorText(result.error, 'Не удалось посчитать сверку.'))
    setPreview(result.data)
  }

  async function confirm() {
    if (!preview || pending) return
    setPending(true)
    setError('')
    const result = await confirmation.mutateAsync(preview)
    setPending(false)
    if (!result.data) {
      setPreview(null)
      return setError(errorText(result.error, 'Не удалось сохранить сверку.'))
    }
    setDone(reconciliationOutcomeLabels[result.data.reconciliation.outcome])
  }

  const money = (value: number) => formatMoney(value, account.currency)
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Сверить остаток</DialogTitle>
          <DialogDescription>
            Введите остаток из приложения банка и момент, на который он показан. Grace посчитает
            свой остаток на этот момент.
          </DialogDescription>
        </DialogHeader>
        {done ? (
          <>
            <Notice tone="success" title="Сверка сохранена">
              {done}.
            </Notice>
            <div className="flex justify-end">
              <Button onClick={onClose}>Готово</Button>
            </div>
          </>
        ) : (
          <>
            <form onSubmit={check} className="space-y-4">
              <Field label="Остаток в банке">
                <MoneyInput
                  allowNegative
                  value={stated}
                  currency={account.currency}
                  onChange={(e) => {
                    setStated(e.target.value)
                    setPreview(null)
                  }}
                />
              </Field>
              <Field label="На дату и время">
                <Input
                  type="datetime-local"
                  value={moment}
                  onChange={(e) => {
                    setMoment(e.target.value)
                    setPreview(null)
                  }}
                />
              </Field>
              {!preview && (
                <div className="flex justify-end">
                  <Button type="submit" disabled={pending}>
                    {pending ? 'Считаем…' : 'Проверить'}
                  </Button>
                </div>
              )}
            </form>
            {preview && (
              <section aria-label="Результат сверки" className="space-y-3">
                <dl className="grace-fields">
                  {preview.outcome === 'opening_set' ? (
                    <>
                      <div>
                        <dt>Операции до этого момента</dt>
                        <dd>{money(preview.computedMinor)}</dd>
                      </div>
                      <div>
                        <dt>Начальный остаток</dt>
                        <dd>{money(preview.proposedOpeningMinor ?? 0)}</dd>
                      </div>
                    </>
                  ) : (
                    <>
                      <div>
                        <dt>Остаток в Grace</dt>
                        <dd>{money(preview.computedMinor)}</dd>
                      </div>
                      <div>
                        <dt>Разница</dt>
                        <dd>{money(preview.differenceMinor)}</dd>
                      </div>
                    </>
                  )}
                </dl>
                <p className="text-sm text-muted-foreground">
                  {preview.outcome === 'opening_set' &&
                    'Счёт ещё не сверялся. Grace запишет начальный остаток так, чтобы баланс совпал с банком, и счёт станет сверенным.'}
                  {preview.outcome === 'matched' &&
                    'Остатки совпадают. Счёт будет отмечен сверенным на этот момент.'}
                  {preview.outcome === 'adjusted' &&
                    'Остатки расходятся. Grace создаст операцию «Корректировка остатка» на разницу; она не попадёт в расходы и доходы.'}
                </p>
                <div className="flex flex-wrap justify-end gap-2">
                  <Button variant="outline" onClick={() => setPreview(null)} disabled={pending}>
                    Изменить данные
                  </Button>
                  <Button onClick={() => void confirm()} disabled={pending}>
                    {pending ? 'Сохраняем…' : 'Подтвердить сверку'}
                  </Button>
                </div>
              </section>
            )}
            {error && (
              <Notice tone="danger" title="Сверка не выполнена">
                {error}
              </Notice>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
