import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'

import {
  CreateObligationSchema,
  type CreateObligationValues,
} from '@/features/create-obligation/model/create-obligation.schema'
import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { Button } from '@/shared/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { parseMinor } from '@/shared/lib/finance-display'
import { Label } from '@/shared/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'

const kindLabels = {
  loan: 'Кредит',
  installment: 'Рассрочка',
  other_fixed: 'Другое фиксированное',
} as const

function emptyPayment() {
  return { dueOn: '', amount: '' }
}

export function CreateObligationForm() {
  const queryClient = useQueryClient()
  const {
    control,
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<CreateObligationValues>({
    resolver: zodResolver(CreateObligationSchema),
    defaultValues: {
      name: '',
      kind: 'installment',
      currency: 'RUB',
      total: '',
      payments: [emptyPayment()],
    },
  })
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'payments',
  })
  const kind = useWatch({ control, name: 'kind' })
  const currency = useWatch({ control, name: 'currency' })

  const onSubmit = handleSubmit(async (values) => {
    const result = await apiClient.POST('/api/v1/obligations', {
      params: {
        header: {
          'Idempotency-Key': createIdempotencyKey(),
          'X-CSRF-Token': csrfToken,
        },
      },
      body: {
        name: values.name,
        kind: values.kind,
        currency: values.currency.toUpperCase(),
        totalMinor: parseMinor(values.total) as number,
        payments: values.payments.map((payment) => ({
          dueOn: payment.dueOn,
          amountMinor: parseMinor(payment.amount) as number,
        })),
      },
    })
    if (result.error) {
      toast.error(result.error.message)
      return
    }
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['obligations'] }),
      queryClient.invalidateQueries({ queryKey: ['cash-position'] }),
    ])
    reset({
      name: '',
      kind: values.kind,
      currency: values.currency,
      total: '',
      payments: [emptyPayment()],
    })
    toast.success('Обязательство сохранено')
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Новое обязательство</CardTitle>
        <CardDescription>
          График планирует будущие платежи и сам по себе не меняет остатки.
        </CardDescription>
      </CardHeader>
      <form onSubmit={onSubmit}>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="obligation-name">Название</Label>
              <Input
                id="obligation-name"
                placeholder="Рассрочка на ноутбук"
                {...register('name')}
              />
              {errors.name && (
                <p className="text-sm text-destructive">
                  {errors.name.message}
                </p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="obligation-kind">Тип</Label>
              <Select
                value={kind}
                onValueChange={(value) =>
                  setValue('kind', value as CreateObligationValues['kind'])
                }
              >
                <SelectTrigger id="obligation-kind" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(kindLabels).map(([value, label]) => (
                    <SelectItem key={value} value={value}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="obligation-total">Общая сумма</Label>
              <MoneyInput
                id="obligation-total"
                currency={currency}
                aria-invalid={errors.total ? true : undefined}
                {...register('total')}
              />
              {errors.total && (
                <p className="text-sm text-destructive">
                  {errors.total.message}
                </p>
              )}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="obligation-currency">Валюта</Label>
              <Input
                id="obligation-currency"
                maxLength={3}
                {...register('currency')}
              />
              {errors.currency && (
                <p className="text-sm text-destructive">
                  {errors.currency.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <Label>График</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => append(emptyPayment())}
              >
                <Plus aria-hidden="true" className="mr-1 size-4" /> Добавить
                платёж
              </Button>
            </div>
            {fields.map((field, index) => (
              <div
                key={field.id}
                className="grid gap-3 rounded-lg border p-3 sm:grid-cols-[1fr_1fr_auto]"
              >
                <div className="grid gap-2">
                  <Label htmlFor={`payment-date-${index}`}>
                    Дата #{index + 1}
                  </Label>
                  <Input
                    id={`payment-date-${index}`}
                    type="date"
                    {...register(`payments.${index}.dueOn`)}
                  />
                  {errors.payments?.[index]?.dueOn && (
                    <p className="text-sm text-destructive">
                      {errors.payments[index].dueOn.message}
                    </p>
                  )}
                </div>
                <div className="grid gap-2">
                  <Label htmlFor={`payment-amount-${index}`}>Сумма</Label>
                  <MoneyInput
                    id={`payment-amount-${index}`}
                    currency={currency}
                    aria-invalid={errors.payments?.[index]?.amount ? true : undefined}
                    {...register(`payments.${index}.amount`)}
                  />
                  {errors.payments?.[index]?.amount && (
                    <p className="text-sm text-destructive">
                      {errors.payments[index].amount.message}
                    </p>
                  )}
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="self-end"
                  disabled={fields.length === 1}
                  onClick={() => remove(index)}
                  aria-label={`Удалить платёж ${index + 1}`}
                >
                  <Trash2 aria-hidden="true" className="size-4" />
                </Button>
              </div>
            ))}
            {errors.payments?.message && (
              <p className="text-sm text-destructive">
                {errors.payments.message}
              </p>
            )}
          </div>
        </CardContent>
        <CardFooter>
          <Button
            className="w-full sm:w-auto"
            type="submit"
            disabled={isSubmitting}
          >
            {isSubmitting ? 'Сохранение…' : 'Сохранить обязательство'}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}
