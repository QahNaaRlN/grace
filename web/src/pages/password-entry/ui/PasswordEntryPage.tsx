import type { PasswordView } from '@/features/auth/password'
import { AuthShell } from '@/shared/ui/auth-shell'
import { PasswordEntry } from '@/widgets/sign-in'

const titles = {
  register: 'Регистрация в Grace',
  forgot: 'Восстановление пароля',
  reset: 'Новый пароль',
}

export function PasswordEntryPage({ view }: { view: Exclude<PasswordView, 'login'> }) {
  return (
    <AuthShell title={titles[view]}>
      <PasswordEntry view={view} />
    </AuthShell>
  )
}
