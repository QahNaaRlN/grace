import { ReminderHistory } from '@/widgets/reminder-history'
import { SubscriptionsBoard } from '@/widgets/subscriptions-board'

export function SubscriptionsPage() {
  return <SubscriptionsBoard reminders={<ReminderHistory />} />
}
