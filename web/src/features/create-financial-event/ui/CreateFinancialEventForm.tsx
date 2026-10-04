import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'

import {
  CreateFinancialEventSchema,
  type CreateFinancialEventValues,
} from '@/features/create-financial-event/model/create-financial-event.schema'
import { apiClient } from '@/shared/api/client'
import type { Account, Category } from '@/shared/api/types'
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
import { Label } from '@/shared/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'

type Props = {
  accounts: Account[]
  categories: Category[]
}

const kindLabels = {
  purchase: 'Покупка',
  income: 'Доход',
  transfer: 'Перевод между счетами',
  debt_payment: 'Платёж по обязательству',
} as const

export function CreateFinancialEventForm({ accounts, categories }: Props) {
  const queryClient = useQueryClient()
  const {
    control,
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<CreateFinancialEventValues>({
    resolver: zodResolver(CreateFinancialEventSchema),
    defaultValues: {
      kind: 'purchase',
      accountId: accounts[0]?.id ?? '',
      destinationAccountId: accounts[1]?.id ?? '',
      categoryId: categories.find((category) => category.kind === 'expense')?.id ?? '',
      amount: Number.NaN,
      merchant: '',
      description: '',
    },
  })
  const kind = useWatch({ control, name: 'kind' })
  const accountID = useWatch({ control, name: 'accountId' })
  const destinationAccountID = useWatch({ control, name: 'destinationAccountId' })
  const categoryID = useWatch({ control, name: 'categoryId' })
  const visibleCategories = categories.filter((category) =>
    kind === 'income' ? category.kind === 'income' : category.kind === 'expense',
  )

  const onSubmit = handleSubmit(async (values) => {
    const headers = { 'Idempotency-Key': createIdempotencyKey() }
    const occurredAt = new Date().toISOString()
    const amountMinor = Math.round(values.amount * 100)

    let error: { message: string } | undefined
    if (values.kind === 'purchase') {
      const result = await apiClient.POST('/api/v1/financial-events/purchases', {
        params: { header: headers },
        body: {
          accountId: values.accountId,
          categoryId: values.categoryId,
          amountMinor,
          merchant: values.merchant,
          description: values.description,
          occurredAt,
        },
      })
      error = result.error
    } else if (values.kind === 'income') {
      const result = await apiClient.POST('/api/v1/financial-events/incomes', {
        params: { header: headers },
        body: {
          accountId: values.accountId,
          categoryId: values.categoryId || null,
          amountMinor,
          merchant: values.merchant,
          description: values.description,
          occurredAt,
        },
      })
      error = result.error
    } else if (values.kind === 'debt_payment') {
      const result = await apiClient.POST('/api/v1/financial-events/debt-payments', {
        params: { header: headers },
        body: {
          accountId: values.accountId,
          categoryId: values.categoryId || null,
          amountMinor,
          merchant: values.merchant,
          description: values.description,
          occurredAt,
        },
      })
      error = result.error
    } else {
      const result = await apiClient.POST('/api/v1/financial-events/transfers', {
        params: { header: headers },
        body: {
          sourceAccountId: values.accountId,
          destinationAccountId: values.destinationAccountId,
          amountMinor,
          description: values.description,
          occurredAt,
        },
      })
      error = result.error
    }

    if (error) {
      toast.error(error.message)
      return
    }
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['accounts'] }),
      queryClient.invalidateQueries({ queryKey: ['financial-events'] }),
    ])
    reset({
      ...values,
      amount: Number.NaN,
      merchant: '',
      description: '',
    })
    toast.success(kindLabels[values.kind] + ' сохранена')
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Новое финансовое событие</CardTitle>
        <CardDescription>
          Остатки вычисляются из неизменяемых проводок, а не изменяются вручную.
        </CardDescription>
      </CardHeader>
      <form onSubmit={onSubmit}>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="grid gap-2 md:col-span-2">
            <Label htmlFor="kind">Вид события</Label>
            <Select value={kind} onValueChange={(value) => setValue('kind', value as CreateFinancialEventValues['kind'])}>
              <SelectTrigger id="kind" className="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                {Object.entries(kindLabels).map(([value, label]) => (
                  <SelectItem key={value} value={value}>{label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="account">{kind === 'transfer' ? 'Со счёта' : 'Счёт'}</Label>
            <Select value={accountID} onValueChange={(value) => setValue('accountId', value, { shouldValidate: true })}>
              <SelectTrigger id="account" className="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                {accounts.map((account) => <SelectItem key={account.id} value={account.id}>{account.name}</SelectItem>)}
              </SelectContent>
            </Select>
            {errors.accountId && <p className="text-sm text-destructive">{errors.accountId.message}</p>}
          </div>

          {kind === 'transfer' ? (
            <div className="grid gap-2">
              <Label htmlFor="destination-account">На счёт</Label>
              <Select value={destinationAccountID} onValueChange={(value) => setValue('destinationAccountId', value, { shouldValidate: true })}>
                <SelectTrigger id="destination-account" className="w-full"><SelectValue placeholder="Выберите счёт" /></SelectTrigger>
                <SelectContent>
                  {accounts.map((account) => <SelectItem key={account.id} value={account.id}>{account.name}</SelectItem>)}
                </SelectContent>
              </Select>
              {errors.destinationAccountId && <p className="text-sm text-destructive">{errors.destinationAccountId.message}</p>}
            </div>
          ) : (
            <div className="grid gap-2">
              <Label htmlFor="category">Категория</Label>
              <Select value={categoryID} onValueChange={(value) => setValue('categoryId', value, { shouldValidate: true })}>
                <SelectTrigger id="category" className="w-full"><SelectValue placeholder="Без категории" /></SelectTrigger>
                <SelectContent>
                  {visibleCategories.map((category) => <SelectItem key={category.id} value={category.id}>{category.name}</SelectItem>)}
                </SelectContent>
              </Select>
              {errors.categoryId && <p className="text-sm text-destructive">{errors.categoryId.message}</p>}
            </div>
          )}

          <div className="grid gap-2">
            <Label htmlFor="amount">Сумма, ₽</Label>
            <Input id="amount" type="number" min="0.01" step="0.01" inputMode="decimal" {...register('amount', { valueAsNumber: true })} />
            {errors.amount && <p className="text-sm text-destructive">{errors.amount.message}</p>}
          </div>

          {kind !== 'transfer' && (
            <div className="grid gap-2">
              <Label htmlFor="merchant">{kind === 'income' ? 'Источник' : 'Получатель'}</Label>
              <Input id="merchant" {...register('merchant')} />
              {errors.merchant && <p className="text-sm text-destructive">{errors.merchant.message}</p>}
            </div>
          )}

          <div className="grid gap-2 md:col-span-2">
            <Label htmlFor="description">Комментарий</Label>
            <Input id="description" {...register('description')} />
          </div>
        </CardContent>
        <CardFooter>
          <Button type="submit" disabled={isSubmitting || accounts.length === 0}>
            {isSubmitting ? 'Сохранение…' : 'Сохранить событие'}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}
