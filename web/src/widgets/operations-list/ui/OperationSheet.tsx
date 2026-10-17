import { CreateReceiptForm } from '@/features/create-receipt'
import type { Account, Category, FinancialEvent, Receipt } from '@/shared/api'
import { eventTimeText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { CategoryIcon } from '@/shared/ui/category-icon'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Money } from '@/shared/ui/money'
import { Notice, StatusBadge } from '@/shared/ui/status'
import { OperationActions, OperationHistory } from '@/features/edit-operation'
import { LoanAllocationSection } from '@/features/loan-allocation'
import {
  kindLabels,
  kindIcons,
  sourceLabels,
  timeText,
  accountText,
  direction,
} from './operation-display'

export function EventIcon({ event }: { event: FinancialEvent }) {
  const Icon = event.kind === 'purchase' ? null : kindIcons[event.kind]
  return (
    <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-secondary text-secondary-foreground">
      {Icon ? (
        <Icon aria-hidden="true" className="size-5" />
      ) : (
        <CategoryIcon icon={event.categoryIcon} />
      )}
    </span>
  )
}

export function OperationSheet({
  event,
  accounts,
  categories,
  receipt,
  onClose,
  onChanged,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
  receipt?: Receipt
  onClose: () => void
  onChanged: (next: FinancialEvent) => void
}) {
  const posting = event.postings[0]
  // Only facts that exist are shown; nothing reads «Не указано».
  const facts: [string, string][] = [
    ['Вид', kindLabels[event.kind]],
    ['Дата', event.occurredAt ? eventTimeText(event) : `${eventTimeText(event)}`],
    [event.kind === 'internal_transfer' ? 'Счета' : 'Счёт / карта', accountText(event, accounts)],
    ...(event.kind === 'purchase'
      ? [['Категория', event.categoryName ?? 'Без категории'] as [string, string]]
      : []),
    ...(event.merchant ? [['Продавец', event.merchant] as [string, string]] : []),
    ...(event.description ? [['Описание', event.description] as [string, string]] : []),
    ['Источник', sourceLabels[event.source] ?? event.source],
  ]
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent
        data-motion="menu"
        className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(420px,calc(100%-1rem))] translate-x-0 translate-y-0 rounded-r-none sm:max-w-[420px]"
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <EventIcon event={event} />
            <span className="min-w-0 break-words">
              {event.merchant || event.description || kindLabels[event.kind]}
            </span>
          </DialogTitle>
          <DialogDescription>{timeText(event)}</DialogDescription>
        </DialogHeader>
        {event.voidedAt && (
          <Notice tone="warning" title="Операция отменена">
            {event.voidReason} Она не учитывается в остатках и статистике.
          </Notice>
        )}
        {event.editedByUser && !event.voidedAt && (
          <p>
            <StatusBadge tone="info">Изменена вручную</StatusBadge>
          </p>
        )}
        {posting && (
          <p>
            <Money
              valueMinor={posting.amountMinor}
              currency={posting.currency}
              direction={direction(event)}
              size="lg"
            />
          </p>
        )}
        <dl className="grace-fields">
          {facts.map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
        <OperationActions
          event={event}
          accounts={accounts}
          categories={categories}
          onChanged={onChanged}
        />
        {event.kind === 'debt_payment' && !event.voidedAt && (
          <LoanAllocationSection eventId={event.id} />
        )}
        <OperationHistory event={event} accounts={accounts} categories={categories} />
        {receipt && <ReceiptDetails receipt={receipt} />}
        {event.kind === 'purchase' && !receipt && !event.voidedAt && (
          <CreateReceiptForm purchase={event} />
        )}
      </DialogContent>
    </Dialog>
  )
}

export function ReceiptDetails({ receipt }: { receipt: Receipt }) {
  return (
    <section aria-label="Чек" className="rounded-xl bg-muted/50 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2 text-sm">
        <span className="font-medium">Чек · {receipt.seller}</span>
        <Money valueMinor={receipt.totalMinor} currency={receipt.currency} size="sm" />
      </div>
      <ul className="space-y-1 text-sm text-muted-foreground">
        {receipt.items.map((item) => (
          <li key={item.id} className="flex items-start justify-between gap-3">
            <span>
              {item.position}. {item.name} · {item.quantityMilli / 1000}
            </span>
            <span className="shrink-0 tabular-nums">
              {formatMoney(item.lineTotalMinor, receipt.currency)}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
