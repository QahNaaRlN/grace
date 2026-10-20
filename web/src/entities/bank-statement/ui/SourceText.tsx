import { useEffect, useRef, useState } from 'react'

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
  const [pageChoice, setPageChoice] = useState<{ focusId: string | undefined; page: string }>({
    focusId: selected[0],
    page: String(layout.spans.find((span) => span.id === selected[0])?.page ?? 1),
  })
  const source = useRef<HTMLDivElement>(null)
  const focusId = selected[0]
  const focusPage = layout.spans.find((span) => span.id === focusId)?.page
  const page = pageChoice.focusId === focusId ? pageChoice.page : String(focusPage ?? 1)
  useEffect(() => {
    const target = Array.from(
      source.current?.querySelectorAll<HTMLElement>('[data-span-id]') ?? [],
    ).find((element) => element.dataset.spanId === focusId)
    target?.scrollIntoView?.({ block: 'nearest' })
  }, [focusId, page])
  const spans = layout.spans.filter((span) => span.page === Number(page))
  return (
    <div className="space-y-3">
      <ChoiceSelect
        aria-label="Страница выписки"
        value={page}
        onValueChange={(value) => setPageChoice({ focusId, page: value })}
      >
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
        ref={source}
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
                data-span-id={span.id}
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
              <span
                data-span-id={span.id}
                className={
                  selected.includes(span.id) ? 'bg-primary text-primary-foreground' : undefined
                }
              >
                {span.text}
              </span>
            )}
          </span>
        ))}
      </div>
    </div>
  )
}
