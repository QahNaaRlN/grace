import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/shared/api/client'
import type { components } from '@/shared/api/generated/schema'
import { env } from '@/shared/config/env'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { AuthActions } from '@/shared/ui/layout/AuthActions'

type Profile = components['schemas']['UserProfile']
type Session = components['schemas']['AuthSession']
type Area = 'personal' | 'email'
type Status = { area: Area; kind: 'notice' | 'error'; text: string }
async function loadProfile() {
  const { data } = await apiClient.GET('/api/auth/profile')
  if (!data) throw new Error('Не удалось загрузить профиль.')
  return data
}
function ProfileForm({ profile }: { profile: Profile }) {
  const cache = useQueryClient()
  const [firstName, setFirst] = useState(profile.firstName),
    [lastName, setLast] = useState(profile.lastName),
    [displayName, setDisplay] = useState(profile.displayName)
  const [customName, setCustomName] = useState(false),
    [email, setEmail] = useState(''),
    [code, setCode] = useState(''),
    [emailStep, setEmailStep] = useState(false)
  const [busy, setBusy] = useState(false),
    [status, setStatus] = useState<Status | null>(null)
  const working = useRef(false)
  const dirty =
    firstName.trim() !== profile.firstName ||
    lastName.trim() !== profile.lastName ||
    displayName.trim() !== profile.displayName
  // SPEC-0036: feedback lands in the card whose action produced it.
  async function action(area: Area, run: () => Promise<string>) {
    if (working.current) return
    working.current = true
    setBusy(true)
    setStatus(null)
    try {
      setStatus({ area, kind: 'notice', text: await run() })
    } catch (e) {
      setStatus({
        area,
        kind: 'error',
        text: e instanceof Error ? e.message : 'Не удалось выполнить действие.',
      })
    } finally {
      working.current = false
      setBusy(false)
    }
  }
  // The POST response already carries the new name and avatar: update the
  // session in place instead of refetching /api/auth/me after every save.
  function update(data: Profile) {
    cache.setQueryData(['user-profile'], data)
    cache.setQueryData<Session | null>(['auth-session'], (current) =>
      current
        ? { ...current, user: { ...current.user, name: data.displayName, avatarUrl: data.avatarUrl } }
        : current,
    )
  }
  function feedback(area: Area) {
    if (status?.area !== area) return null
    return status.kind === 'error' ? (
      <p role="alert" className="mt-4 text-sm text-destructive">
        {status.text}
      </p>
    ) : (
      <p role="status" className="mt-4 text-sm text-success">
        {status.text}
      </p>
    )
  }
  const headers = { 'X-CSRF-Token': csrfToken }
  async function saveNames() {
    const { data, error: failure } = await apiClient.POST('/api/auth/profile', {
      params: { header: headers },
      body: {
        displayName: displayName.trim(),
        firstName: firstName.trim(),
        lastName: lastName.trim(),
      },
    })
    if (!data) throw new Error(failure?.message ?? 'Профиль не сохранён.')
    update(data)
    setFirst(data.firstName)
    setLast(data.lastName)
    setDisplay(data.displayName)
    setCustomName(false)
    return 'Профиль сохранён.'
  }
  async function upload(file: File) {
    if (file.size > 2 * 1024 * 1024)
      throw new Error('Изображение должно быть не больше 2 МБ.')
    const form = new FormData()
    form.append('avatar', file)
    const { data, error: failure } = await apiClient.POST(
      '/api/auth/profile/avatar',
      {
        params: { header: headers },
        body: { avatar: '' },
        bodySerializer: () => form,
      },
    )
    if (!data) throw new Error(failure?.message ?? 'Аватар не сохранён.')
    update(data)
    return 'Аватар обновлён.'
  }
  async function remove() {
    const { response, error: failure } = await apiClient.DELETE(
      '/api/auth/profile/avatar',
      { params: { header: headers } },
    )
    if (!response.ok)
      throw new Error(failure?.message ?? 'Не удалось удалить аватар.')
    update({ ...profile, avatarUrl: null })
    return 'Аватар удалён.'
  }
  async function requestEmail() {
    const { data, error: failure } = await apiClient.POST(
      '/api/auth/profile/email/request',
      { params: { header: headers }, body: { email } },
    )
    if (!data) throw new Error(failure?.message ?? 'Не удалось отправить код.')
    setEmailStep(true)
    return `Код отправлен на ${email}.`
  }
  async function verifyEmail() {
    const { response, error: failure } = await apiClient.POST(
      '/api/auth/profile/email/verify',
      { params: { header: headers }, body: { code } },
    )
    if (!response.ok)
      throw new Error(failure?.message ?? 'Не удалось подтвердить почту.')
    setEmailStep(false)
    setCode('')
    await cache.invalidateQueries({ queryKey: ['user-profile'] })
    return 'Почта подтверждена. Другие сеансы завершены.'
  }
  return (
    <div className="grid items-start gap-6 xl:grid-cols-2">
      <section className="rounded-2xl border border-border bg-card p-6">
        <h2>Личные данные</h2>
        <div className="mt-5 flex flex-wrap items-center gap-5">
          <span className="grid size-24 shrink-0 place-items-center overflow-hidden rounded-full bg-secondary text-title-1 text-primary">
            {profile.avatarUrl ? (
              <img
                src={env.VITE_API_BASE_URL + profile.avatarUrl}
                alt="Ваш аватар"
                className="size-full object-cover"
              />
            ) : (
              Array.from(profile.displayName)[0]?.toUpperCase()
            )}
          </span>
          <div className="min-w-0 space-y-3">
            <Label htmlFor="avatar-upload">Аватар</Label>
            <Input
              id="avatar-upload"
              type="file"
              accept="image/png,image/jpeg"
              disabled={busy}
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) void action('personal', () => upload(file))
                e.target.value = ''
              }}
            />
            <p className="text-sm text-muted-foreground">
              PNG или JPEG, до 2 МБ. Фото будет обрезано по центру.
            </p>
            {profile.avatarUrl && (
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => void action('personal', remove)}
              >
                Удалить аватар
              </Button>
            )}
          </div>
        </div>
        <form
          className="mt-6 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (dirty) void action('personal', saveNames)
          }}
        >
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="profile-first">Имя</Label>
              <Input
                id="profile-first"
                autoComplete="given-name"
                maxLength={80}
                value={firstName}
                onChange={(e) => {
                  setFirst(e.target.value)
                  if (!customName)
                    setDisplay(
                      [e.target.value, lastName].filter(Boolean).join(' '),
                    )
                }}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="profile-last">Фамилия</Label>
              <Input
                id="profile-last"
                autoComplete="family-name"
                maxLength={80}
                value={lastName}
                onChange={(e) => {
                  setLast(e.target.value)
                  if (!customName)
                    setDisplay(
                      [firstName, e.target.value].filter(Boolean).join(' '),
                    )
                }}
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="profile-display">Отображаемое имя</Label>
            <Input
              id="profile-display"
              required
              maxLength={160}
              value={displayName}
              onChange={(e) => {
                setCustomName(true)
                setDisplay(e.target.value)
              }}
            />
          </div>
          <div className="grace-actions">
            <Button type="submit" disabled={busy || !dirty || !displayName.trim()}>
              {busy ? 'Сохранение…' : 'Сохранить профиль'}
            </Button>
          </div>
        </form>
        {feedback('personal')}
      </section>
      <section className="rounded-2xl border border-border bg-card p-6">
        <h2>Почта и вход</h2>
        <p className="mt-3 break-words">
          {profile.email ?? 'Почта пока не привязана'}
        </p>
        {profile.email && (
          <p className="mt-1 text-sm text-muted-foreground">
            Подтверждённый адрес для входа
          </p>
        )}
        <p className="mt-4 text-sm text-muted-foreground">
          Для смены адреса нужен недавний вход и подтверждение нового адреса
          кодом.
        </p>
        <form
          className="mt-5 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            void action('email', emailStep ? verifyEmail : requestEmail)
          }}
        >
          {emailStep ? (
            <div className="space-y-2">
              <Label htmlFor="profile-code">Код из письма на {email}</Label>
              <Input
                id="profile-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                pattern="[0-9]{6}"
                required
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
              />
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="profile-email">
                {profile.email ? 'Новая почта' : 'Почта'}
              </Label>
              <Input
                id="profile-email"
                type="email"
                autoComplete="email"
                maxLength={254}
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
          )}
          <div className="grace-actions">
            <Button disabled={busy} type="submit">
              {emailStep ? 'Подтвердить почту' : 'Получить код'}
            </Button>
            {emailStep && (
              <Button
                variant="outline"
                type="button"
                disabled={busy}
                onClick={() => setEmailStep(false)}
              >
                Изменить адрес
              </Button>
            )}
          </div>
        </form>
        {feedback('email')}
      </section>
      <section className="rounded-2xl border border-border bg-card p-6">
        <h2>Доступ и сеансы</h2>
        <div className="grace-actions">
          <AuthActions />
        </div>
      </section>
    </div>
  )
}
export function ProfilePanel() {
  const result = useQuery({ queryKey: ['user-profile'], queryFn: loadProfile })
  if (result.isPending) return <LoadingSkeleton label="Загружаем профиль…" />
  if (result.error)
    return (
      <div>
        <p role="alert">{result.error.message}</p>
        <div className="grace-actions">
          <Button onClick={() => void result.refetch()}>Повторить</Button>
        </div>
      </div>
    )
  return <ProfileForm key={result.data.id} profile={result.data} />
}
