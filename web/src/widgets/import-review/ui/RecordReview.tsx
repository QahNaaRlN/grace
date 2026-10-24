import { useQuery } from '@tanstack/react-query'
import { type FormEvent } from 'react'
import { Search } from 'lucide-react'

import { importBatchQueries, type ImportRecordsView } from '@/entities/import-batch'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { RecordTable } from './RecordTable'

type Batch = components['schemas']['ImportBatch']
type Status = components['schemas']['ImportRecordStatus']
type Sort = ImportRecordsView['sort']
export type View = ImportRecordsView
export type Selection = { mode: 'ids'; ids: string[] } | { mode: 'filter'; total: number }

const MAX_FILTER_SELECTION = 1000
const sortLabels: [Sort, string][] = [
  ['row', 'По строкам листа'],
  ['-date', 'Сначала новые'],
  ['date', 'Сначала старые'],
  ['-amount', 'Сначала крупные'],
  ['amount', 'Сначала мелкие'],
]

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
  const selectionLimit = ['bank_statement_pdf', 'bank_statement_image'].includes(batch.format ?? '')
    ? 100
    : MAX_FILTER_SELECTION
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
              disabled={disabled || matching > selectionLimit}
              onClick={() => onSelection({ mode: 'filter', total: matching })}
            >
              Выбрать все по фильтру ({matching})
            </Button>
          )}
          {matching > selectionLimit && selection.mode === 'ids' && (
            <span className="text-muted-foreground">
              За раз — не больше {selectionLimit.toLocaleString('ru-RU')} строк: сузьте фильтр
              поиском или статусом.
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
        <RecordTable
          items={items}
          offset={offset}
          matching={matching}
          selection={selection}
          disabled={disabled}
          onSelection={onSelection}
          onOpen={onOpen}
        />
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
