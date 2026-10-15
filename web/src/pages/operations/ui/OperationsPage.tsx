import { LedgerScreen } from '@/widgets/ledger-screen'
import { Operations } from '@/widgets/operations-list'

export function OperationsPage() {
  return (
    <LedgerScreen
      title="Операции"
      description="Покупки, поступления и переводы."
      accountAction="when-empty"
      operationAction
    >
      {({ categories, accounts, receipts }) => (
        <Operations
          categories={categories.data ?? []}
          accounts={accounts.data ?? []}
          receipts={receipts.data ?? []}
        />
      )}
    </LedgerScreen>
  )
}
