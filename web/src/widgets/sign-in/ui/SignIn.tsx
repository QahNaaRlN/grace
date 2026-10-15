import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { sessionQueries } from '@/entities/session'
import { authMethodQueries } from '@/entities/user'
import { EmailLoginForm } from '@/features/auth/email'
import { PasswordForm } from '@/features/auth/password'
import { TelegramLoginForm } from '@/features/auth/telegram'
import { Button } from '@/shared/ui/button'

/**
 * The sign-in method the server offers: password, email code or Telegram. A signed-in
 * visitor goes straight to the overview.
 */
export function SignIn() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const [error, setError] = useState(
    params.has('error') ? 'Вход не завершён. Попробуйте снова.' : '',
  )
  const methods = useQuery({ ...authMethodQueries.list(), retry: false })
  const session = useQuery(sessionQueries.me())
  useEffect(() => {
    if (session.data) void navigate('/', { replace: true })
  }, [session.data, navigate])
  return (
    <>
      {methods.isPending ? (
        <p className="text-muted-foreground">Проверяем способы входа…</p>
      ) : methods.error ? (
        <>
          <p role="alert">{methods.error.message}</p>
          <div className="grace-actions">
            <Button onClick={() => void methods.refetch()}>Повторить</Button>
          </div>
        </>
      ) : methods.data?.password ? (
        <PasswordForm view="login" />
      ) : methods.data?.email ? (
        <EmailLoginForm onError={setError} />
      ) : methods.data?.telegram ? (
        <TelegramLoginForm onError={setError} />
      ) : (
        <p role="status" className="text-muted-foreground">
          Вход пока не настроен. Обратитесь к владельцу Grace.
        </p>
      )}
      {error && (
        <p role="alert" className="mt-5 text-destructive">
          {error}
        </p>
      )}
    </>
  )
}
