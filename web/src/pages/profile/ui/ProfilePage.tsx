import { AuthActions } from '@/features/logout'
import { ProfilePanel } from '@/features/user-profile'
import { LedgerScreen } from '@/widgets/ledger-screen'

export function ProfilePage() {
  return (
    <LedgerScreen
      title="Профиль и доступ"
      description="Личные данные, почта и управление доступом."
      gated={false}
    >
      {() => <ProfilePanel sessionActions={<AuthActions />} />}
    </LedgerScreen>
  )
}
