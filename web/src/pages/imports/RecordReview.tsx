import { useQuery } from '@tanstack/react-query'
import { type FormEvent, type KeyboardEvent } from 'react'
import { Search } from 'lucide-react'

import { importBatchQueries, type ImportRecordsView } from '@/entities/import-batch'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Money } from '@/shared/ui/money'
import { StatusBadge, type Tone } from '@/shared/ui/status'

type Batch = components['schemas']['ImportBatch']
type Record = components['schemas']['ImportRecord']
type Status = components['schemas']['ImportRecordStatus']
type Sort = ImportRecordsView['sort']
export type View = ImportRecordsView
export type Selection = { mode: 'ids'; ids: string[] } | { mode: 'filter'; total: number }

const MAX_FILTER_SELECTION = 1000
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
const sortLabels: [Sort, string][] = [
  ['row', 'По строкам листа'],
  ['-date', 'Сначала новые'],
  ['date', 'Сначала старые'],
  ['-amount', 'Сначала крупные'],
  ['amount', 'Сначала мелкие'],
]

function rowStatus(row: Record) {
  return statusLabels[row.published ? 'published' : row.decision.action] ?? statusLabels.unresolved
}

// SPEC-0044: a compact, server-paged table of rows; details open in a sheet.
export function RecordReview({
  batchId,
  batch,
  view,
  onView,
  selection,
  onSelection,
  onOpen,
  disabled,
}: {
  batchId: string
  batch: Batch
  view: View
  onView: (next: Partial<View>) => void
  selection: Selection
  onSelection: (next: Selection) => void
  onOpen: (id: string) => void
  disabled: boolean
}) {
  const records = useQuery(importBatchQueries.records(batchId, view))
  const counts = batch.counts ?? {}
  const total = batch.total ?? 0
  const evidence = Object.entries(batch.domainCounts ?? {})
    .filter(([domain]) => domain !== 'transaction')
    .reduce((sum, [, n]) => sum + n, 0)
  const items = records.data?.items ?? []
  const matching = records.data?.total ?? items.length
  const offset = Number(view.cursor || 0)
  const ids = selection.mode === 'ids' ? selection.ids : []
  const selectable = (row: Record) => !row.published && row.source.domain === 'transaction'
  const chips: [Status | '', string, number][] = [
    ['', 'Все', total],
    ['needs_review', 'С вопросами', batch.needsReview ?? 0],
    ['unresolved', 'Не проверено', counts.unresolved ?? 0],
    ['create', 'Новая операция', counts.create ?? 0],
    ['link_existing', 'Связать', counts.link_existing ?? 0],
    ['duplicate_of', 'Дубль', counts.duplicate_of ?? 0],
    ['exclude', 'Исключено', counts.exclude ?? 0],
    ['published', 'Подтверждено', counts.published ?? 0],
  ]

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onView({ q: String(new FormData(event.currentTarget).get('q') ?? '').trim(), cursor: '' })
  }
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
    <section aria-label="Строки черновика" className="space-y-4">
      <div className="space-y-2">
        <p className="text-sm font-medium">
          Проверено {batch.reviewed ?? 0} из {total}
        </p>
        <progress
          className="h-2 w-full accent-primary"
          max={Math.max(total, 1)}
          value={batch.reviewed ?? 0}
          aria-label="Прогресс проверки"
        />
      </div>
      <div className="flex flex-wrap gap-2" role="group" aria-label="Листы">
        <Button
          variant={view.domain === 'transaction' ? 'default' : 'outline'}
          aria-pressed={view.domain === 'transaction'}
          onClick={() => {
            onView({ domain: 'transaction', status: '', cursor: '' })
            onSelection({ mode: 'ids', ids: [] })
          }}
        >
          Операции · {total}
        </Button>
        <Button
          variant={view.domain === 'unsupported' ? 'default' : 'outline'}
          aria-pressed={view.domain === 'unsupported'}
          onClick={() => {
            onView({ domain: 'unsupported', status: '', cursor: '' })
            onSelection({ mode: 'ids', ids: [] })
          }}
        >
          Первичные документы · {evidence}
        </Button>
      </div>
      {view.domain === 'transaction' ? (
        <div className="flex flex-wrap gap-2" role="group" aria-label="Статус строк">
          {chips
            .filter(
              ([status, , n]) =>
                status === '' || status === 'unresolved' || status === view.status || n > 0,
            )
            .map(([status, label, n]) => (
              <Button
                key={status || 'all'}
                variant={view.status === status ? 'secondary' : 'ghost'}
                aria-pressed={view.status === status}
                onClick={() => {
                  onView({ status, cursor: '' })
                  onSelection({ mode: 'ids', ids: [] })
                }}
              >
                {label} {n}
              </Button>
            ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">
          Эти листы сохранены как свидетельства. Их данные не публикуются в финансовый журнал.
        </p>
      )}

      <form
        key={`${view.q}|${view.domain}`}
        onSubmit={search}
        className="grid gap-3 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-end"
      >
        <Field label="Поиск по описанию">
          <Input
            name="q"
            type="search"
            defaultValue={view.q}
            maxLength={100}
            placeholder="Магазин, назначение платежа"
          />
        </Field>
        <Field label="Сортировка">
          <ChoiceSelect
            value={view.sort}
            onValueChange={(sort) => onView({ sort: sort as Sort, cursor: '' })}
          >
            {sortLabels.map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </ChoiceSelect>
        </Field>
        <Field label="Строк на странице">
          <ChoiceSelect
            value={String(view.limit)}
            onValueChange={(limit) => onView({ limit: Number(limit) as 50 | 100, cursor: '' })}
          >
            <option value="50">50</option>
            <option value="100">100</option>
          </ChoiceSelect>
        </Field>
        <Button type="submit" variant="outline">
          <Search />
          Найти
        </Button>
      </form>

      {view.domain === 'transaction' && matching > 0 && (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span>
            {selection.mode === 'filter'
              ? `Выбраны все строки по фильтру: ${selection.total}`
              : `Выбрано на странице: ${ids.length}`}
          </span>
          {selection.mode === 'filter' ? (
            <Button variant="ghost" onClick={() => onSelection({ mode: 'ids', ids: [] })}>
              Снять выбор
            </Button>
          ) : (
            <Button
              variant="ghost"
              aria-pressed={false}
              disabled={disabled || matching > MAX_FILTER_SELECTION}
              onClick={() => onSelection({ mode: 'filter', total: matching })}
            >
              Выбрать все по фильтру ({matching})
            </Button>
          )}
          {matching > MAX_FILTER_SELECTION && selection.mode === 'ids' && (
            <span className="text-muted-foreground">
              За раз — не больше 1 000 строк: сузьте фильтр поиском или статусом.
            </span>
          )}
        </div>
      )}

      {records.isPending ? (
        <p className="text-sm text-muted-foreground">Загружаем строки…</p>
      ) : records.isError ? (
        <p role="alert">
          Не удалось загрузить строки.{' '}
          <Button variant="outline" onClick={() => void records.refetch()}>
            Повторить
          </Button>
        </p>
      ) : items.length === 0 ? (
        <p className="rounded-2xl border bg-card p-6 text-center text-muted-foreground">
          {view.q || view.status ? 'По этим условиям строк нет.' : 'Строк нет.'}
        </p>
      ) : (
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
                const exact =
                  Number.isSafeInteger(n.amountMinor) && /^[A-Z]{3}$/.test(n.currency ?? '')
                return (
                  <tr key={row.id} className="h-12 hover:bg-accent/60">
                    <td className="px-2">
                      <label className="grid size-11 cursor-pointer place-items-center">
                        <Checkbox
                          className="size-5"
                          aria-label={`Выбрать строку ${row.source.row}`}
                          disabled={!selectable(row) || disabled || selection.mode === 'filter'}
                          checked={
                            selection.mode === 'filter' ? selectable(row) : ids.includes(row.id)
                          }
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
                    <td className="hidden px-2 sm:table-cell">
                      {row.questions ? row.questions : ''}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      {items.length > 0 && (
        <nav aria-label="Страницы строк" className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-muted-foreground">
            Строки {offset + 1}–{offset + items.length} из {matching}
          </span>
          <Button
            variant="outline"
            disabled={offset === 0}
            onClick={() =>
              onView({ cursor: offset - view.limit > 0 ? String(offset - view.limit) : '' })
            }
          >
            Назад
          </Button>
          <Button
            variant="outline"
            disabled={!records.data?.nextCursor}
            onClick={() => onView({ cursor: records.data?.nextCursor ?? '' })}
          >
            Вперёд
          </Button>
        </nav>
      )}
    </section>
  )
}
