import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Banknote, FileText, Landmark, Pencil, Plus } from 'lucide-react'

import {
  creditCardQueries,
  graceModelLabels,
  unknownLabels,
  type CreditCard,
} from '@/entities/credit-card'
import {
  CardSnapshotDialog,
  CardTermsDialog,
  useCardReminders,
} from '@/features/credit-card/manage'
import { InstallmentLinks } from '@/features/installment/manage'
import { RepayCardDialog } from '@/features/credit-card/repay'
import type { Account } from '@/shared/api'
import { dateText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { Disclosure } from '@/shared/ui/disclosure'
import { Label } from '@/shared/ui/label'
import { Notice, StatusBadge } from '@/shared/ui/status'

type Dialog = 'repay' | 'snapshot' | 'terms' | 'new-terms' | null

/** SPEC-0051: debt, limit, grace period and minimum payment of a credit card account. */
export function CreditCardPanel({ account, accounts }: { account: Account; accounts: Account[] }) {
  const card = useQuery(creditCardQueries.detail(account.id))
  const [dialog, setDialog] = useState<Dialog>(null)
  const reminders = useCardReminders(account.id)
  const payingAccounts = accounts.filter(
    (item) =>
      item.id !== account.id &&
      item.currency === account.currency &&
      (item.type === 'debit' || item.type === 'cash'),
  )
  if (card.isPending)
    return <p className="text-sm text-muted-foreground">Загружаем условия карты…</p>
  if (card.isError)
    return (
      <Notice tone="danger" title="Карта недоступна">
        {card.error.message}
      </Notice>
    )
  const data = card.data
  const active = data.terms.find((item) => item.id === data.activeTermsId)
  const status = data.status
  const money = (value: number | null | undefined) => formatMoney(value ?? null, account.currency)
  return (
    <section aria-labelledby="credit-card" className="space-y-3">
      <h3 id="credit-card" className="text-sm font-semibold">
        Кредитная карта
      </h3>
      <div className="flex flex-wrap gap-2">
        <Button onClick={() => setDialog('repay')}>
          <Banknote />
          Погасить карту
        </Button>
        <Button variant="outline" onClick={() => setDialog('snapshot')}>
          <Landmark />
          Данные из банка
        </Button>
      </div>
      <dl className="grace-fields">
        <div>
          <dt>Долг</dt>
          <dd>{status.debtMinor === null ? 'Неизвестен' : money(Math.max(status.debtMinor, 0))}</dd>
        </div>
        {status.debtMinor !== null && status.debtMinor < 0 && (
          <div>
            <dt>Переплата</dt>
            <dd>{money(-status.debtMinor)}</dd>
          </div>
        )}
        <div>
          <dt>Доступно</dt>
          <dd>{status.availableMinor === null ? 'Неизвестно' : money(status.availableMinor)}</dd>
        </div>
        {status.source === 'snapshot' && (
          <div>
            <dt>Источник</dt>
            <dd>Данные банка и операции после них: остаток счёта не сверен</dd>
          </div>
        )}
      </dl>
      <GraceBlock data={data} money={money} />
      <InstallmentLinks accountId={account.id} />
      {data.paymentRequirement && (
        <Notice
          tone={data.paymentRequirement.source === 'bank_confirmed' ? 'info' : 'warning'}
          title="Платёж по карте с рассрочками"
        >
          {data.paymentRequirement.reason}
          {data.paymentRequirement.takenOn &&
            ` Снимок на ${dateText(data.paymentRequirement.takenOn)}.`}
        </Notice>
      )}
      {data.bankComparison && data.bankComparison.differenceMinor !== 0 && (
        <Notice tone="warning" title="Долг в банке и в Grace расходится">
          На {dateText(data.bankComparison.takenOn)}: в банке{' '}
          {money(data.bankComparison.bankDebtMinor)}, в Grace{' '}
          {money(data.bankComparison.ledgerDebtMinor)}. Проверьте, все ли операции по карте внесены;
          при необходимости сверьте остаток.
        </Notice>
      )}
      {status.unknown.length > 0 && (
        <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
          {status.unknown.map((code) => (
            <li key={code}>{unknownLabels[code]}</li>
          ))}
        </ul>
      )}
      <Label className="flex items-center gap-2 font-normal">
        <Checkbox
          checked={data.reminders}
          disabled={reminders.isPending || (!active && !data.paymentRequirement)}
          onCheckedChange={(checked) => void reminders.mutateAsync(checked)}
        />
        Напоминать письмом за 3 и 1 день до платежа
      </Label>
      <Disclosure
        title={
          active
            ? `Условия: ${graceModelLabels[active.graceModel].toLowerCase()}`
            : 'Условия не заданы'
        }
      >
        <div className="space-y-2 px-3 text-sm">
          {active && (
            <p>
              Лимит {active.limitMinor === null ? 'неизвестен' : money(active.limitMinor)}
              {active.statementDay !== null &&
                `, выписка ${active.statementDay}-го, ${active.paymentDays} дн. на оплату`}
              {active.graceDays !== null && `, ${active.graceDays} дн. с первой покупки`}. Действуют
              с {dateText(active.effectiveFrom)}
            </p>
          )}
          {data.documents.map((doc) => (
            <a
              key={doc.id}
              className="flex items-center gap-2 underline"
              href={`/api/v1/accounts/${account.id}/credit-card/documents/${doc.id}`}
            >
              <FileText className="size-4" aria-hidden="true" />
              {doc.filename}
            </a>
          ))}
          <div className="flex flex-wrap gap-2">
            {active && (
              <Button variant="outline" onClick={() => setDialog('terms')}>
                <Pencil />
                Исправить
              </Button>
            )}
            <Button variant="ghost" onClick={() => setDialog('new-terms')}>
              <Plus />
              {active ? 'Новые условия с даты' : 'Добавить условия'}
            </Button>
          </div>
        </div>
      </Disclosure>
      {dialog === 'repay' && (
        <RepayCardDialog
          card={account}
          payingAccounts={payingAccounts}
          suggestedMinor={status.graceAmountMinor || status.minimumMinor || null}
          debtMinor={status.debtMinor}
          onClose={() => setDialog(null)}
        />
      )}
      {dialog === 'snapshot' && (
        <CardSnapshotDialog
          accountId={account.id}
          currency={account.currency}
          onClose={() => setDialog(null)}
        />
      )}
      {(dialog === 'terms' || dialog === 'new-terms') && (
        <CardTermsDialog
          accountId={account.id}
          currency={account.currency}
          version={dialog === 'terms' ? active : undefined}
          onClose={() => setDialog(null)}
        />
      )}
    </section>
  )
}

function GraceBlock({
  data,
  money,
}: {
  data: CreditCard
  money: (v: number | null | undefined) => string
}) {
  const status = data.status
  const bank = data.snapshots.find(
    (item) => status.statementClosedOn && item.takenOn >= status.statementClosedOn,
  )
  const grace = status.graceAmountMinor
  return (
    <div className="space-y-2">
      {grace !== null && status.graceDueOn && (
        <p className="flex flex-wrap items-center gap-2">
          {grace === 0 ? (
            <>
              <StatusBadge tone="success">Льготный период сохранён</StatusBadge>
              <span className="text-sm">Платить сейчас не нужно.</span>
            </>
          ) : (
            <span>
              Заплатите <strong>{money(grace)}</strong> до {dateText(status.graceDueOn)}, чтобы не
              платить проценты
              {status.graceEstimate && status.nextStatementOn
                ? ` (оценка до выписки ${dateText(status.nextStatementOn)})`
                : ''}
              .
            </span>
          )}
          {status.graceLost && <StatusBadge tone="danger">Льготный период потерян</StatusBadge>}
          {!status.graceLost && status.graceAtRisk && (
            <StatusBadge tone="warning">Скоро срок</StatusBadge>
          )}
        </p>
      )}
      {status.minimumMinor !== null && status.minimumDueOn && status.minimumMinor > 0 && (
        <p className="text-sm">
          Минимальный платёж: {status.minimumIsUpperBound ? 'не больше ' : ''}
          {money(status.minimumMinor)}, оплатить до {dateText(status.minimumDueOn)}
          {status.overdueMinimumMinor > 0 &&
            `; из них просрочено ${money(status.overdueMinimumMinor)}`}
        </p>
      )}
      {bank && (bank.graceAmountMinor !== null || bank.minimumMinor !== null) && (
        <p className="text-sm text-muted-foreground">
          По данным банка на {dateText(bank.takenOn)}:
          {bank.graceAmountMinor !== null &&
            ` для льготного периода ${money(bank.graceAmountMinor)}${bank.graceDueOn ? ` до ${dateText(bank.graceDueOn)}` : ''}`}
          {bank.minimumMinor !== null &&
            `; минимальный ${money(bank.minimumMinor)}${bank.minimumDueOn ? ` до ${dateText(bank.minimumDueOn)}` : ''}`}
        </p>
      )}
    </div>
  )
}
