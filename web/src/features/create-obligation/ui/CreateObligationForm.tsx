import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { toast } from 'sonner'

import {
  CreateObligationSchema,
  type CreateObligationValues,
} from '@/features/create-obligation/model/create-obligation.schema'
import { previewSchedule } from '@/features/schedule-builder/api/preview-schedule'
import { rowsForPreview, type ScheduleRow } from '@/features/schedule-builder/model/schedule-rows'
import { ScheduleBuilder } from '@/features/schedule-builder/ui/ScheduleBuilder'
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
import { Field } from '@/shared/ui/field'
import { CurrencySelect } from '@/shared/ui/currency-select'
import { DEFAULT_CURRENCY } from '@/shared/lib/currencies'
import { Input } from '@/shared/ui/input'
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

export function CreateObligationForm() {
  const queryClient = useQueryClient()
  const [rows, setRows] = useState<ScheduleRow[]>([])
  const [scheduleError, setScheduleError] = useState('')
  const {
    control,
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<CreateObligationValues>({
    resolver: zodResolver(CreateObligationSchema),
    defaultValues: { name: '', kind: 'installment', currency: DEFAULT_CURRENCY },
  })
  const kind = useWatch({ control, name: 'kind' })
  const currency = useWatch({ control, name: 'currency' })

  const onSubmit = handleSubmit(async (values) => {
    setScheduleError('')
    if (rows.length === 0) {
      setScheduleError('Добавьте хотя бы один платёж.')
      return
    }
    // Go validates the rows and computes the total that the API must match.
    const checked = await previewSchedule({ mode: 'rows', payments: rowsForPreview(rows) })
    if (!checked.ok || checked.preview.rejectedLines.length > 0) {
      const lines = checked.ok ? checked.preview.rejectedLines : checked.rejectedLines
      setScheduleError(
        lines.length
          ? `Исправьте график: ${lines.map(line => `строка ${line.line} — ${line.reason}`).join('; ')}.`
          : checked.ok ? '' : checked.message,
      )
      return
    }
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
        totalMinor: checked.preview.totalMinor,
        payments: checked.preview.payments,
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
    reset({ name: '', kind: values.kind, currency: values.currency })
    setRows([])
    toast.success('Обязательство сохранено')
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Новое обязательство</CardTitle>
        <CardDescription>
          График планирует будущие платежи и сам по себе не меняет остатки.
          Итог считается по графику.
        </CardDescription>
      </CardHeader>
      <form onSubmit={onSubmit}>
        <CardContent className="space-y-6">
          <div className="grid gap-4 md:grid-cols-[2fr_1fr_1fr]">
            <Field label="Название" error={errors.name?.message}>
              <Input id="obligation-name" placeholder="Рассрочка на ноутбук" {...register('name')} />
            </Field>
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
            <Field label="Валюта" error={errors.currency?.message}>
              <CurrencySelect id="obligation-currency" value={currency} onValueChange={value => setValue('currency', value, { shouldValidate: true })} />
            </Field>
          </div>

          <fieldset className="space-y-3">
            <legend className="text-sm font-medium">График</legend>
            <ScheduleBuilder
              rows={rows}
              onChange={next => {
                setRows(next)
                setScheduleError('')
              }}
              currency={currency}
              idPrefix="obligation-schedule"
            />
            {scheduleError && (
              <p role="alert" className="text-sm text-destructive">{scheduleError}</p>
            )}
          </fieldset>
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
