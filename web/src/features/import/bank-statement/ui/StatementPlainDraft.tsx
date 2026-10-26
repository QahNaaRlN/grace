import { useState } from 'react'
import { statementIssueLabel } from '@/entities/bank-statement'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { Disclosure } from '@/shared/ui/disclosure'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { useStatementMutations } from '../api/mutations'
import { usePlainReview } from '../api/use-plain-review'
import { plainFields, plainHeaders } from './plain-labels'
import { StatementPages } from './StatementPages'

type Statement = components['schemas']['BankStatement']
type Reading = components['schemas']['PlainStatementReading']
const empty = (): Reading => ({
  version: 'statement.plain.0058.v1',
  rawAnswer: '',
  parseStatus: 'manual',
  rows: [],
  header: {},
})
export function StatementPlainDraft({ statement: s }: { statement: Statement }) {
  const { manual, remove } = useStatementMutations(s.id)
  const review = usePlainReview(s.id)
  const [reading, setReading] = useState<Reading>(() =>
    structuredClone(s.plain?.candidate ?? empty()),
  )
  const [revision, setRevision] = useState(s.revision)
  const [dirty, setDirty] = useState(false)
  const [viewed, setViewed] = useState<number[]>([])
  const [confirmed, setConfirmed] = useState(false)
  const [error, setError] = useState('')
  const stale = revision !== s.revision
  function load(next: Statement) {
    setReading(structuredClone(next.plain?.candidate ?? empty()))
    setRevision(next.revision)
    setDirty(false)
    setConfirmed(false)
    setError('')
  }
  if (!dirty && stale) load(s)
  function change(next: Reading) {
    setReading(next)
    setDirty(true)
    setConfirmed(false)
  }
  async function save() {
    try {
      load(
        await manual.mutateAsync({
          key: crypto.randomUUID(),
          body: {
            expectedRevision: revision,
            sourceSHA256: s.sha256,
            plain: reading,
            reason: 'Пользователь проверил и изменил таблицу выписки',
          },
        }),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Правки не сохранены.')
    }
  }
  async function check() {
    if (!s.plain || dirty || stale || !confirmed) return
    try {
      load(
        await review.mutateAsync({
          expectedRevision: s.revision,
          sourceSHA256: s.sha256,
          draftDigest: s.plain.digest,
          reviewedPages: viewed,
          rolesConfirmed: true,
        }),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Сверка не сохранена.')
    }
  }
  if (s.deletedAt || s.publishedRows > 0)
    return <p>Импортировано строк: {s.publishedRows}. Результат доступен в проверке импорта.</p>
  const checked = !!s.plain && s.plain.reviewedDigest === s.plain.digest
  return (
    <section className="space-y-4" aria-label="Таблица операций выписки">
      <p>
        Распознано строк: {reading.rows.length}. Сверьте даты, время, суммы и описания с оригиналом.
        Перевод денег ещё не означает расход.
      </p>
      <StatementPages
        id={s.id}
        count={s.pageCount}
        onViewed={(n) => setViewed((v) => (v.includes(n) ? v : [...v, n]))}
      />
      {stale && (
        <p role="alert">
          Документ изменился. Ваши правки сохранены в форме.{' '}
          <Button variant="outline" onClick={() => load(s)}>
            Загрузить актуальную таблицу
          </Button>
        </p>
      )}
      {reading.parseStatus === 'parse_error' && (
        <p role="alert">
          Ответ получен, но таблица разобрана не полностью. Проверьте исходный ответ и исправьте
          таблицу вручную.
        </p>
      )}
      {s.plain?.original.rawAnswer && (
        <Disclosure title="Исходный ответ модели">
          <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words text-xs">
            {s.plain.original.rawAnswer}
          </pre>
        </Disclosure>
      )}
      <Disclosure title="Период и итоги, напечатанные в документе">
        <div className="grid gap-3 sm:grid-cols-2">
          {plainHeaders.map(([key, label]) => (
            <Field label={label} key={key}>
              <Input
                aria-label={label}
                value={reading.header[key] ?? ''}
                onChange={(e) =>
                  change({ ...reading, header: { ...reading.header, [key]: e.target.value } })
                }
              />
            </Field>
          ))}
        </div>
        <p className="text-sm">
          Не указанные в документе поля оставьте пустыми. Доступный остаток не изменяет начальный
          баланс счёта.
        </p>
      </Disclosure>
      <div className="space-y-3">
        {reading.rows.map((row, index) => (
          <Disclosure
            key={row.id}
            title={`${index + 1}. ${row.raw.operationDate ?? 'Дата не указана'} · ${row.raw.accountAmount ?? 'Сумма не указана'} · ${row.raw.description ?? ''}`}
          >
            <div className="grid gap-3 sm:grid-cols-2">
              {plainFields.map(([key, label]) => (
                <Field key={key} label={label}>
                  <Input
                    aria-label={`${label}, строка ${index + 1}`}
                    value={row.raw[key] ?? ''}
                    onChange={(e) =>
                      change({
                        ...reading,
                        rows: reading.rows.map((r) =>
                          r.id === row.id ? { ...r, raw: { ...r.raw, [key]: e.target.value } } : r,
                        ),
                      })
                    }
                  />
                </Field>
              ))}
            </div>
            {row.issues.length > 0 && (
              <p role="status">Нужна проверка: {row.issues.map(statementIssueLabel).join(' ')}</p>
            )}
            <Button
              variant="outline"
              onClick={() =>
                change({ ...reading, rows: reading.rows.filter((r) => r.id !== row.id) })
              }
            >
              Удалить строку {index + 1} из черновика
            </Button>
          </Disclosure>
        ))}
      </div>
      <Button
        variant="outline"
        onClick={() =>
          change({
            ...reading,
            rows: [
              ...reading.rows,
              {
                id: crypto.randomUUID(),
                raw: {},
                operationDate: null,
                bookingDate: null,
                transactionAmount: null,
                accountAmount: null,
                description: null,
                cardLast4: null,
                issues: [],
              },
            ],
          })
        }
      >
        Добавить строку вручную
      </Button>
      <Button disabled={!dirty || stale || manual.isPending} onClick={save}>
        Сохранить правки таблицы
      </Button>
      <p>
        Просмотрено страниц: {viewed.length} из {s.pageCount}.{' '}
        {checked
          ? 'Сверка текущей таблицы сохранена.'
          : 'После просмотра всех страниц подтвердите сверку документа.'}
      </p>
      <label className="flex min-h-11 items-center gap-3">
        <Checkbox
          checked={confirmed}
          disabled={dirty || stale}
          onCheckedChange={(v) => setConfirmed(v === true)}
        />
        Я сверил все страницы, операции и назначение напечатанных итогов
      </label>
      <Button
        disabled={
          dirty ||
          stale ||
          !confirmed ||
          viewed.length !== s.pageCount ||
          review.isPending ||
          checked
        }
        onClick={check}
      >
        Сохранить сверку документа
      </Button>
      <Disclosure title="Удаление исходника">
        <Button
          variant="outline"
          disabled={remove.isPending}
          onClick={async () => {
            try {
              await remove.mutateAsync(s.revision)
            } catch (e) {
              setError(e instanceof Error ? e.message : 'Не удалось удалить источник.')
            }
          }}
        >
          Удалить исходник и полный ответ
        </Button>
      </Disclosure>
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
    </section>
  )
}
