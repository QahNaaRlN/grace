import { InstallmentLinks } from '@/features/installment/manage'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CreditCard, Pencil, Plus, Scale } from 'lucide-react'

import {
  accountQueries,
  accountTypeLabels,
  reconciliationOutcomeLabels,
  type PaymentCard,
} from '@/entities/account'
import { CardDialog, EditAccountDialog, useToggleCard } from '@/features/edit-account'
import { ReconcileDialog } from '@/features/reconcile-account'
import type { Account } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { momentText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { AccountIcon } from '@/shared/ui/account-icon'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Disclosure } from '@/shared/ui/disclosure'
import { Money } from '@/shared/ui/money'
import { Notice, StatusBadge } from '@/shared/ui/status'
import { CreditCardPanel } from './CreditCardPanel'

function errorText(error: unknown, fallback: string) {
  return (error as { message?: string } | undefined)?.message ?? fallback
}

// SPEC-0040: Grace has no bank connection, so «синхронизация» means the last
// reconciliation or the last confirmed import.
function syncStatus(account: Account) {
  const last = account.lastReconciliation
  const imported = account.lastImportConfirmedAt
    ? `импорт ${momentText(account.lastImportConfirmedAt)}`
    : null
  if (last?.stale)
    return { tone: 'warning' as const, text: 'Нужна повторная сверка', detail: imported }
  if (last)
    return { tone: 'success' as const, text: `Сверен ${momentText(last.asOf)}`, detail: imported }
  return { tone: 'unknown' as const, text: 'Не сверён', detail: imported }
}

export function Accounts({ items }: { items: Account[] }) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = items.find((item) => item.id === selectedId) ?? null
  return (
    <>
      <ul className="divide-y divide-border rounded-2xl border bg-card p-2">
        {items.map((account) => {
          const status = syncStatus(account)
          return (
            <li key={account.id}>
              <button
                type="button"
                onClick={() => setSelectedId(account.id)}
                className="flex min-h-14 w-full items-center gap-3 rounded-xl px-3 py-3 text-left hover:bg-accent"
              >
                <AccountIcon type={account.type} />
                <span className="min-w-0 flex-1">
                  <strong className="block break-words">{accountLabel(account)}</strong>
                  <span className="mt-1 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                    <StatusBadge tone={status.tone}>{status.text}</StatusBadge>
                    {status.detail && <span>{status.detail}</span>}
                  </span>
                </span>
                <Money
                  valueMinor={account.balanceMinor}
                  currency={account.currency}
                  className="max-w-[48%] text-right"
                />
              </button>
            </li>
          )
        })}
      </ul>
      {selected && (
        <AccountSheet account={selected} accounts={items} onClose={() => setSelectedId(null)} />
      )}
    </>
  )
}

