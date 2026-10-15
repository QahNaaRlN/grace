import { FinancialHealth } from '@/widgets/financial-health'
import { LedgerScreen } from '@/widgets/ledger-screen'

export function HealthPage() {
  return (
    <LedgerScreen
      title="Финансовое здоровье"
      description="Ближайшие обязательства, доступные деньги и качество данных."
    >
      {({ incomplete, cash, accounts, categories, events, obligations, receipts }) => (
        <FinancialHealth
          incomplete={incomplete}
          cash={cash.isError ? undefined : cash.data}
          accounts={accounts.data ?? []}
          categories={categories.data ?? []}
          events={events.data ?? []}
          obligations={obligations.data ?? []}
          receipts={receipts.data ?? []}
        />
      )}
    </LedgerScreen>
  )
}
