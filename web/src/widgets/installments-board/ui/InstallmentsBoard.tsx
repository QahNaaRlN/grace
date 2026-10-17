import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { installmentQueries, type InstallmentPlan } from '@/entities/installment'
import {
  CreateInstallmentForm,
  InstallmentAllocationSection,
  InstallmentPaymentForm,
  InstallmentScheduleEditor,
} from '@/features/installment/manage'
import type { Account, Category, FinancialEvent } from '@/shared/api'
import { dateText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Disclosure } from '@/shared/ui/disclosure'
import { Field } from '@/shared/ui/field'
import { Notice } from '@/shared/ui/status'

type Ledger = { accounts: Account[]; categories: Category[]; events: FinancialEvent[] }
export function InstallmentsBoard(props: Ledger) {
  const plans = useQuery(installmentQueries.list())
  const [creating, setCreating] = useState(false)
  return (
    <Card>
      <CardHeader>
        <CardTitle>Рассрочки и BNPL</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        <Button variant="outline" onClick={() => setCreating(!creating)}>
          {creating ? 'Закрыть форму' : 'Добавить рассрочку / BNPL'}
        </Button>
        {creating && <CreateInstallmentForm {...props} onDone={() => setCreating(false)} />}
        {plans.isPending ? (
          <p>Загружаем рассрочки…</p>
        ) : plans.isError ? (
          <Notice tone="danger" title="Рассрочки недоступны">
            {plans.error.message}
            <Button variant="outline" onClick={() => void plans.refetch()}>
              Повторить
            </Button>
          </Notice>
        ) : plans.data.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Добавьте отдельную рассрочку или свяжите покупку с рассрочкой внутри карты.
          </p>
        ) : (
          plans.data.map((plan) => <PlanCard key={plan.id} plan={plan} {...props} />)
        )}
      </CardContent>
    </Card>
  )
}
function PlanCard({ plan, accounts, events }: Ledger & { plan: InstallmentPlan }) {
  const [eventId, setEventId] = useState('')
  const [recording, setRecording] = useState(false)
  const transfers = events.filter(
    (event) =>
      event.kind === 'internal_transfer' &&
      !event.voidedAt &&
      event.postings.some((p) => p.direction === 'credit' && p.accountId === plan.accountId),
  )
  return (
    <article className="space-y-3 rounded-xl border p-4" aria-label={plan.name}>
      <h3 className="break-words font-semibold">{plan.name}</h3>
      <p className="text-sm text-muted-foreground">
        {plan.provider} ·{' '}
        {plan.mode === 'credit_card' ? 'В составе долга карты' : 'Отдельный договор'}
      </p>
      <dl className="grace-fields">
        <div>
          <dt>Покупка</dt>
          <dd>{formatMoney(plan.principalMinor, plan.currency)}</dd>
        </div>
        <div>
          <dt>Оплачено за время учёта</dt>
          <dd>{formatMoney(plan.paidMinor, plan.currency)}</dd>
        </div>
        <div>
          <dt>Остаток тела плана</dt>
          <dd>{formatMoney(plan.remainingMinor, plan.currency)}</dd>
        </div>
      </dl>
      {!plan.purchaseEventId && (
        <Notice tone="warning" title="Покупка не учтена">
          Исторический остаток на {dateText(plan.observedOn)}. Прошлые расходы и оплаты не
          восстановлены.
        </Notice>
      )}
      {plan.differenceMinor !== null && plan.differenceMinor !== 0 && (
        <Notice tone="warning" title="Долг и график расходятся">
          На долговом счёте {formatMoney(plan.accountDebtMinor, plan.currency)}. Проверьте
          нераспределённые переводы и операции.
        </Notice>
      )}
      {plan.mode === 'credit_card' && (
        <p className="text-sm">
          Обновите данные карты после оплаты; взносы уже могут входить в требование банка.
        </p>
      )}
      <Disclosure title={`График платежей (${plan.payments.length})`}>
        <ul className="space-y-2 py-2">
          {plan.payments.map((row) => (
            <li key={row.id} className="flex flex-wrap justify-between gap-2 text-sm">
              <span>{dateText(row.dueOn)}</span>
              <span>
                {formatMoney(row.amountMinor, plan.currency)} · осталось{' '}
                {formatMoney(row.remainingMinor, plan.currency)}
              </span>
            </li>
          ))}
        </ul>
      </Disclosure>
      <InstallmentScheduleEditor plan={plan} />
      <Disclosure title="Погашение и привязка перевода">
        <div className="space-y-3 py-3">
          <Field label="Уже записанный перевод">
            <ChoiceSelect
              value={eventId}
              onValueChange={(value) => {
                setEventId(value)
                setRecording(false)
              }}
            >
              <option value="">Выберите перевод</option>
              {transfers.map((event) => (
                <option key={event.id} value={event.id}>
                  {dateText(event.occurredDate ?? event.occurredAt?.slice(0, 10) ?? '')} ·{' '}
                  {formatMoney(event.postings[0].amountMinor, plan.currency)}
                </option>
              ))}
            </ChoiceSelect>
          </Field>
          {!eventId && (
            <Button variant="outline" onClick={() => setRecording(!recording)}>
              Записать новое погашение
            </Button>
          )}
          {recording && !eventId && (
            <InstallmentPaymentForm
              plan={plan}
              accounts={accounts}
              onRecorded={(id) => {
                setEventId(id)
                setRecording(false)
              }}
            />
          )}
          {eventId && <InstallmentAllocationSection eventId={eventId} />}
        </div>
      </Disclosure>
    </article>
  )
}
