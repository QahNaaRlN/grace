import { useState } from 'react'

import { operationsText } from '@/entities/category'
import type { Category } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { useDeleteCategory } from '../api/category-mutations'

export function CategoryDeleteDialog({
  category,
  targets,
  onClose,
}: {
  category: Category
  targets: Category[]
  onClose: () => void
}) {
  const [target, setTarget] = useState('')
  const [pending, setPending] = useState(false)
  const deleteCategory = useDeleteCategory(category)
  const [error, setError] = useState('')
  const count = category.operationCount
  const chosen = targets.find((t) => t.id === target)
  async function remove() {
    if (pending) return
    setPending(true)
    setError('')
    try {
      await deleteCategory.mutateAsync(target)
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
