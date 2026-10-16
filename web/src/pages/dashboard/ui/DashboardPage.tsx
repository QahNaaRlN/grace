import { CreditCardsDue } from '@/widgets/credit-cards-due'
import { LedgerScreen } from '@/widgets/ledger-screen'
import { Overview } from '@/widgets/overview'
import { StatisticsPanel } from '@/widgets/statistics-charts'

export function DashboardPage() {
  return (
    <LedgerScreen
      title="Обзор"
      description="Деньги и ближайшие дела — в одном месте."
      accountAction="when-empty"
      operationAction
    >
      {({ cash, events, accounts }) => (
        <div className="space-y-6">
          <Overview
            cash={cash.data}
            pending={cash.isPending}
            failed={cash.isError}
            events={events.data ?? []}
            accounts={accounts.data ?? []}
            retry={() => void cash.refetch()}
          />
          <CreditCardsDue accounts={accounts.data ?? []} />
          <StatisticsPanel compact />
        </div>
      )}
    </LedgerScreen>
  )
}
