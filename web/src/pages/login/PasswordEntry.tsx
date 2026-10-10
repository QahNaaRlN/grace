import { PasswordInput } from '@/shared/ui/password-input'
import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/shared/api/client'
import { setCSRFToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { AuthShell } from '@/shared/ui/auth-shell'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'

type View = 'login' | 'register' | 'forgot' | 'reset'

export function PasswordForm({ view }: { view: View }) {
  const navigate = useNavigate(), cache = useQueryClient()
  const [fragment] = useState(() => new URLSearchParams(window.location.hash.slice(1)))
  const [token, setToken] = useState(fragment.get('token') ?? '')
  const [invitation, setInvitation] = useState(fragment.get('invitation') ?? '')
  const [email, setEmail] = useState(''), [password, setPassword] = useState(''), [repeat, setRepeat] = useState('')
  const [busy, setBusy] = useState(false), [done, setDone] = useState(false)
  const [error, setError] = useState(''), [notice, setNotice] = useState(''), [cooldown, setCooldown] = useState(0)
  const sending = useRef(false)
  useEffect(() => {
    if (window.location.hash) window.history.replaceState(null, '', window.location.pathname + window.location.search)
    const previous = document.querySelector('meta[name="referrer"]')
    const meta = document.createElement('meta'); meta.name = 'referrer'; meta.content = 'no-referrer'; document.head.append(meta)
    return () => { meta.remove(); if (previous) document.head.append(previous) }
  }, [])
  useEffect(() => {
    if (cooldown <= 0) return
    const timer = window.setTimeout(() => setCooldown(value => Math.max(0, value - 1)), 1000)
    return () => window.clearTimeout(timer)
  }, [cooldown])
  const confirmation = view === 'register' && token !== ''
  const newPassword = view === 'reset' || (view === 'register' && !confirmation)
  async function submit() {
    if (sending.current) return
    if (newPassword && (password !== repeat || Array.from(password).length < 15 || Array.from(password).length > 128)) { setError('Пароли должны совпадать и содержать от 15 до 128 символов.'); return }
    sending.current = true; setBusy(true); setError(''); setNotice('')
    try {
      if (view === 'login') {
        const { data, error } = await apiClient.POST('/api/auth/password/login', { body: { email, password } })
        if (!data) throw new Error(error?.message ?? 'Не удалось войти.')
        setCSRFToken(data.csrfToken); setPassword(''); cache.clear(); void navigate('/', { replace: true }); return
      }
      if (confirmation || view === 'reset') {
        const result = confirmation
          ? await apiClient.POST('/api/auth/password/confirm', { body: { token } })
          : await apiClient.POST('/api/auth/password/reset', { body: { token, password } })
        if (!result.response.ok) throw new Error(result.error?.message ?? 'Ссылка недействительна. Запросите новое письмо.')
        setToken(''); setPassword(''); setRepeat(''); setDone(true); setCSRFToken(''); cache.clear()
        setNotice(confirmation ? 'Почта подтверждена. Аккаунт готов — войдите с паролем.' : 'Пароль сохранён. Старые сеансы завершены — войдите заново.'); return
      }
      const result = view === 'register'
        ? await apiClient.POST('/api/auth/password/register', { body: { email, password, invitation } })
        : await apiClient.POST('/api/auth/password/forgot', { body: { email } })
      if (!result.data) throw new Error(result.error?.message ?? 'Не удалось отправить письмо.')
      setCooldown(result.data.resendAfterSeconds)
      setNotice(view === 'register' ? 'Откройте письмо на этой почте и подтвердите регистрацию. Приглашение пока не использовано.' : 'Если почта зарегистрирована, на неё придёт ссылка для установки пароля. Приглашение не нужно.')
    } catch (err) { setError(err instanceof Error ? err.message : 'Действие недоступно.') }
    finally { sending.current = false; setBusy(false) }
  }
  if (done) return <div className="space-y-5"><p role="status">{notice}</p><Button asChild><Link to="/login">Перейти ко входу</Link></Button></div>
  if (view === 'reset' && !token) return <div className="space-y-4"><p>Откройте ссылку из письма. Если она истекла, запросите новую.</p><Button asChild><Link to="/forgot-password">Запросить ссылку</Link></Button></div>
  return <form className="space-y-5" onSubmit={event => { event.preventDefault(); void submit() }}>
    {view === 'login' && <p className="text-sm text-muted-foreground">Раньше входили по почтовому коду? Установите первый пароль через «Забыли пароль?». Приглашение не потребуется.</p>}
    {view === 'forgot' && <p className="text-sm text-muted-foreground">Укажите подтверждённую почту. Здесь же можно установить первый пароль для существующего аккаунта Grace.</p>}
    {confirmation ? <p>Подтвердите почту и завершите регистрацию. Это действие создаст доступ к вашему аккаунту.</p> : view !== 'reset' && <div className="space-y-2"><Label htmlFor="password-email">Почта</Label><Input id="password-email" type="email" autoComplete="username" value={email} onChange={e => setEmail(e.target.value)} required maxLength={254} /></div>}
    {!confirmation && view !== 'forgot' && <>
      <div className="space-y-2"><Label htmlFor="entry-password">{view === 'reset' ? 'Новый пароль' : 'Пароль'}</Label><PasswordInput id="entry-password" autoComplete={newPassword ? 'new-password' : 'current-password'} value={password} onChange={e => setPassword(e.target.value)} required aria-describedby={newPassword ? 'password-help' : undefined} />{newPassword && <p id="password-help" className="text-sm text-muted-foreground">От 15 до 128 символов. Можно использовать фразу и вставлять пароль из менеджера.</p>}</div>
      {newPassword && <div className="space-y-2"><Label htmlFor="repeat-password">Повторите пароль</Label><PasswordInput id="repeat-password" visibilityLabel="повтор пароля" autoComplete="new-password" required value={repeat} onChange={e => setRepeat(e.target.value)} /></div>}
    </>}
    {view === 'register' && !confirmation && <div className="space-y-2"><Label htmlFor="registration-invite">Код приглашения</Label><Input id="registration-invite" value={invitation} onChange={e => setInvitation(e.target.value.trim())} autoComplete="off" required minLength={43} maxLength={43} /><p className="text-sm text-muted-foreground">Приглашение действует только для указанной администратором почты и используется один раз.</p></div>}
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    {notice && <p role="status" className="text-sm">{notice}</p>}
    <Button className="w-full" disabled={busy || cooldown > 0} type="submit">{busy ? 'Подождите…' : cooldown > 0 ? `Повторить через ${cooldown} с` : confirmation ? 'Подтвердить почту' : view === 'login' ? 'Войти' : view === 'register' ? 'Зарегистрироваться' : view === 'forgot' ? 'Получить ссылку' : 'Сохранить пароль'}</Button>
    <nav aria-label="Способы доступа" className="flex flex-wrap gap-x-5 gap-y-2 text-sm">
      {view === 'login' ? <><Link className="inline-flex min-h-11 items-center text-primary underline underline-offset-4" to="/forgot-password">Забыли пароль?</Link><Link className="inline-flex min-h-11 items-center text-primary underline underline-offset-4" to="/register">Регистрация</Link></> : <Link className="inline-flex min-h-11 items-center text-primary underline underline-offset-4" to="/login">Уже есть аккаунт? Войти</Link>}
      {view === 'reset' && <Link className="inline-flex min-h-11 items-center text-primary underline underline-offset-4" to="/forgot-password">Запросить новую ссылку</Link>}
    </nav>
  </form>
}
export function PasswordEntry({ view }: { view: Exclude<View, 'login'> }) {
  const methods = useQuery({ queryKey: ['auth-methods'], queryFn: async () => { const { data } = await apiClient.GET('/api/auth/methods'); if (!data) throw new Error('Проверка доступа недоступна.'); return data } })
  const titles = { register: 'Регистрация в Grace', forgot: 'Восстановление пароля', reset: 'Новый пароль' }
  return <AuthShell title={titles[view]}>{methods.isPending ? <p>Проверяем доступ…</p> : methods.data?.password ? <PasswordForm view={view} /> : <div className="space-y-4"><p role="alert">Этот способ доступа пока недоступен.</p><Button variant="outline" onClick={() => void methods.refetch()}>Повторить</Button><Link className="block text-primary underline" to="/login">Вернуться ко входу</Link></div>}</AuthShell>
}
