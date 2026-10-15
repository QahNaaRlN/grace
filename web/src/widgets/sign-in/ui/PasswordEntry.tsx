import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'

import { authMethodQueries } from '@/entities/user'
import { PasswordForm, type PasswordView } from '@/features/auth/password'
import { Button } from '@/shared/ui/button'

/** Registration and password recovery, offered only when the server enables passwords. */
export function PasswordEntry({ view }: { view: Exclude<PasswordView, 'login'> }) {
  const methods = useQuery(authMethodQueries.list())
  return methods.isPending ? (
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
  )
}
