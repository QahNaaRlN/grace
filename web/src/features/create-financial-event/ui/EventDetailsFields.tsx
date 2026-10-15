import type { Category } from '@/shared/api'
import { Disclosure } from '@/shared/ui/disclosure'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'

// Optional details of an operation: the category and the counterparty.
export function EventDetailsFields({
  kind,
  category,
  onCategory,
  categories,
}: {
  kind: 'purchase' | 'income' | 'transfer' | 'refund' | 'debt_payment'
  category: string
  onCategory: (id: string) => void
  /** Categories offered for this kind; hidden ones are already left out. */
  categories: Category[]
}) {
  return (
    <Disclosure
      className="rounded-xl border p-3"
      triggerClassName="min-h-11 rounded-lg px-3 py-2 font-medium hover:bg-accent"
      title={<>Дополнительные сведения</>}
    >
      <div className="mt-4 space-y-4">
        {kind !== 'transfer' && kind !== 'refund' && (
          <div className="space-y-2">
            <Label htmlFor="event-category">Категория</Label>
            <Select value={category} onValueChange={onCategory}>
              <SelectTrigger id="event-category" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Не указана</SelectItem>
                {categories.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {kind !== 'transfer' && (
          <div className="space-y-2">
            <Label htmlFor="event-merchant">{kind === 'income' ? 'Источник' : 'Получатель'}</Label>
            <Input name="merchant" id="event-merchant" maxLength={160} />
          </div>
        )}
      </div>
    </Disclosure>
  )
}
