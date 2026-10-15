import { AuthShell } from '@/shared/ui/auth-shell'
import { SignIn } from '@/widgets/sign-in'

export function LoginPage() {
  return (
    <AuthShell title="Вход в Grace">
      <SignIn />
    </AuthShell>
  )
}
