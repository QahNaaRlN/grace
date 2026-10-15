import { useState, type FormEvent } from 'react'

import type { Category } from '@/shared/api'
import { categoryIcons, type CategoryIconName } from '@/shared/lib/category-icons'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { CategoryIcon } from '@/shared/ui/category-icon'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { useSaveCategory } from '../api/category-mutations'

export function CategoryEditDialog({
  category,
  onClose,
}: {
  category: Category | null
  onClose: () => void
}) {
  const [name, setName] = useState(category?.name ?? '')
  const [icon, setIcon] = useState<CategoryIconName | null>(category?.icon ?? null)
  const [pending, setPending] = useState(false)
  const saveCategory = useSaveCategory(category)
  const [error, setError] = useState('')
  async function save(e: FormEvent) {
    e.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    try {
      await saveCategory.mutateAsync({ name: name.trim(), icon })
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
