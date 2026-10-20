import { useRef, useState } from 'react'

import { SourceText, StatementReconciliation } from '@/entities/bank-statement'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Textarea } from '@/shared/ui/textarea'
import { emptyRow, initial, rowFields, headerFields } from '../model/manual-fields'
import { useStatementMutations } from '../api/mutations'

type Statement = components['schemas']['BankStatement']
type Proposal = components['schemas']['StatementProposal']
type Row = components['schemas']['StatementRowMapping']

/** A manual mapping selects original fragments. All parsing and eligibility stay in Go. */
export function StatementManualDraft({ statement }: { statement: Statement }) {
  const { manual } = useStatementMutations(statement.id)
  const [proposal, setProposal] = useState<Proposal>(() =>
    structuredClone(statement.draftMapping?.proposal ?? initial()),
  )
  const [target, setTarget] = useState('row:0:operationDate')
  const [reason, setReason] = useState('')
  const request = useRef<{ signature: string; key: string }>(undefined)
  const [error, setError] = useState('')
  const [regroup, setRegroup] = useState(false)
  const [editingRevision, setEditingRevision] = useState(statement.revision)
  const stale = editingRevision !== statement.revision
  const unavailable = !!statement.batchId && !statement.draftMapping?.proposal
  function open() {
    setProposal(structuredClone(statement.draftMapping?.proposal ?? initial()))
    setEditingRevision(statement.revision)
    setTarget('row:0:operationDate')
    setReason('')
    setError('')
    setRegroup(true)
  }
  if (!statement.layout || statement.publishedRows > 0) return null
  if (statement.batchId && !regroup)
    return (
      <Button variant="outline" disabled={unavailable} onClick={open}>
        Исправить группировку до подтверждения
      </Button>
    )
  const parts = target.split(':')
  const rowIndex = Number(parts[1])
  const selected =
    parts[0] === 'row'
      ? (proposal.rows[rowIndex]?.[parts[2] as Exclude<keyof Row, 'direction'>] ?? [])
      : parts[0] === 'label'
        ? (proposal.header.labels[parts[1]] ?? [])
        : parts[0] === 'header'
          ? proposal.header[parts[1] as Exclude<keyof Proposal['header'], 'labels'>]
          : (proposal.service[Number(parts[1])]?.spans ?? [])
  function select(ids: string[]) {
    setProposal((prev) => {
      if (parts[0] === 'row')
        return {
          ...prev,
          rows: prev.rows.map((row, i) => (i === rowIndex ? { ...row, [parts[2]]: ids } : row)),
        }
      if (parts[0] === 'label')
        return {
          ...prev,
          header: { ...prev.header, labels: { ...prev.header.labels, [parts[1]]: ids } },
        }
      if (parts[0] === 'header') return { ...prev, header: { ...prev.header, [parts[1]]: ids } }
      return {
        ...prev,
        service: prev.service.map((group, i) =>
          i === Number(parts[1]) ? { ...group, spans: ids } : group,
        ),
      }
    })
  }
  const options = proposal.rows.flatMap((_, i) =>
    rowFields.map(([field, label]) => ({
      value: `row:${i}:${field}`,
      label: `Строка ${i + 1}: ${label}`,
    })),
  )
  options.push(
    ...headerFields.flatMap(([field, label]) => [
      { value: `header:${field}`, label },
      { value: `label:${field}`, label: `Заголовок: ${label}` },
    ]),
    { value: 'header:currency', label: 'Валюта итогов' },
    { value: 'header:openingBookDate', label: 'Дата начального бухгалтерского остатка' },
    { value: 'header:closingBookDate', label: 'Дата конечного бухгалтерского остатка' },
    ...proposal.service.map((_, i) => ({
      value: `service:${i}`,
      label: `Служебная группа ${i + 1}`,
    })),
  )
  async function save() {
    try {
      if (stale) return
      const body = { expectedRevision: editingRevision, proposal, reason }
      const signature = JSON.stringify(body)
      if (request.current?.signature !== signature)
        request.current = { signature, key: crypto.randomUUID() }
      await manual.mutateAsync({ body, key: request.current.key })
      setError('')
      if (statement.batchId) setRegroup(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Черновик не сохранён.')
    }
  }
  return (
    <div className="space-y-4">
      <h3 className="font-medium">
        {statement.batchId ? 'Исправить текущую разметку' : 'Подготовить строки вручную'}
      </h3>
      <p>
        Строк в форме: {proposal.rows.length}. Служебных групп: {proposal.service.length}.
      </p>
      {statement.draftMapping?.status === 'reconstructed_partial' && (
        <p role="status">
          Служебная разметка старого черновика не сохранена. Строки восстановлены; оставшиеся
          фрагменты нужно проверить и классифицировать.
        </p>
      )}
      {statement.batchId && (
        <p role="status">
          Сохранение сбросит прежние решения по строкам и отметки проверки страниц и ролей.
          Потребуются повторная проверка и новый preview перед подтверждением операций.
        </p>
      )}
      {stale && (
        <section className="space-y-3 rounded-xl border p-3">
          <p role="status">
            Черновик изменился. Ваши правки сохранены в форме. Проверьте актуальные итоги и исходник
            перед повторным сохранением.
          </p>
          {statement.review && (
            <StatementReconciliation review={statement.review} layout={statement.layout} />
          )}
          <p>Строк в актуальном черновике: {statement.review?.rows.length ?? 0}.</p>
          <Button variant="outline" onClick={() => setEditingRevision(statement.revision)}>
            Актуальный черновик проверен — продолжить с моими правками
          </Button>
        </section>
      )}
      <p className="text-sm">
        {JSON.stringify(proposal) === JSON.stringify(statement.draftMapping?.proposal ?? initial())
          ? 'Изменений разметки нет.'
          : 'Есть несохранённые изменения разметки.'}
      </p>
      <p className="text-sm text-muted-foreground">
        Выберите поле, затем его фрагменты на странице. Для суммы используйте фрагменты одной
        строки. Обе валютные колонки сохраняются отдельно. Неприсутствующие остатки оставьте
        пустыми. Остальной текст отметьте служебным после проверки.
      </p>
      <Button
        variant="outline"
        onClick={() => {
          setProposal((p) => ({ ...p, rows: [...p.rows, emptyRow()] }))
          setTarget(`row:${proposal.rows.length}:operationDate`)
        }}
      >
        Добавить строку
      </Button>
      <Button
        variant="outline"
        onClick={() => {
          setProposal((p) => ({ ...p, service: [...p.service, { spans: [], reason: '' }] }))
          setTarget(`service:${proposal.service.length}`)
        }}
      >
        Добавить служебную группу
      </Button>
      <Field label="Заполняемое поле">
        <ChoiceSelect value={target} onValueChange={setTarget}>
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </ChoiceSelect>
      </Field>
      {parts[0] === 'row' && (
        <Field label="Направление для суммы без знака">
          <ChoiceSelect
            value={proposal.rows[rowIndex].direction || 'signed'}
            onValueChange={(v) =>
              setProposal((p) => ({
                ...p,
                rows: p.rows.map((row, i) =>
                  i === rowIndex ? { ...row, direction: v === 'signed' ? '' : v } : row,
                ),
              }))
            }
          >
            {[
              { value: 'signed', label: 'Знак указан в источнике' },
              { value: 'debit', label: 'Списание — проверить заголовок' },
              { value: 'credit', label: 'Зачисление — проверить заголовок' },
            ].map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </ChoiceSelect>
        </Field>
      )}
      {parts[0] === 'service' && (
        <Field label="Причина служебной группы">
          <Textarea
            value={proposal.service[Number(parts[1])]?.reason ?? ''}
            onChange={(e) =>
              setProposal((p) => ({
                ...p,
                service: p.service.map((group, i) =>
                  i === Number(parts[1]) ? { ...group, reason: e.target.value } : group,
                ),
              }))
            }
          />
        </Field>
      )}
      <p className="text-sm">Выбрано фрагментов: {selected.length}</p>
      <SourceText layout={statement.layout} selected={selected} onSelect={select} />
      <Field label="Основание группировки строк и служебных фрагментов">
        <Textarea value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      <Button disabled={!reason.trim() || manual.isPending || stale} onClick={save}>
        Сохранить черновик без записи операций
      </Button>
      {statement.batchId && (
        <Button variant="outline" disabled={manual.isPending} onClick={() => setRegroup(false)}>
          Отменить исправление
        </Button>
      )}
    </div>
  )
}
