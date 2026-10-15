import { useState } from 'react'
import { Eye, EyeOff, Pencil, Plus, Trash2 } from 'lucide-react'

import { operationsText } from '@/entities/category'
import {
  CategoryDeleteDialog,
  CategoryEditDialog,
  useToggleCategoryHidden,
} from '@/features/manage-category'
import type { Category } from '@/shared/api'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card'
import { CategoryIcon } from '@/shared/ui/category-icon'
import { Notice, StatusBadge } from '@/shared/ui/status'

// SPEC-0042: personal categories are edited and deleted (operations move to a
// chosen category); shared ones can be hidden from new-operation choices.
export function Categories({ items }: { items: Category[] }) {
  const toggle = useToggleCategoryHidden()
  const [editing, setEditing] = useState<Category | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Category | null>(null)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const expense = items.filter((c) => c.kind === 'expense')

  async function toggleHidden(category: Category) {
    if (busy) return
    setBusy(category.id)
    setError('')
    const result = await toggle.mutateAsync(category)
    if (!result.data) setError(result.error?.message ?? 'Не удалось изменить видимость категории.')
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
        <CategoryEditDialog
          category={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting && (
        <CategoryDeleteDialog
          category={deleting}
          targets={expense.filter((c) => c.id !== deleting.id)}
          onClose={() => setDeleting(null)}
        />
      )}
    </>
  )
}
