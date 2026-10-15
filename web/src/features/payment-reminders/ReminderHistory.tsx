import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/shared/api/client'
import { Button } from '@/shared/ui/button'
import { Disclosure } from '@/shared/ui/disclosure'
import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { dateText } from '@/shared/lib/date'

const labels = {
  pending: 'Ожидает отправки',
  dispatching: 'Передаётся почтовому сервису',
  sent: 'Принято почтовым сервисом',
  cancelled: 'Отменено',
  uncertain: 'Доставка не подтверждена',
}
export function ReminderHistory() {
  const history = useQuery({
    queryKey: ['payment-reminders'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/payment-reminders')
      if (error) throw new Error(error.message)
      return data?.items ?? []
    },
    refetchInterval: 60000,
  })
  return (
    <Disclosure
      className="rounded-2xl border bg-card p-5"
      triggerClassName="min-h-11 font-medium"
      title="Почтовые напоминания"
    >
      <div className="mt-4 space-y-4">
        <p className="text-sm text-muted-foreground">
          До 100 последних напоминаний по подпискам и кредитам. Отправка начинается после 09:00 по
          Екатеринбургу, на подтверждённую почту.
        </p>
        {history.isPending ? (
          <LoadingSkeleton />
        ) : history.isError ? (
          <div>
            <p role="alert">{history.error.message}</p>
            <Button variant="outline" onClick={() => void history.refetch()}>
              Повторить
            </Button>
          </div>
        ) : history.data?.length ? (
          <ul className="space-y-3">
            {history.data.map((item) => (
              <li
                key={item.id}
                className="flex flex-wrap justify-between gap-2 rounded-xl bg-muted/50 p-3 text-sm"
              >
                <span>
                  {item.sourceKind === 'loan' ? 'Кредит' : 'Подписка'} · {dateText(item.dueOn)} · за{' '}
                  {item.leadDays} дн.
                </span>
                <span
                  className={
                    item.status === 'uncertain' ? 'text-destructive' : 'text-muted-foreground'
                  }
                >
                  {labels[item.status]}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-sm text-muted-foreground">
            Отправок пока нет. Включите напоминания в настройках нужного кредита или подписки.
          </p>
        )}
        <p className="text-xs text-muted-foreground">
          При неподтверждённой доставке повторная отправка автоматически не выполняется. Пропущенные
          даты не отправляются задним числом. Принятое сервисом письмо может попасть в спам.
        </p>
      </div>
    </Disclosure>
  )
}
