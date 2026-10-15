import { Accounts } from '@/widgets/accounts-list'
import { LedgerScreen } from '@/widgets/ledger-screen'

export function AccountsPage() {
  return (
    <LedgerScreen
      title="Счета и карты"
      description="Счета, карты и подтверждённые остатки."
      accountAction="always"
    >
      {({ accounts }) => accounts.data && <Accounts items={accounts.data} />}
    </LedgerScreen>
  )
}