function AccountSheet({
  account,
  accounts,
  onClose,
}: {
  account: Account
  accounts: Account[]
  onClose: () => void
}) {
  const [dialog, setDialog] = useState<'reconcile' | 'edit' | 'card' | null>(null)
  const [card, setCard] = useState<PaymentCard | null>(null)
  const toggle = useToggleCard(account.id)
  const [cardError, setCardError] = useState('')
  const cards = account.paymentInstruments ?? []
  const canHaveCards = account.type === 'debit' || account.type === 'credit'
  const status = syncStatus(account)
  const last = account.lastReconciliation

  async function toggleCard(target: PaymentCard) {
    setCardError('')
    const result = await toggle.mutateAsync(target)
    if (!result.data) return setCardError(errorText(result.error, 'Не удалось изменить карту.'))
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent
        data-motion="menu"
        className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(440px,calc(100%-1rem))] translate-x-0 translate-y-0 overflow-y-auto rounded-r-none sm:max-w-[440px]"
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <AccountIcon type={account.type} />
            <span className="min-w-0 break-words">{account.name}</span>
          </DialogTitle>
          <DialogDescription>
            {accountTypeLabels[account.type]} · {account.currency}
          </DialogDescription>
        </DialogHeader>
        <p>
          <Money valueMinor={account.balanceMinor} currency={account.currency} size="lg" />
        </p>
        <div className="flex flex-wrap gap-2">
          <Button onClick={() => setDialog('reconcile')}>
            <Scale />
            Сверить остаток
          </Button>
          <Button variant="outline" onClick={() => setDialog('edit')}>
            <Pencil />
            Изменить сведения
          </Button>
        </div>

        {account.type === 'credit' && <CreditCardPanel account={account} accounts={accounts} />}
        {account.type === 'installment' && <InstallmentLinks accountId={account.id} />}

        <section aria-labelledby="account-sync" className="space-y-2">
          <h3 id="account-sync" className="text-sm font-semibold">
            Последняя синхронизация
          </h3>
          <p className="text-sm text-muted-foreground">
            Grace не подключён к банку: данные обновляются сверкой остатка или подтверждённым
            импортом выписки.
          </p>
          <dl className="grace-fields">
            <div>
              <dt>Сверка</dt>
              <dd>
                <StatusBadge tone={status.tone}>
                  {last
                    ? `${reconciliationOutcomeLabels[last.outcome]}, на ${momentText(last.asOf)}`
                    : 'Не сверялся'}
                </StatusBadge>
              </dd>
            </div>
            {last && (
              <div>
                <dt>Остаток в банке</dt>
                <dd>{formatMoney(last.statedMinor, account.currency)}</dd>
              </div>
            )}
            {last?.stale && (
              <div>
                <dt>Почему повторно</dt>
                <dd>Операции до {momentText(last.asOf)} изменились после сверки.</dd>
              </div>
            )}
            <div>
              <dt>Последний импорт</dt>
              <dd>
                {account.lastImportConfirmedAt
                  ? momentText(account.lastImportConfirmedAt)
                  : 'Не было'}
              </dd>
            </div>
          </dl>
        </section>

        {canHaveCards && (
          <section aria-labelledby="account-cards" className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <h3 id="account-cards" className="text-sm font-semibold">
                Карты
              </h3>
              <Button
                variant="ghost"
                onClick={() => {
                  setCard(null)
                  setDialog('card')
                }}
              >
                <Plus />
                Добавить карту
              </Button>
            </div>
            {cards.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Карт нет. Баланс хранится на счёте; карта не создаёт отдельный остаток.
              </p>
            ) : (
              <ul className="divide-y divide-border">
                {cards.map((item) => (
                  <li key={item.id} className="flex flex-wrap items-center gap-2 py-2">
                    <CreditCard aria-hidden="true" className="size-4 text-muted-foreground" />
                    <span className="min-w-0 flex-1">
                      <span className="block">
                        {item.bank} *{item.last4}
                      </span>
                      <span className="block text-sm text-muted-foreground">
                        {item.name}
                        {item.disabledAt ? ' · отключена' : ''}
                      </span>
                    </span>
                    <Button
                      variant="ghost"
                      aria-label={`Изменить карту *${item.last4}`}
                      onClick={() => {
                        setCard(item)
                        setDialog('card')
                      }}
                    >
                      <Pencil />
                    </Button>
                    <Button variant="ghost" onClick={() => void toggleCard(item)}>
                      {item.disabledAt ? 'Включить' : 'Отключить'}
                    </Button>
                  </li>
                ))}
              </ul>
            )}
            {cardError && (
              <Notice tone="danger" title="Карта не изменена">
                {cardError}
              </Notice>
            )}
          </section>
        )}

        <ReconciliationHistory account={account} />

        {dialog === 'reconcile' && (
          <ReconcileDialog account={account} onClose={() => setDialog(null)} />
        )}
        {dialog === 'edit' && (
          <EditAccountDialog account={account} onClose={() => setDialog(null)} />
        )}
        {dialog === 'card' && (
          <CardDialog account={account} card={card} onClose={() => setDialog(null)} />
        )}
      </DialogContent>
    </Dialog>
  )
}

function ReconciliationHistory({ account }: { account: Account }) {
  const history = useQuery({
    ...accountQueries.reconciliations(account),
    enabled: !!account.lastReconciliation,
  })
  if (!account.lastReconciliation) return null
  return (
    <Disclosure title="История сверок">
      {history.isPending ? (
        <p className="px-3 text-sm text-muted-foreground">Загружаем…</p>
      ) : history.isError ? (
        <p className="px-3 text-sm text-destructive">История сверок недоступна.</p>
      ) : (
        <ol className="space-y-2 px-3 text-sm">
          {history.data.map((item) => (
            <li key={item.id}>
              <span className="block">
                {momentText(item.asOf)} — {reconciliationOutcomeLabels[item.outcome]}
              </span>
              <span className="block text-muted-foreground">
                в банке {formatMoney(item.statedMinor, account.currency)}
                {item.outcome === 'adjusted' &&
                  `, разница ${formatMoney(item.differenceMinor, account.currency)}`}
                {item.outcome === 'opening_set' &&
                  `, начальный остаток ${formatMoney(item.differenceMinor, account.currency)}`}
              </span>
            </li>
          ))}
        </ol>
      )}
    </Disclosure>
  )
}
