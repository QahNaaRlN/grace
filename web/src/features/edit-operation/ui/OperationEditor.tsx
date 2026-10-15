import { financialEventQueries } from '@/entities/financial-event'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import type { Account, Category, FinancialEvent } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { dateTimeText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Disclosure } from '@/shared/ui/disclosure'
import { kindLabels, fieldLabels } from './operation-labels'
import { EditDialog } from './EditOperationDialog'
import { VoidDialog } from './VoidOperationDialog'

export function OperationActions({
  event,
  accounts,
  categories,
  onChanged,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
  onChanged: (next: FinancialEvent) => void
}) {
  const [mode, setMode] = useState<'edit' | 'void' | null>(null)
  if (event.voidedAt) return null
  return (
    <>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" onClick={() => setMode('edit')}>
          Изменить
        </Button>
        <Button variant="destructive" onClick={() => setMode('void')}>
          Отменить операцию
        </Button>
      </div>
      {mode === 'edit' && (
        <EditDialog
          event={event}
          accounts={accounts}
          categories={categories}
          onClose={() => setMode(null)}
          onSaved={(next) => {
            setMode(null)
            onChanged(next)
          }}
        />
      )}
      {mode === 'void' && (
        <VoidDialog
          event={event}
          onClose={() => setMode(null)}
          onVoided={(next) => {
            setMode(null)
            onChanged(next)
          }}
        />
      )}
    </>
  )
}

export function OperationHistory({
  event,
  accounts,
  categories,
}: {
  event: FinancialEvent
  accounts: Account[]
  categories: Category[]
}) {
  const history = useQuery({
    ...financialEventQueries.history(event),
    enabled: event.revision > 1,
  })
  if (event.revision <= 1) return null
  const currency = event.postings[0]?.currency ?? 'RUB'
  const show = (field: string, value: unknown) => {
    if (value === null || value === undefined || value === '') return '—'
    const text = String(value)
    if (field === 'amountMinor') return formatMoney(Number(value), currency)
    if (field === 'kind') return kindLabels[text] ?? text
    if (field === 'category')
      return categories.find((c) => c.id === text)?.name ?? 'категория удалена'
    if (field === 'account' || field === 'destinationAccount') {
      const a = accounts.find((item) => item.id === text)
      return a ? accountLabel(a) : 'счёт недоступен'
    }
    if (field === 'occurred' && text.includes('T')) return dateTimeText(text)
    return text
  }
  return (
    <div>
      <Disclosure title={`История изменений · ${event.revision - 1}`}>
        {history.isPending ? (
          <p className="px-3 text-sm text-muted-foreground">Загружаем…</p>
        ) : history.isError ? (
          <p className="px-3 text-sm text-destructive">История недоступна.</p>
        ) : (
          <ol className="space-y-3 px-3 text-sm">
            {history.data?.map((item) => (
              <li key={item.revision}>
                <p className="text-muted-foreground">{dateTimeText(item.createdAt)}</p>
                <ul>
                  {item.changes.map((change, i) => (
                    <li key={i}>
                      {fieldLabels[change.field] ?? change.field}:{' '}
                      {change.field === 'voided'
                        ? show(change.field, change.after)
                        : `${show(change.field, change.before)} → ${show(change.field, change.after)}`}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        )}
      </Disclosure>
    </div>
  )
}
