import { useState } from 'react'

import type { components } from '@/shared/api/generated/schema'
import { ChoiceSelect } from '@/shared/ui/choice-select'

type Layout = components['schemas']['StatementLayout']

/** Source text is rendered only as React text, never HTML or instructions. */
export function SourceText({
  layout,
  selected = [],
  onSelect,
}: {
  layout: Layout
  selected?: string[]
  onSelect?: (ids: string[]) => void
}) {
  const [page, setPage] = useState('1')
  const spans = layout.spans.filter((span) => span.page === Number(page))
  return (
    <div className="space-y-3">
      <ChoiceSelect aria-label="Страница выписки" value={page} onValueChange={setPage}>
        {layout.pages
          .map((p) => ({
            value: String(p.number),
            label: `Страница ${p.number}`,
          }))
          .map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
      </ChoiceSelect>
      <div
        className="max-h-80 overflow-auto rounded-xl border p-3"
        aria-label={`Текст страницы ${page}`}
      >
        {spans.map((span, index) => (
          <span key={span.id}>
            {index > 0 &&
            (spans[index - 1].line !== span.line || spans[index - 1].block !== span.block) ? (
              <br />
            ) : (
              ' '
            )}
            {onSelect ? (
              <button
                type="button"
                className="min-h-11 rounded border px-2 py-1 aria-pressed:bg-primary aria-pressed:text-primary-foreground"
                aria-pressed={selected.includes(span.id)}
                onClick={() =>
                  onSelect(
                    selected.includes(span.id)
                      ? selected.filter((id) => id !== span.id)
                      : layout.spans
                          .filter((s) => s.id === span.id || selected.includes(s.id))
                          .map((s) => s.id),
                  )
                }
              >
                {span.text}
              </button>
            ) : (
              span.text
            )}
          </span>
        ))}
      </div>
    </div>
  )
}
