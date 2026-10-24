import { useState } from 'react'

import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Textarea } from '@/shared/ui/textarea'
import { useStatementMutations } from '../api/mutations'

type Statement = components['schemas']['BankStatement']
export function StatementOwnerReview({ statement }: { statement: Statement }) {
  const { edit, remove } = useStatementMutations(statement.id)
  const [retention, setRetention] = useState(statement.retention)
  const [roles, setRoles] = useState(statement.rolesConfirmed)
  const [pages, setPages] = useState(statement.reviewedPages)
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [deleteRequested, setDeleteRequested] = useState(false)
  const [editingRevision, setEditingRevision] = useState(statement.revision)
  const stale = editingRevision !== statement.revision
  function loadReview(source: Statement) {
    setRetention(source.retention)
    setRoles(source.rolesConfirmed)
    setPages([...source.reviewedPages])
    setEditingRevision(source.revision)
    setReason('')
    setError('')
  }
  async function save() {
    try {
      if (stale) return
      const saved = await edit.mutateAsync({
        expectedRevision: editingRevision,
        accountId: statement.accountId,
        retention,
        rolesConfirmed: roles,
        reviewedPages: pages,
        reason,
      })
      loadReview(saved)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Проверка не сохранена.')
    }
  }
  if (statement.deletedAt)
    return (
      <p className="text-sm">
        Оригинал удалён. Значения, фрагменты полей и история решений сохранены. Прежние резервные
        копии исчезнут по сроку их хранения.
      </p>
    )
  return (
    <div className="space-y-3">
      {stale && (
        <section className="space-y-3 rounded-xl border p-3">
          <p role="status">
            Источник изменился. Отметки и основание ниже относятся к предыдущей версии и не
            подтверждают текущий черновик. Ввод сохранён. Загрузите актуальную проверку и заново
            проверьте роли и страницы.
          </p>
          <Button variant="outline" onClick={() => loadReview(statement)} disabled={edit.isPending}>
            Загрузить актуальную проверку
          </Button>
        </section>
      )}
      <p className="text-sm">
        Срок проверки: {new Date(statement.expiresAt).toLocaleString('ru-RU')}. Резервные копии
        могут хранить оригинал до окончания их срока хранения.
      </p>
      <Field label="Хранение оригинала">
        <ChoiceSelect
          value={retention}
          onValueChange={(v) => setRetention(v as Statement['retention'])}
        >
          {[
            {
              value: 'delete_after_confirmation',
              label: 'Удалить после всех строк или через 7 дней',
            },
            { value: 'keep', label: 'Сохранить оригинал' },
          ].map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </ChoiceSelect>
      </Field>
      {statement.readingMode !== 'image_review' && (
        <>
          {' '}
          <label className="flex min-h-11 items-center gap-3">
            <Checkbox checked={roles} onCheckedChange={(v) => setRoles(v === true)} />
            Проверены роли дат, валютных колонок, оборотов и остатков
          </label>
          <fieldset>
            <legend>Проверено покрытие страниц</legend>
            <div className="flex flex-wrap gap-3">
              {Array.from({ length: statement.pageCount }, (_, i) => i + 1).map((page) => (
                <label key={page} className="flex min-h-11 items-center gap-2">
                  <Checkbox
                    checked={pages.includes(page)}
                    onCheckedChange={(v) =>
                      setPages(v ? [...pages, page] : pages.filter((p) => p !== page))
                    }
                  />
                  Страница {page}
                </label>
              ))}
            </div>
          </fieldset>
        </>
      )}
      <Field label="Основание проверки">
        <Textarea value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      <Button onClick={save} disabled={stale || !reason.trim() || edit.isPending}>
        Сохранить проверку
      </Button>
      <div>
        <Button variant="outline" onClick={() => setDeleteRequested(true)}>
          Удалить оригинал
        </Button>
      </div>
      {deleteRequested && (
        <div className="space-y-2">
          <p>
            Удалить исходник, изображения страниц и полный извлечённый текст? Фрагменты
            использованных полей и финансовая история сохранятся.
          </p>
          <Button
            variant="destructive"
            disabled={remove.isPending}
            onClick={async () => {
              try {
                await remove.mutateAsync(statement.revision)
              } catch (e) {
                setError(e instanceof Error ? e.message : 'Удаление не выполнено.')
              }
            }}
          >
            Подтвердить удаление оригинала
          </Button>
          <Button variant="ghost" onClick={() => setDeleteRequested(false)}>
            Отмена
          </Button>
        </div>
      )}
    </div>
  )
}
