import { useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { authMethodQueries } from '@/entities/user'
import { sessionQueries } from '@/entities/session'
import { apiClient } from '@/shared/api'
import { setCSRFToken } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import { AuthShell } from '@/shared/ui/auth-shell'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { PasswordForm } from './PasswordEntry'

export function LoginPage() {
  const [params] = useSearchParams(),
    navigate = useNavigate(),
    cache = useQueryClient()
  const [email, setEmail] = useState(''),
    [invitation, setInvitation] = useState(''),
    [code, setCode] = useState('')
  const [step, setStep] = useState<'email' | 'code'>('email'),
    [pending, setPending] = useState(false)
  const [error, setError] = useState(
    params.has('error') ? 'Вход не завершён. Попробуйте снова.' : '',
  )
  const [cooldown, setCooldown] = useState(0)
  const sending = useRef(false)
  const methods = useQuery({ ...authMethodQueries.list(), retry: false })
  const session = useQuery(sessionQueries.me())
  useEffect(() => {
    if (session.data) void navigate('/', { replace: true })
  }, [session.data, navigate])
  useEffect(() => {
    if (cooldown <= 0) return
    const id = window.setTimeout(() => setCooldown((v) => Math.max(v - 1, 0)), 1000)
    return () => clearTimeout(id)
  }, [cooldown])
  async function requestCode() {
    if (sending.current) return
    sending.current = true
    setPending(true)
    setError('')
    try {
      const { data, error: failure } = await apiClient.POST('/api/auth/email/request', {
        body: {
          email: email.trim(),
          ...(invitation.trim() ? { invitation: invitation.trim() } : {}),
        },
      })
      if (!data) throw new Error(failure?.message ?? 'Не удалось отправить код.')
      setStep('code')
      setCode('')
      setCooldown(data.resendAfterSeconds)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Отправка временно недоступна.')
    } finally {
      sending.current = false
      setPending(false)
    }
  }
  async function verify() {
    if (sending.current) return
    sending.current = true
    setPending(true)
    setError('')
    try {
      const { data, error: failure } = await apiClient.POST('/api/auth/email/verify', {
        body: { code },
      })
      if (!data) throw new Error(failure?.message ?? 'Не удалось подтвердить код.')
      cache.clear()
      setCSRFToken(data.csrfToken)
      cache.setQueryData(sessionQueries.me().queryKey, data)
      void navigate('/', { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Вход временно недоступен.')
    } finally {
      sending.current = false
      setPending(false)
    }
  }
  async function telegram() {
    setPending(true)
    setError('')
    try {
      const { data } = await apiClient.POST('/api/auth/login', {
        body: invitation.trim() ? { invitation: invitation.trim() } : {},
      })
      if (!data) throw new Error('Не удалось начать вход.')
      window.location.assign(data.authorizationUrl)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Вход недоступен.')
      setPending(false)
    }
  }
  return (
    <AuthShell title="Вход в Grace">
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
        <>
          {step === 'email' ? (
            <form
              className="space-y-5"
              onSubmit={(e) => {
                e.preventDefault()
                void requestCode()
              }}
            >
              <p className="text-muted-foreground">
                Введите почту — пришлём одноразовый код. Пароль не нужен.
              </p>
              <div className="space-y-2">
                <Label htmlFor="login-email">Почта</Label>
                <Input
                  id="login-email"
                  type="email"
                  autoComplete="email"
                  required
                  maxLength={254}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="invitation">Код приглашения (при первом входе)</Label>
                <Input
                  id="invitation"
                  autoComplete="off"
                  maxLength={43}
                  value={invitation}
                  onChange={(e) => setInvitation(e.target.value)}
                />
              </div>
              <div className="grace-actions">
                <Button className="w-full" type="submit" disabled={pending}>
                  {pending ? 'Отправляем…' : 'Получить код'}
                </Button>
              </div>
            </form>
          ) : (
            <form
              className="space-y-5"
              onSubmit={(e) => {
                e.preventDefault()
                void verify()
              }}
            >
              <p className="break-words text-muted-foreground">
                Код отправлен на {email}. Введите его здесь; он действует 10 минут.
              </p>
              <div className="space-y-2">
                <Label htmlFor="login-code">Код из письма</Label>
                <Input
                  id="login-code"
                  autoComplete="one-time-code"
                  inputMode="numeric"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  required
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
                />
              </div>
              <div className="grace-actions">
                <Button type="submit" disabled={pending || code.length !== 6}>
                  {pending ? 'Проверяем…' : 'Войти'}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={pending}
                  onClick={() => {
                    setStep('email')
                    setError('')
                  }}
                >
                  Изменить почту
                </Button>
              </div>
              <Button
                className="mt-4 w-full"
                type="button"
                variant="ghost"
                disabled={pending || cooldown > 0}
                onClick={() => void requestCode()}
              >
                {cooldown > 0 ? `Повторная отправка через ${cooldown} с` : 'Отправить код ещё раз'}
              </Button>
            </form>
          )}
        </>
      ) : methods.data?.telegram ? (
        <form
          className="space-y-5"
          onSubmit={(e) => {
            e.preventDefault()
            void telegram()
          }}
        >
          <p className="text-muted-foreground">
            Войдите через Telegram. При первом входе нужен код приглашения.
          </p>
          <div className="space-y-2">
            <Label htmlFor="invitation">Код приглашения (при первом входе)</Label>
            <Input
              id="invitation"
              value={invitation}
              onChange={(e) => setInvitation(e.target.value)}
              maxLength={43}
            />
          </div>
          <div className="grace-actions">
            <Button disabled={pending} type="submit">
              Войти через Telegram
            </Button>
          </div>
        </form>
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
    </AuthShell>
  )
}
