import { useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Eye, EyeOff, Pencil, Plus, Trash2 } from 'lucide-react'
import { statisticsQueries } from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import type { Category } from '@/shared/api'
import { apiClient } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import { CategoryIcon } from '@/shared/ui/category-icon'
import { categoryIcons, type CategoryIconName } from '@/shared/lib/category-icons'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/shared/ui/card'
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogHeader,
} from '@/shared/ui/dialog'
import { Notice, StatusBadge } from '@/shared/ui/status'
import { cn } from '@/shared/lib/utils'

function operationsText(count: number) {
  const tail = count % 100
  const word =
    tail >= 11 && tail <= 14
      ? 'операций'
      : count % 10 === 1
        ? 'операция'
        : count % 10 >= 2 && count % 10 <= 4
          ? 'операции'
          : 'операций'
  return `${count} ${word}`
}

// SPEC-0042: personal categories are edited and deleted (operations move to a
// chosen category); shared ones can be hidden from new-operation choices.
export function Categories({ items }: { items: Category[] }) {
  const cache = useQueryClient()
  const [editing, setEditing] = useState<Category | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Category | null>(null)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const expense = items.filter((c) => c.kind === 'expense')
  const refresh = () =>
    Promise.all(
      [categoryQueries.all(), financialEventQueries.all(), statisticsQueries.all()].map(
        (queryKey) => cache.invalidateQueries({ queryKey }),
      ),
    )

  async function toggleHidden(category: Category) {
    if (busy) return
    setBusy(category.id)
    setError('')
    const params = { path: { categoryId: category.id } }
    const result = category.hidden
      ? await apiClient.DELETE('/api/v1/categories/{categoryId}/hidden', { params })
      : await apiClient.PUT('/api/v1/categories/{categoryId}/hidden', { params })
    if (!result.data) setError(result.error?.message ?? 'Не удалось изменить видимость категории.')
    else await refresh()
    setBusy('')
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Категории расходов</CardTitle>
          <CardDescription>
            Свои категории можно изменить и удалить; общие доступны всем — их можно скрыть из выбора
            для новых операций.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Button onClick={() => setEditing('new')}>
            <Plus />
            Добавить категорию
          </Button>
          {error && (
            <Notice tone="danger" title="Не получилось">
              {error}
            </Notice>
          )}
          <div className="space-y-2">
            {expense.map((c) => (
              <div
                key={c.id}
                className={cn(
                  'flex flex-wrap items-center gap-3 rounded-xl border p-3',
                  c.hidden && 'bg-muted/50',
                )}
              >
                <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-secondary text-secondary-foreground">
                  <CategoryIcon icon={c.icon} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="break-words font-medium">{c.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {c.editable ? 'Ваша категория' : 'Общая категория'} ·{' '}
                    {operationsText(c.operationCount)}
                  </p>
                </div>
                {c.hidden && <StatusBadge tone="unknown">Скрыта из выбора</StatusBadge>}
                {c.editable ? (
                  <div className="flex gap-1">
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Изменить ${c.name}`}
                      onClick={() => setEditing(c)}
                    >
                      <Pencil />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Удалить ${c.name}`}
                      onClick={() => setDeleting(c)}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                ) : (
                  <Button
                    variant="ghost"
                    disabled={busy === c.id}
                    onClick={() => void toggleHidden(c)}
                  >
                    {c.hidden ? <Eye /> : <EyeOff />}
                    {c.hidden ? 'Показать' : 'Скрыть'}
                  </Button>
                )}
              </div>
            ))}
          </div>
          <p className="text-sm text-muted-foreground">
            Изменение и удаление сохраняют суммы: операции удалённой категории переносятся в
            выбранную.
          </p>
        </CardContent>
      </Card>
      {editing !== null && (
        <EditDialog
          category={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={refresh}
        />
      )}
      {deleting && (
        <DeleteDialog
          category={deleting}
          targets={expense.filter((c) => c.id !== deleting.id)}
          onClose={() => setDeleting(null)}
          onDeleted={refresh}
        />
      )}
    </>
  )
}

function EditDialog({
  category,
  onClose,
  onSaved,
}: {
  category: Category | null
  onClose: () => void
  onSaved: () => Promise<unknown>
}) {
  const [name, setName] = useState(category?.name ?? '')
  const [icon, setIcon] = useState<CategoryIconName | null>(category?.icon ?? null)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  async function save(e: FormEvent) {
    e.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    try {
      const result =
        category === null
          ? await apiClient.POST('/api/v1/categories', {
              body: { name: name.trim(), ...(icon ? { icon } : {}) },
            })
          : await apiClient.PATCH('/api/v1/categories/{categoryId}', {
              params: { path: { categoryId: category.id } },
              body: { name: name.trim(), icon: icon ?? '' },
            })
      if (!result.data) throw new Error(result.error?.message ?? 'Не удалось сохранить категорию.')
      await onSaved()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось сохранить категорию.')
    } finally {
      setPending(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{category === null ? 'Новая категория' : 'Изменить категорию'}</DialogTitle>
          <DialogDescription>
            Название и иконка помогут найти покупки в операциях и статистике.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className="space-y-4">
          <Field label="Название категории">
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              maxLength={80}
              placeholder="Например, Спорт"
            />
          </Field>
          <fieldset className="space-y-2">
            <legend className="text-sm font-medium">Иконка</legend>
            <div
              role="radiogroup"
              aria-label="Иконка категории"
              className="grid grid-cols-6 gap-1 sm:grid-cols-9"
            >
              {(Object.keys(categoryIcons) as CategoryIconName[]).map((key) => (
                <button
                  key={key}
                  type="button"
                  role="radio"
                  aria-checked={icon === key}
                  aria-label={categoryIcons[key].label}
                  title={categoryIcons[key].label}
                  onClick={() => setIcon(icon === key ? null : key)}
                  className={cn(
                    'grid size-11 place-items-center rounded-xl border border-transparent text-muted-foreground hover:bg-accent',
                    icon === key && 'border-primary bg-selected text-selected-foreground',
                  )}
                >
                  <CategoryIcon icon={key} />
                </button>
              ))}
            </div>
          </fieldset>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          <Button type="submit" disabled={pending || !name.trim()}>
            {pending ? 'Сохраняем…' : 'Сохранить'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function DeleteDialog({
  category,
  targets,
  onClose,
  onDeleted,
}: {
  category: Category
  targets: Category[]
  onClose: () => void
  onDeleted: () => Promise<unknown>
}) {
  const [target, setTarget] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const count = category.operationCount
  const chosen = targets.find((t) => t.id === target)
  async function remove() {
    if (pending) return
    setPending(true)
    setError('')
    try {
      // The preview count is checked again by Go inside the deletion.
      const preview = await apiClient.POST('/api/v1/categories/{categoryId}/delete-preview', {
        params: { path: { categoryId: category.id } },
      })
      if (!preview.data) throw new Error(preview.error?.message ?? 'Категория недоступна.')
      if (preview.data.operations !== count)
        throw new Error('Количество операций изменилось. Обновите страницу.')
      const result = await apiClient.DELETE('/api/v1/categories/{categoryId}', {
        params: {
          path: { categoryId: category.id },
          query: { expectedOperations: count, ...(target ? { moveTo: target } : {}) },
        },
      })
      if (!result.data) throw new Error(result.error?.message ?? 'Не удалось удалить категорию.')
      await onDeleted()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось удалить категорию.')
    } finally {
      setPending(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Удалить «{category.name}»?</DialogTitle>
          <DialogDescription>
            {count === 0
              ? 'В категории нет операций.'
              : `В категории ${operationsText(count)}. Покупка не может остаться без категории — выберите, куда их перенести.`}
          </DialogDescription>
        </DialogHeader>
        {count > 0 && (
          <Field label="Перенести операции в">
            <ChoiceSelect value={target} onValueChange={setTarget}>
              <option value="">Выберите категорию</option>
              {targets.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </ChoiceSelect>
          </Field>
        )}
        {count > 0 && chosen && (
          <p className="text-sm">
            Перенесём {operationsText(count)} в «{chosen.name}». Суммы и даты не изменятся.
          </p>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={pending}>
            Отмена
          </Button>
          <Button
            variant="destructive-solid"
            disabled={pending || (count > 0 && !target)}
            onClick={() => void remove()}
          >
            {pending ? 'Удаляем…' : 'Удалить категорию'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
