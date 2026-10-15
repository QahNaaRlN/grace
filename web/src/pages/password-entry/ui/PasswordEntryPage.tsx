import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { authMethodQueries } from '@/entities/user'
import { PasswordForm, type PasswordView } from '@/features/password-auth'
import { Button } from '@/shared/ui/button'
import { AuthShell } from '@/shared/ui/auth-shell'

export function PasswordEntryPage({ view }: { view: Exclude<PasswordView, 'login'> }) {
  const methods = useQuery(authMethodQueries.list())
  const titles = {
    register: 'Регистрация в Grace',
    forgot: 'Восстановление пароля',
    reset: 'Новый пароль',
  }
  return (
    <AuthShell title={titles[view]}>
      {methods.isPending ? (
        <p>Проверяем доступ…</p>
      ) : methods.data?.password ? (
        <PasswordForm view={view} />
      ) : (
        <div className="space-y-4">
          <p role="alert">Этот способ доступа пока недоступен.</p>
          <Button variant="outline" onClick={() => void methods.refetch()}>
            Повторить
          </Button>
          <Link className="block text-primary underline" to="/login">
            Вернуться ко входу
          </Link>
        </div>
      )}
    </AuthShell>
  )
}
