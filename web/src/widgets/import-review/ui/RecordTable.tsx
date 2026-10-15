import type { KeyboardEvent } from 'react'

import type { components } from '@/shared/api/generated/schema'
import { Checkbox } from '@/shared/ui/checkbox'
import { Money } from '@/shared/ui/money'
import { StatusBadge, type Tone } from '@/shared/ui/status'
import type { Selection } from './RecordReview'

type Record = components['schemas']['ImportRecord']

const kinds: { [key: string]: string } = {
  purchase: 'Покупка',
  income_received: 'Доход',
  debt_payment: 'Погашение',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
  unclassified_transfer: 'Перевод?',
}
const statusLabels: { [key: string]: [string, Tone] } = {
  unresolved: ['Не проверено', 'unknown'],
  create: ['Новая операция', 'info'],
  link_existing: ['Связать', 'info'],
  duplicate_of: ['Дубль', 'info'],
  exclude: ['Исключено', 'warning'],
  published: ['Подтверждено', 'success'],
}

function rowStatus(row: Record) {
  return statusLabels[row.published ? 'published' : row.decision.action] ?? statusLabels.unresolved
}

// One page of draft rows: a checkbox to select and a button that opens the row.
export function RecordTable({
  items,
  offset,
  matching,
  selection,
  disabled,
  onSelection,
  onOpen,
}: {
  items: Record[]
  offset: number
  matching: number
  selection: Selection
  disabled: boolean
  onSelection: (next: Selection) => void
  onOpen: (id: string) => void
}) {
  const ids = selection.mode === 'ids' ? selection.ids : []
  const selectable = (row: Record) => !row.published && row.source.domain === 'transaction'
  // ↑/↓ move between rows, Enter opens (the row button handles Enter itself).
  function moveFocus(event: KeyboardEvent<HTMLTableSectionElement>) {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
    const buttons = Array.from(
      event.currentTarget.querySelectorAll<HTMLButtonElement>('button[data-row]'),
    )
    const index = buttons.indexOf(document.activeElement as HTMLButtonElement)
    if (index < 0) return
    event.preventDefault()
    buttons[
      Math.max(0, Math.min(buttons.length - 1, index + (event.key === 'ArrowDown' ? 1 : -1)))
    ]?.focus()
  }
  return (
    <div className="overflow-x-auto rounded-2xl border bg-card">
      <table className="w-full text-sm">
        <caption className="sr-only">
          Строки {offset + 1}–{offset + items.length} из {matching}
        </caption>
        <thead className="text-left text-muted-foreground">
          <tr className="border-b">
            <th scope="col" className="w-11 px-2 py-2">
              <span className="sr-only">Выбор</span>
            </th>
            <th scope="col" className="px-2 py-2">
              Строка
            </th>
            <th scope="col" className="hidden px-2 py-2 sm:table-cell">
              Дата
            </th>
            <th scope="col" className="hidden px-2 py-2 md:table-cell">
              Вид
            </th>
            <th scope="col" className="px-2 py-2 text-right">
              Сумма
            </th>
            <th scope="col" className="hidden px-2 py-2 lg:table-cell">
              Описание
            </th>
            <th scope="col" className="px-2 py-2">
              Статус
            </th>
            <th scope="col" className="hidden px-2 py-2 sm:table-cell">
              Вопросы
            </th>
          </tr>
        </thead>
        <tbody onKeyDown={moveFocus} className="divide-y divide-border">
          {items.map((row) => {
            const n = row.source.normalized
            const [label, tone] = rowStatus(row)
            const exact = Number.isSafeInteger(n.amountMinor) && /^[A-Z]{3}$/.test(n.currency ?? '')
            return (
              <tr key={row.id} className="h-12 hover:bg-accent/60">
                <td className="px-2">
                  <label className="grid size-11 cursor-pointer place-items-center">
                    <Checkbox
                      className="size-5"
                      aria-label={`Выбрать строку ${row.source.row}`}
                      disabled={!selectable(row) || disabled || selection.mode === 'filter'}
                      checked={selection.mode === 'filter' ? selectable(row) : ids.includes(row.id)}
                      onCheckedChange={(checked) =>
                        onSelection({
                          mode: 'ids',
                          ids: checked ? [...ids, row.id] : ids.filter((id) => id !== row.id),
                        })
                      }
                    />
                  </label>
                </td>
                <td className="px-2">
                  <button
                    type="button"
                    data-row
                    aria-label={`${row.source.sheet}, строка ${row.source.row}`}
                    onClick={() => onOpen(row.id)}
                    className="min-h-11 rounded-lg px-1 text-left font-medium underline-offset-4 hover:underline"
                  >
                    <span aria-hidden="true">{row.source.row}</span>
                    <span
                      aria-hidden="true"
                      className="block text-xs font-normal text-muted-foreground lg:hidden"
                    >
                      {n.description || row.source.sheet}
                    </span>
                  </button>
                </td>
                <td className="hidden whitespace-nowrap px-2 sm:table-cell">
                  {n.date?.local?.slice(0, 10).split('-').reverse().join('.') ?? '—'}
                </td>
                <td className="hidden px-2 md:table-cell">
                  {kinds[n.kind ?? ''] ?? row.source.domain}
                </td>
                <td className="px-2 text-right">
                  {exact ? (
                    <Money valueMinor={n.amountMinor!} currency={n.currency!} size="sm" />
                  ) : (
                    <span className="text-muted-foreground">в исходнике</span>
                  )}
                </td>
                <td
                  className="hidden max-w-[22rem] truncate px-2 lg:table-cell"
                  title={n.description}
                >
                  {n.description}
                </td>
                <td className="px-2">
                  <StatusBadge tone={tone}>{label}</StatusBadge>
                </td>
                <td className="hidden px-2 sm:table-cell">{row.questions ? row.questions : ''}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
