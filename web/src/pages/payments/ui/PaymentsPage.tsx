import { CreateObligationForm } from '@/features/create-obligation'
import { ReminderHistory } from '@/features/payment-reminders/ReminderHistory'
import { Disclosure } from '@/shared/ui/disclosure'
import { LedgerScreen } from '@/widgets/ledger-screen'
import { Obligations } from '@/widgets/obligations-board'

export function PaymentsPage() {
  return (
    <LedgerScreen
      title="Платежи"
      description="Обязательства и ближайшие платежи."
      accountAction="when-empty"
    >
      {({ obligations, events }) => (
        <div className="space-y-6">
          <ReminderHistory />
          <Disclosure
            className="rounded-2xl border bg-card p-5"
            triggerClassName="min-h-11 rounded-xl px-3 py-2 hover:bg-accent"
            title={<>Добавить обязательство</>}
          >
            <div className="mt-5">
              <CreateObligationForm />
            </div>
          </Disclosure>
          {obligations.data && <Obligations items={obligations.data} events={events.data ?? []} />}
        </div>
      )}
    </LedgerScreen>
  )
}
