import { type KeyboardEvent } from 'react'
import { ArrowDownToLine, Trash2 } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { type RowParts, type ScheduleRow } from '../model/schedule-rows'

export function ScheduleLine({
  row,
  number,
  currency,
  idPrefix,
  onDate,
  onAmount,
  onAmountKey,
  onFillDown,
  onRemove,
  onParts,
}: {
  row: ScheduleRow
  number: number
  currency: string
  idPrefix: string
  onDate: (value: string) => void
  onAmount: (value: string) => void
  onAmountKey: (event: KeyboardEvent<HTMLInputElement>) => void
  onFillDown?: () => void
  onRemove: () => void
  // SPEC-0053: shown for loan rows; editing marks the parts as the bank's.
  onParts?: (parts: RowParts) => void
}) {
  const parts = row.parts ?? { principal: '', interest: '', other: '' }
  const paid = row.locked ? ' · есть оплата' : ''
  return (
    <>
      <Input
        id={`${idPrefix}-date-${number}`}
        type="date"
        aria-label={`Дата платежа ${number}${paid}`}
        value={row.dueOn}
        disabled={row.locked}
        onChange={(event) => onDate(event.target.value)}
      />
      <MoneyInput
        id={`${idPrefix}-amount-${number}`}
        aria-label={`Сумма платежа ${number}`}
        currency={currency}
        value={row.amount}
        disabled={row.locked}
        onChange={(event) => onAmount(event.target.value)}
        onKeyDown={onAmountKey}
      />
      {onFillDown && !row.locked ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={`Сумму платежа ${number} — во все следующие`}
          title="Заполнить сумму вниз"
          onClick={onFillDown}
        >
          <ArrowDownToLine aria-hidden="true" />
        </Button>
      ) : (
        <span />
      )}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={`Удалить платёж ${number}`}
        disabled={row.locked}
        onClick={onRemove}
      >
        <Trash2 aria-hidden="true" />
      </Button>
      {onParts && (
        <div className="col-span-4 grid grid-cols-3 gap-2 pb-2">
          {(
            [
              ['principal', 'Тело'],
              ['interest', 'Проценты'],
              ['other', 'Прочее'],
            ] as const
          ).map(([name, label]) => (
            <MoneyInput
              key={name}
              id={`${idPrefix}-${name}-${number}`}
              aria-label={`${label}, платёж ${number}`}
              placeholder={label}
              currency={currency}
              value={parts[name]}
              onChange={(event) => onParts({ ...parts, [name]: event.target.value })}
            />
          ))}
          {row.partsSource === 'calculated' && (
            <p className="col-span-3 text-xs text-muted-foreground">
              Рассчитано Grace — сверьте с графиком банка
            </p>
          )}
        </div>
      )}
    </>
  )
}
