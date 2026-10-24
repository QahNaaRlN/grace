import { useState } from 'react'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { Disclosure } from '@/shared/ui/disclosure'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Textarea } from '@/shared/ui/textarea'
import { useStatementMutations } from '../api/mutations'
import { useVisualReview } from '../api/use-visual-review'
import { StatementPages } from './StatementPages'
import { VisualFieldInputs } from './VisualFieldInputs'
import { rowLabels, headerLabels, emptyHeader } from './visual-labels'

type Statement = components['schemas']['BankStatement']
type Reading = components['schemas']['VisualReading']
export function StatementVisualDraft({ statement: s }: { statement: Statement }) {
  const { manual } = useStatementMutations(s.id)
  const [reading, setReading] = useState<Reading>(() =>
    structuredClone(s.visual?.candidate ?? { header: emptyHeader, rows: [] }),
  )
  const [baseRevision, setBaseRevision] = useState(s.revision)
  const [dirty, setDirty] = useState(false)
  const [reason, setReason] = useState('')
  const [rows, setRows] = useState(s.visual?.reviewedRows ?? {})
  const [pages, setPages] = useState(s.reviewedPages)
  const [roles, setRoles] = useState(s.rolesConfirmed)
  const [viewed, setViewed] = useState<number[]>([])
  const [error, setError] = useState('')
  const stale = s.revision !== baseRevision
  const review = useVisualReview(s.id, load)
  if (!dirty && s.revision !== baseRevision) {
    setReading(structuredClone(s.visual?.candidate ?? { header: emptyHeader, rows: [] }))
    setRows(s.visual?.reviewedRows ?? {})
    setPages(s.reviewedPages)
    setRoles(s.rolesConfirmed)
    setBaseRevision(s.revision)
  }

  function load(next: Statement) {
    setReading(structuredClone(next.visual?.candidate ?? { header: emptyHeader, rows: [] }))
    setRows(next.visual?.reviewedRows ?? {})
    setPages(next.reviewedPages)
    setRoles(next.rolesConfirmed)
    setBaseRevision(next.revision)
    setDirty(false)
    setReason('')
    setError('')
  }
  function change(next: Reading) {
    setReading(next)
    setDirty(true)
  }
  async function save() {
    try {
      const next = await manual.mutateAsync({
        key: crypto.randomUUID(),
        body: { expectedRevision: baseRevision, sourceSHA256: s.sha256, reading, reason },
      })
      load(next)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Правки не сохранены.')
    }
  }
  if (s.deletedAt || s.publishedRows > 0) return null
  return (
    <section className="space-y-4">
      <p>
        Распознанные значения — черновик. Сверьте каждую строку с изображением, полноту страниц и
        роли итогов перед импортом.
      </p>
      <p className="text-sm">
        Результат чтения: кандидат. Связь с источником: ручная сверка изображений. Проверено строк:{' '}
        {Object.keys(s.visual?.reviewedRows ?? {}).length} из {s.visual?.candidate.rows.length ?? 0}
        .
      </p>
      <div className="grid items-start gap-4 xl:grid-cols-2">
        <div className="xl:sticky xl:top-4">
          <StatementPages
            id={s.id}
            count={s.pageCount}
            onViewed={(p) =>
              setViewed((current) => (current.includes(p) ? current : [...current, p]))
            }
          />
        </div>
        <div className="space-y-4">
          {stale && (
            <div role="status" className="space-y-2">
              <p>
                Источник изменился. Ввод сохранён; загрузите актуальный черновик перед проверкой.
              </p>
              <Button variant="outline" onClick={() => load(s)}>
                Загрузить актуальный черновик
              </Button>
            </div>
          )}
          <Disclosure title="Итоги, напечатанные в источнике">
            <VisualFieldInputs
              fields={reading.header}
              labels={headerLabels}
              onChange={(header) => change({ ...reading, header })}
            />
          </Disclosure>
          {reading.rows.map((row, i) => (
            <section key={i} className="space-y-3 rounded border p-3">
              <h4 className="font-medium">Операция {i + 1}</h4>
              <Button
                variant="outline"
                onClick={() => change({ ...reading, rows: reading.rows.filter((_, n) => n !== i) })}
              >
                Удалить операцию {i + 1}
              </Button>
              <Disclosure title="Поля и цитаты строки" defaultOpen={reading.rows.length < 4}>
                <Field label="Начало строки — дословный фрагмент источника">
                  <Input
                    value={row.anchor[0]?.quote ?? ''}
                    onChange={(e) =>
                      change({
                        ...reading,
                        rows: reading.rows.map((r, n) =>
                          n === i
                            ? {
                                ...r,
                                anchor: [
                                  {
                                    page: row.anchor[0]?.page ?? 1,
                                    quote: e.target.value,
                                    box: [0, 0, 1, 1],
                                  },
                                ],
                              }
                            : r,
                        ),
                      })
                    }
                  />
                </Field>
                <VisualFieldInputs
                  fields={row.fields}
                  labels={rowLabels}
                  onChange={(fields) =>
                    change({
                      ...reading,
                      rows: reading.rows.map((r, n) => (n === i ? { ...r, fields } : r)),
                    })
                  }
                />
              </Disclosure>
              {row.issues.length > 0 && (
                <div>
                  <p>Неясности: {row.issues.join('; ')}</p>
                  <Button
                    variant="outline"
                    onClick={() =>
                      change({
                        ...reading,
                        rows: reading.rows.map((r, n) => (n === i ? { ...r, issues: [] } : r)),
                      })
                    }
                  >
                    Неясности устранены вручную
                  </Button>
                </div>
              )}
              <label className="flex min-h-11 items-center gap-2">
                <Checkbox
                  checked={Boolean(rows[`row-${i + 1}`])}
                  disabled={dirty || stale || !s.visual?.rowDigests[`row-${i + 1}`]}
                  onCheckedChange={(v) =>
                    setRows((current) => {
                      const next = { ...current }
                      if (v && s.visual) next[`row-${i + 1}`] = s.visual.rowDigests[`row-${i + 1}`]
                      else delete next[`row-${i + 1}`]
                      return next
                    })
                  }
                />
                Все поля и описание строки сверены с изображением
              </label>
            </section>
          ))}
          <Button
            variant="outline"
            onClick={() =>
              change({
                ...reading,
                rows: [
                  ...reading.rows,
                  {
                    anchor: [{ page: 1, quote: '', box: [0, 0, 1, 1] }],
                    fields: {
                      operationDate: null,
                      bookingDate: null,
                      accountAmount: null,
                      accountCurrency: null,
                      direction: null,
                      transactionAmount: null,
                      transactionCurrency: null,
                      description: null,
                    },
                    issues: [],
                  },
                ],
              })
            }
          >
            Добавить операцию вручную
          </Button>
          <Field label="Основание исправлений">
            <Textarea value={reason} onChange={(e) => setReason(e.target.value)} />
          </Field>
          <Button
            disabled={!dirty || stale || !reason.trim() || manual.isPending}
            onClick={() => void save()}
          >
            Сохранить значения
          </Button>
          <fieldset>
            <legend>Полнота операций проверена по страницам</legend>
            {Array.from({ length: s.pageCount }, (_, i) => i + 1).map((p) => (
              <label key={p} className="flex min-h-11 items-center gap-2">
                <Checkbox
                  disabled={dirty || stale || (!viewed.includes(p) && !s.reviewedPages.includes(p))}
                  checked={pages.includes(p)}
                  onCheckedChange={(v) =>
                    setPages(
                      v ? [...pages.filter((n) => n !== p), p] : pages.filter((n) => n !== p),
                    )
                  }
                />
                Страница {p}
              </label>
            ))}
          </fieldset>
          <label className="flex min-h-11 items-center gap-2">
            <Checkbox
              checked={roles}
              disabled={dirty || stale}
              onCheckedChange={(v) => setRoles(v === true)}
            />
            Проверены роли дат, валют, оборотов и учётных остатков
          </label>
          <Button
            disabled={dirty || stale || !s.visual || review.isPending}
            onClick={() =>
              void review
                .mutateAsync({
                  expectedRevision: baseRevision,
                  sourceSHA256: s.sha256,
                  rows,
                  reviewedPages: pages,
                  rolesConfirmed: roles,
                })
                .catch((e: unknown) =>
                  setError(e instanceof Error ? e.message : 'Проверка не сохранена.'),
                )
            }
          >
            Сохранить проверку изображения
          </Button>
          {error && <p role="alert">{error}</p>}
        </div>
      </div>
    </section>
  )
}
