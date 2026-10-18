import { useRef, useState } from 'react'

import { SourceText } from '@/entities/bank-statement'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Textarea } from '@/shared/ui/textarea'
import { useStatementMutations } from '../api/mutations'

type Statement = components['schemas']['BankStatement']
type Proposal = components['schemas']['StatementProposal']
type Row = components['schemas']['StatementRowMapping']
const emptyRow = (): Row => ({
  operationDate: [],
  bookingDate: [],
  accountAmount: [],
  accountCurrency: [],
  transactionAmount: [],
  transactionCurrency: [],
  description: [],
  direction: '',
  directionEvidence: [],
})
const rowFields = [
  ['operationDate', 'Дата и время операции'],
  ['bookingDate', 'Дата и время списания'],
  ['accountAmount', 'Сумма по счёту'],
  ['accountCurrency', 'Валюта счёта'],
  ['transactionAmount', 'Сумма в валюте операции'],
  ['transactionCurrency', 'Валюта операции'],
  ['description', 'Описание'],
  ['directionEvidence', 'Заголовок направления для суммы без знака'],
] as const
const headerFields = [
  ['openingBookBalance', 'Начальный бухгалтерский остаток'],
  ['closingBookBalance', 'Конечный бухгалтерский остаток'],
  ['availableBalanceObserved', 'Доступный остаток банка'],
  ['debitTurnover', 'Оборот списаний'],
  ['creditTurnover', 'Оборот зачислений'],
] as const
const initial = (): Proposal => ({
  rows: [emptyRow()],
  header: {
    openingBookDate: [],
    closingBookDate: [],
    openingBookBalance: [],
    closingBookBalance: [],
    availableBalanceObserved: [],
    debitTurnover: [],
    creditTurnover: [],
    currency: [],
    labels: {},
  },
  service: [],
})

/** A manual mapping selects original fragments. All parsing and eligibility stay in Go. */
export function StatementManualDraft({ statement }: { statement: Statement }) {
  const { manual } = useStatementMutations(statement.id)
  const [proposal, setProposal] = useState<Proposal>(initial)
  const [target, setTarget] = useState('row:0:operationDate')
  const [reason, setReason] = useState('')
  const request = useRef<{ signature: string; key: string }>(undefined)
  const [error, setError] = useState('')
  const [regroup, setRegroup] = useState(false)
  if (!statement.layout || statement.publishedRows > 0) return null
  if (statement.batchId && !regroup)
    return (
      <Button variant="outline" onClick={() => setRegroup(true)}>
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
          : (proposal.service[0]?.spans ?? [])
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
        service: [
          { spans: ids, reason: 'Владелец проверил служебные заголовки и текст: ' + reason },
        ],
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
    { value: 'service', label: 'Служебные фрагменты вне строк и итогов' },
  )
  async function save() {
    try {
      const body = { expectedRevision: statement.revision, proposal, reason }
      const signature = JSON.stringify(body)
      if (request.current?.signature !== signature)
        request.current = { signature, key: crypto.randomUUID() }
      await manual.mutateAsync({ body, key: request.current.key })
      setError('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Черновик не сохранён.')
    }
  }
  return (
    <div className="space-y-4">
      <h3 className="font-medium">Подготовить строки вручную</h3>
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
      <Button disabled={!reason.trim() || manual.isPending} onClick={save}>
        Сохранить черновик без записи операций
      </Button>
    </div>
  )
}
