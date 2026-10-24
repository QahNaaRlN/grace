import { useInfiniteQuery } from '@tanstack/react-query'

import { importBatchQueries } from '@/entities/import-batch'
import type { components } from '@/shared/api/generated/schema'
import { momentText } from '@/shared/lib/date'
import { Button } from '@/shared/ui/button'
import { StatusBadge } from '@/shared/ui/status'

type Batch = components['schemas']['ImportBatch']

const day = new Intl.DateTimeFormat('ru-RU', {
  timeZone: 'UTC',
  day: 'numeric',
  month: 'short',
  year: 'numeric',
})
function periodText(from?: string | null, to?: string | null) {
  if (!from || !to) return 'Даты не распознаны'
  const a = day.format(new Date(`${from}T00:00:00Z`))
  const b = day.format(new Date(`${to}T00:00:00Z`))
  return a === b ? a : `${a} – ${b}`
}

// SPEC-0044: the history answers «что, когда и с каким итогом загружено».
export function ImportHistory({
  selectedId,
  onOpen,
}: {
  selectedId: string
  onOpen: (id: string) => void
}) {
  const history = useInfiniteQuery(importBatchQueries.batches())
  const items = history.data?.pages.flatMap((page) => page.items) ?? []
  if (history.isError)
    return (
      <p role="alert">
        Не удалось загрузить историю импортов.{' '}
        <Button variant="outline" onClick={() => void history.refetch()}>
          Повторить
        </Button>
      </p>
    )
  if (!history.isPending && items.length === 0) return null
  return (
    <section aria-labelledby="import-history" className="space-y-2">
      <h2 id="import-history" className="text-lg font-semibold">
        История импортов
      </h2>
      <ul className="divide-y divide-border rounded-2xl border bg-card p-2">
        {items.map((batch) => (
          <HistoryRow
            key={batch.id}
            batch={batch}
            selected={batch.id === selectedId}
            onOpen={onOpen}
          />
        ))}
      </ul>
      {history.hasNextPage && (
        <Button
          variant="outline"
          disabled={history.isFetchingNextPage}
          onClick={() => void history.fetchNextPage()}
        >
          {history.isFetchingNextPage ? 'Загружаем…' : 'Показать более ранние'}
        </Button>
      )}
    </section>
  )
}

function HistoryRow({
  batch,
  selected,
  onOpen,
}: {
  batch: Batch
  selected: boolean
  onOpen: (id: string) => void
}) {
  const total = batch.total ?? 0
  const last = batch.lastConfirmation
  return (
    <li>
      <button
        type="button"
        aria-current={selected || undefined}
        onClick={() => onOpen(batch.id)}
        className="grid w-full gap-1 rounded-xl px-3 py-3 text-left hover:bg-accent aria-[current]:bg-accent sm:grid-cols-[minmax(0,2fr)_minmax(0,1.4fr)_minmax(0,1.6fr)_auto] sm:items-center sm:gap-4"
      >
        <span className="min-w-0">
          <strong className="block truncate">{batch.fileName ?? batch.source.sourceId}</strong>
          <span className="block truncate text-sm text-muted-foreground">
            {['bank_statement_pdf', 'bank_statement_image'].includes(batch.format ?? '')
              ? batch.format === 'bank_statement_image'
                ? 'Фото-выписка'
                : 'PDF-выписка'
              : batch.format === 'csv'
                ? 'Таблица CSV'
                : batch.format === 'json'
                  ? 'Снимок JSON'
                  : 'Файл'}
            {batch.fileName && batch.fileName !== batch.source.sourceId
              ? ` · ${batch.source.sourceId}`
              : ''}{' '}
            · загружено {momentText(batch.createdAt)}
          </span>
        </span>
        <span className="text-sm">
          <span className="block">{periodText(batch.periodFrom, batch.periodTo)}</span>
          <span className="block text-muted-foreground">строк: {total}</span>
        </span>
        <span className="text-sm">
          <span className="block">
            Проверено {batch.reviewed ?? 0} из {total}
            {batch.needsReview ? ` · с вопросами ${batch.needsReview}` : ''}
          </span>
          <span className="block text-muted-foreground">
            {last
              ? `Подтверждено ${momentText(last.at)} · создано операций: ${last.createdEvents}`
              : 'Ещё не подтверждалось'}
          </span>
        </span>
        <StatusBadge tone={batch.status === 'completed' ? 'success' : 'info'}>
          {batch.status === 'completed' ? 'Завершён' : 'На проверке'}
        </StatusBadge>
      </button>
    </li>
  )
}
