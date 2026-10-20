import { useState } from 'react'

import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { SourceText } from './SourceText'

export function MappingCoverage({
  layout,
  mapping,
}: {
  layout: components['schemas']['StatementLayout']
  mapping: components['schemas']['StatementDraftMapping']
}) {
  const [query, setQuery] = useState('')
  const [page, setPage] = useState(0)
  const [selected, setSelected] = useState<string[]>([])
  if (mapping.status === 'unavailable')
    return (
      <p role="status">
        Разметка недоступна. Откройте сохранённые доказательства строк; исходную разметку сейчас
        исправить нельзя.
      </p>
    )
  const legacy = mapping.status === 'reconstructed_partial'
  const ids = new Set(legacy ? mapping.unclassifiedSpanIds : (mapping.unmappedSpanIds ?? []))
  const all = layout.spans.filter((span) => ids.has(span.id))
  const filtered = all.filter((span) =>
    `${span.page} ${span.text}`.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
  )
  const last = Math.max(0, Math.ceil(filtered.length / 20) - 1)
  const current = Math.min(page, last)
  return (
    <section className="space-y-3 rounded-xl border p-3" aria-label="Разметка и полнота источника">
      {legacy && (
        <p role="status">
          Служебная разметка старого черновика не сохранена. Фрагменты ниже требуют классификации;
          это не список доказанных пропусков модели.
        </p>
      )}
      <p>
        {legacy ? 'Неклассифицированных фрагментов' : 'Непривязанных фрагментов'}: {all.length}
      </p>
      <Input
        aria-label="Поиск фрагментов источника"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setPage(0)
        }}
      />
      <p>
        Найдено: {filtered.length}. Страница списка {current + 1} из {last + 1}.
      </p>
      <ul className="space-y-2">
        {filtered.slice(current * 20, current * 20 + 20).map((span) => (
          <li key={span.id}>
            <Button
              variant="outline"
              className="h-auto min-h-11 whitespace-normal text-left"
              onClick={() => setSelected([span.id])}
            >
              Стр. {span.page}: {span.text}
            </Button>
          </li>
        ))}
      </ul>
      <div className="flex gap-2">
        <Button variant="outline" disabled={current === 0} onClick={() => setPage(current - 1)}>
          Предыдущие фрагменты
        </Button>
        <Button variant="outline" disabled={current === last} onClick={() => setPage(current + 1)}>
          Следующие фрагменты
        </Button>
      </div>
      {selected.length > 0 && <SourceText layout={layout} selected={selected} />}
    </section>
  )
}
