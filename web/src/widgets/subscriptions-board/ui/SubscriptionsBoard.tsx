import { useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CalendarClock, Pause, Pencil, Plus, Repeat2 } from 'lucide-react'
import { toast } from 'sonner'

import { accountQueries } from '@/entities/account'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import {
  subscriptionExpenseQueries,
  subscriptionPeriodLabels,
  subscriptionQueries,
} from '@/entities/subscription'
import { CreateFinancialEventForm } from '@/features/create-financial-event'
import {
  SubscriptionForm,
  useLinkSubscriptionExpense,
  useToggleSubscriptionPause,
} from '@/features/manage-subscription'
import type { components } from '@/shared/api/generated/schema'
import { dateText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Money } from '@/shared/ui/money'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { subscriptionCard } from '../lib/subscription-card'
import { ExpenseHistory, SubscriptionCategory } from './subscription-details'

type Subscription = components['schemas']['Subscription']
/**
 * Subscriptions with their expected charges, cards and linked purchases. `reminders` is
 * composed by the page (the payment reminder history widget).
 */
export function SubscriptionsBoard({ reminders }: { reminders?: ReactNode }) {
  const [edit, setEdit] = useState<Subscription | 'new' | null>(null)
  const [expense, setExpense] = useState<Subscription | null>(null)
  const [purchaseOpen, setPurchaseOpen] = useState(false)
  const [eventID, setEventID] = useState('')
  const subscriptions = useQuery(subscriptionQueries.list())
  const accounts = useQuery(accountQueries.list())
  const events = useQuery(financialEventQueries.subscriptionPicker())
  const expenseHistory = useQuery({
    ...subscriptionExpenseQueries.list(expense?.id),
    enabled: expense !== null,
  })
  const categories = useQuery(categoryQueries.list())
  const pause = useToggleSubscriptionPause({ onError: (e) => toast.error(e.message) })
  const link = useLinkSubscriptionExpense({
    onSuccess: () => {
      setExpense(null)
      setEventID('')
      toast.success('Расход связан с подпиской')
    },
    onError: (e) => toast.error(e.message),
  })
  const used = new Set(subscriptions.data?.flatMap((s) => s.expenseIds) ?? [])
  const candidates =
    events.data?.filter(
      (e) =>
        e.kind === 'purchase' &&
        !used.has(e.id) &&
        e.postings.some((p) => p.currency === expense?.currency),
    ) ?? []
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1>Подписки</h1>
          <p className="mt-2 text-muted-foreground">
            Регулярные списания, карты и фактические расходы.
          </p>
        </div>
        <Button onClick={() => setEdit('new')}>
          <Plus className="size-4" />
          Добавить подписку
        </Button>
      </div>
      <p className="text-sm text-muted-foreground">
        Планируемые списания не изменяют остатки. Подтверждённые покупки учитываются в статистике
        один раз. Напоминания приходят за 3 и 1 день на подтверждённую почту.
      </p>
      {subscriptions.isPending ? (
        <LoadingSkeleton label="Загружаем подписки…" />
      ) : subscriptions.isError ? (
        <Card>
          <CardContent className="space-y-4 pt-5">
            <p role="alert">{subscriptions.error.message}</p>
            <Button variant="outline" onClick={() => void subscriptions.refetch()}>
              Повторить
            </Button>
          </CardContent>
        </Card>
      ) : !subscriptions.data?.length ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Repeat2 className="mx-auto mb-4 size-10 text-primary" />
            <h2>Все регулярные расходы в одном месте</h2>
            <p className="mt-2 text-muted-foreground">
              Добавьте связь, сервис или другую подписку.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {subscriptions.data.map((s) => (
            <Card key={s.id}>
              <CardContent className="space-y-4 pt-5">
                <div className="flex justify-between gap-3">
                  <div>
                    <h2>{s.name}</h2>
                    <p className="mt-1">
                      <Money valueMinor={s.amountMinor} currency={s.currency} size="lg" />
                    </p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {subscriptionPeriodLabels[s.period]}
                    </p>
                    <SubscriptionCategory
                      category={categories.data?.find((c) => c.id === s.categoryId)}
                    />
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Редактировать ${s.name}`}
                    onClick={() => setEdit(s)}
                  >
                    <Pencil className="size-4" />
                  </Button>
                </div>
                <p className="flex items-center gap-2 text-sm">
                  <CalendarClock className="size-4 text-primary" />
                  {s.paused
                    ? 'Приостановлена'
                    : s.nextDueOn
                      ? `Следующее списание: ${dateText(s.nextDueOn)}`
                      : 'Период завершён'}
                </p>
                <p className="text-sm text-muted-foreground">
                  {subscriptionCard(s, accounts.data ?? [])} ·{' '}
                  {s.reminders ? 'Напоминания включены' : 'Без напоминаний'}
                </p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    onClick={() => {
                      setExpense(s)
                      setEventID('')
                    }}
                  >
                    Расходы · {s.expenseIds.length}
                  </Button>
                  <Button
                    variant="ghost"
                    disabled={pause.isPending}
                    onClick={() => pause.mutate(s)}
                  >
                    {s.paused ? <Repeat2 className="size-4" /> : <Pause className="size-4" />}
                    {s.paused ? 'Возобновить' : 'Приостановить'}
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
      {reminders}
      <Dialog
        open={edit !== null}
        onOpenChange={(open) => {
          if (!open) setEdit(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {edit === 'new' ? 'Новая подписка' : 'Редактировать подписку'}
            </DialogTitle>
            <DialogDescription>
              Ожидаемое списание и необязательная карта. Суммы хранятся в валюте подписки.
            </DialogDescription>
          </DialogHeader>
          {edit && (
            <SubscriptionForm
              key={edit === 'new' ? 'new' : edit.id}
              initial={edit === 'new' ? undefined : edit}
              accounts={accounts.data ?? []}
              categories={categories.data ?? []}
              onSaved={() => setEdit(null)}
            />
          )}
        </DialogContent>
      </Dialog>
      <Dialog
        open={expense !== null}
        onOpenChange={(open) => {
          if (!open) setExpense(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Расходы: {expense?.name}</DialogTitle>
            <DialogDescription>
              Свяжите уже созданную покупку. Это не создаёт повторное списание.
            </DialogDescription>
          </DialogHeader>
          <ExpenseHistory
            loading={expenseHistory.isPending}
            error={expenseHistory.error}
            events={expenseHistory.data ?? []}
          />
          <p className="text-sm text-muted-foreground">
            Связано покупок: {expense?.expenseIds.length ?? 0}. Ниже — до 100 последних операций.
          </p>
          {events.isError ? (
            <p role="alert">{events.error.message}</p>
          ) : (
            <ChoiceSelect
              aria-label="Фактическая покупка"
              value={eventID}
              onValueChange={setEventID}
            >
              <option value="">Выберите покупку</option>
              {candidates.map((e) => (
                <option value={e.id} key={e.id}>
                  {e.merchant || e.description || 'Покупка'} ·{' '}
                  {formatMoney(e.postings[0].amountMinor, e.postings[0].currency)}
                </option>
              ))}
            </ChoiceSelect>
          )}
          <Button
            disabled={!eventID || link.isPending}
            onClick={() => link.mutate({ subscription: expense, financialEventId: eventID })}
          >
            Связать расход
          </Button>
          <Button variant="outline" onClick={() => setPurchaseOpen(true)}>
            Создать новую покупку
          </Button>
        </DialogContent>
      </Dialog>
      <Dialog open={purchaseOpen} onOpenChange={setPurchaseOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Новая покупка</DialogTitle>
            <DialogDescription>
              После подтверждения выберите покупку в списке и свяжите её с подпиской.
            </DialogDescription>
          </DialogHeader>
          <CreateFinancialEventForm
            accounts={accounts.data ?? []}
            categories={categories.data ?? []}
            events={events.data ?? []}
            onSaved={() => {
              setPurchaseOpen(false)
              void events.refetch()
            }}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
