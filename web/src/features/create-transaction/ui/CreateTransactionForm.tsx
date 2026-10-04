import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'

import {
  CreateTransactionFormSchema,
  type CreateTransactionFormValues,
} from '@/features/create-transaction/model/create-transaction.schema'
import { apiClient } from '@/shared/api/client'
import type { Account, Category } from '@/shared/api/types'
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

export function CreateTransactionForm({ accounts, categories }: Props) {
  const queryClient = useQueryClient()
  const expenseCategories = categories.filter((category) => category.kind === 'expense')
  const {
    register,
    handleSubmit,
    reset,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<CreateTransactionFormValues>({
    resolver: zodResolver(CreateTransactionFormSchema),
    defaultValues: {
      accountId: accounts[0]?.id ?? '',
      categoryId: expenseCategories[0]?.id ?? '',
      amount: Number.NaN,
      merchant: '',
      description: '',
    },
  })
  const accountId = useWatch({ control, name: 'accountId' })
  const categoryId = useWatch({ control, name: 'categoryId' })

  const onSubmit = handleSubmit(async (values) => {
    const { data, error } = await apiClient.POST('/api/v1/transactions', {
      body: {
        accountId: values.accountId,
        categoryId: values.categoryId,
        amountMinor: Math.round(values.amount * 100),
        direction: 'expense',
        kind: 'purchase',
        source: 'manual',
        merchant: values.merchant,
        description: values.description,
        occurredAt: new Date().toISOString(),
      },
    })
    if (error || !data) {
      toast.error(error?.message ?? 'Не удалось сохранить операцию')
      return
    }
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['accounts'] }),
      queryClient.invalidateQueries({ queryKey: ['transactions'] }),
    ])
    reset({
      accountId: values.accountId,
      categoryId: values.categoryId,
      amount: Number.NaN,
      merchant: '',
      description: '',
    })
    toast.success('Покупка сохранена')
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Добавить покупку</CardTitle>
        <CardDescription>
          Сумма будет сохранена в копейках, а остаток счёта изменится атомарно.
        </CardDescription>
      </CardHeader>
      <form onSubmit={onSubmit}>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor="account">Счёт</Label>
            <Select
              value={accountId}
              onValueChange={(value) => setValue('accountId', value, { shouldValidate: true })}
            >
              <SelectTrigger id="account" className="w-full">
                <SelectValue placeholder="Выберите счёт" />
              </SelectTrigger>
              <SelectContent>
                {accounts.map((account) => (
                  <SelectItem key={account.id} value={account.id}>{account.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.accountId && <p className="text-sm text-destructive">{errors.accountId.message}</p>}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="category">Категория</Label>
            <Select
              value={categoryId}
              onValueChange={(value) => setValue('categoryId', value, { shouldValidate: true })}
            >
              <SelectTrigger id="category" className="w-full">
                <SelectValue placeholder="Выберите категорию" />
              </SelectTrigger>
              <SelectContent>
                {expenseCategories.map((category) => (
                  <SelectItem key={category.id} value={category.id}>{category.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            {errors.categoryId && <p className="text-sm text-destructive">{errors.categoryId.message}</p>}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="amount">Сумма, ₽</Label>
            <Input
              id="amount"
              type="number"
              min="0.01"
              step="0.01"
              inputMode="decimal"
              {...register('amount', { valueAsNumber: true })}
            />
            {errors.amount && <p className="text-sm text-destructive">{errors.amount.message}</p>}
          </div>
          <div className="grid gap-2">
            <Label htmlFor="merchant">Получатель</Label>
            <Input id="merchant" placeholder="Например, Пятёрочка" {...register('merchant')} />
            {errors.merchant && <p className="text-sm text-destructive">{errors.merchant.message}</p>}
          </div>
          <div className="grid gap-2 md:col-span-2">
            <Label htmlFor="description">Комментарий</Label>
            <Input id="description" {...register('description')} />
          </div>
        </CardContent>
        <CardFooter>
          <Button type="submit" disabled={isSubmitting || accounts.length === 0}>
            {isSubmitting ? 'Сохранение…' : 'Сохранить покупку'}
          </Button>
        </CardFooter>
      </form>
    </Card>
  )
}
