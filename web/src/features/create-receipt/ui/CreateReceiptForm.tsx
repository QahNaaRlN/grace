import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { Plus, ReceiptText, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'

import {
  CreateReceiptSchema,
  type CreateReceiptValues,
} from '@/features/create-receipt/model/create-receipt.schema'
import { receiptQueries } from '@/entities/receipt'
import { apiClient } from '@/shared/api'
import type { FinancialEvent } from '@/shared/api'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { parseMilli, parseMinor } from '@/shared/lib/money'

function emptyItem() {
  return { name: '', quantity: '1', unitPrice: '', lineTotal: '' }
}

function localDateTime(value: string | null) {
  if (!value) return ''
  const date = new Date(value)
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16)
}

export function CreateReceiptForm({ purchase }: { purchase: FinancialEvent }) {
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const {
    control,
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateReceiptValues>({
    resolver: zodResolver(CreateReceiptSchema),
    defaultValues: {
      seller: purchase.merchant,
      issuedAt: localDateTime(purchase.occurredAt),
      fiscalDocumentNumber: '',
      items: [emptyItem()],
    },
  })
  const { fields, append, remove } = useFieldArray({ control, name: 'items' })
  const watchedItems = useWatch({ control, name: 'items' })
  const enteredTotalMinor = watchedItems.reduce(
    (sum, item) => sum + (parseMinor(item.lineTotal ?? '') ?? 0),
    0,
  )
  const purchasePosting = purchase.postings.find((posting) => posting.direction === 'debit')
  const currency = purchasePosting?.currency ?? 'RUB'
  const purchaseAmountMinor = purchasePosting?.amountMinor ?? 0
  const formatMoney = (amountMinor: number) =>
    new Intl.NumberFormat('ru-RU', {
      style: 'currency',
      currency,
    }).format(amountMinor / 100)

  const onSubmit = handleSubmit(async (values) => {
    const result = await apiClient.POST('/api/v1/receipts', {
      params: { header: { 'Idempotency-Key': createIdempotencyKey() } },
      body: {
        purchaseEventId: purchase.id,
        seller: values.seller,
        issuedAt: new Date(values.issuedAt).toISOString(),
        fiscalDocumentNumber: values.fiscalDocumentNumber,
        items: values.items.map((item) => ({
          name: item.name,
          quantityMilli: parseMilli(item.quantity) as number,
          unitPriceMinor: parseMinor(item.unitPrice) as number,
          lineTotalMinor: parseMinor(item.lineTotal) as number,
        })),
      },
    })
    if (result.error) {
      toast.error(result.error.message)
      return
    }
    await queryClient.invalidateQueries({ queryKey: receiptQueries.all() })
    reset()
    setOpen(false)
    toast.success('Чек сохранён')
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <ReceiptText aria-hidden="true" />
          Добавить состав
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Состав покупки</DialogTitle>
          <DialogDescription>
            Позиции должны дать точную сумму покупки {formatMoney(purchaseAmountMinor)}. Чек не
            изменяет остаток повторно.
          </DialogDescription>
        </DialogHeader>
        <form className="space-y-4" onSubmit={onSubmit}>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor={`seller-${purchase.id}`}>Продавец</Label>
              <Input id={`seller-${purchase.id}`} {...register('seller')} />
              {errors.seller && <p className="text-sm text-destructive">{errors.seller.message}</p>}
            </div>
            <div className="grid gap-2">
              <Label htmlFor={`issued-${purchase.id}`}>Дата чека</Label>
              <Input id={`issued-${purchase.id}`} type="datetime-local" {...register('issuedAt')} />
              {errors.issuedAt && (
                <p className="text-sm text-destructive">{errors.issuedAt.message}</p>
              )}
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor={`fiscal-${purchase.id}`}>Номер фискального документа</Label>
            <Input id={`fiscal-${purchase.id}`} {...register('fiscalDocumentNumber')} />
          </div>

          <div className="space-y-3">
            {fields.map((field, index) => (
              <div
                key={field.id}
                className="grid gap-3 rounded-lg border p-3 sm:grid-cols-[2fr_1fr_1fr_1fr_auto]"
              >
                <div className="grid gap-2">
                  <Label htmlFor={`item-name-${field.id}`}>Позиция</Label>
                  <Input id={`item-name-${field.id}`} {...register(`items.${index}.name`)} />
                  {errors.items?.[index]?.name && (
                    <p className="text-xs text-destructive">{errors.items[index].name.message}</p>
                  )}
                </div>
                <div className="grid gap-2">
                  <Label htmlFor={`item-quantity-${field.id}`}>Количество</Label>
                  <Input
                    id={`item-quantity-${field.id}`}
                    inputMode="decimal"
                    className="tabular-nums"
                    aria-invalid={errors.items?.[index]?.quantity ? true : undefined}
                    {...register(`items.${index}.quantity`)}
                  />
                  {errors.items?.[index]?.quantity && (
                    <p className="text-xs text-destructive">
                      {errors.items[index].quantity.message}
                    </p>
                  )}
                </div>
                <div className="grid gap-2">
                  <Label htmlFor={`item-price-${field.id}`}>Цена</Label>
                  <MoneyInput
                    id={`item-price-${field.id}`}
                    currency={currency}
                    aria-invalid={errors.items?.[index]?.unitPrice ? true : undefined}
                    {...register(`items.${index}.unitPrice`)}
                  />
                  {errors.items?.[index]?.unitPrice && (
                    <p className="text-xs text-destructive">
                      {errors.items[index].unitPrice.message}
                    </p>
                  )}
                </div>
                <div className="grid gap-2">
                  <Label htmlFor={`item-total-${field.id}`}>Итого</Label>
                  <MoneyInput
                    id={`item-total-${field.id}`}
                    currency={currency}
                    aria-invalid={errors.items?.[index]?.lineTotal ? true : undefined}
                    {...register(`items.${index}.lineTotal`)}
                  />
                  {errors.items?.[index]?.lineTotal && (
                    <p className="text-xs text-destructive">
                      {errors.items[index].lineTotal.message}
                    </p>
                  )}
                </div>
                <Button
                  className="self-end"
                  type="button"
                  variant="ghost"
                  size="icon"
                  disabled={fields.length === 1}
                  onClick={() => remove(index)}
                >
                  <Trash2 aria-hidden="true" />
                  <span className="sr-only">Удалить позицию</span>
                </Button>
              </div>
            ))}
          </div>

          <Button type="button" variant="outline" onClick={() => append(emptyItem())}>
            <Plus aria-hidden="true" />
            Добавить позицию
          </Button>

          <div className="flex flex-wrap justify-between gap-2 rounded-lg bg-muted p-3 text-sm">
            <span>
              Сумма позиций: <strong>{formatMoney(enteredTotalMinor)}</strong>
            </span>
            <span>
              Сумма покупки: <strong>{formatMoney(purchaseAmountMinor)}</strong>
            </span>
          </div>

          <DialogFooter>
            <Button className="w-full sm:w-auto" type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Сохранение…' : 'Сохранить чек'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
