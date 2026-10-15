import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { useRef, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { userQueries } from '@/entities/user'
import type { components } from '@/shared/api/generated/schema'
import { env } from '@/shared/config'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { useProfileActions } from '../api/use-profile-actions'

type Profile = components['schemas']['UserProfile']
type Area = 'personal' | 'email'
type Status = { area: Area; kind: 'notice' | 'error'; text: string }
// sessionActions: access and sign-out controls composed by the page (features/logout).
function ProfileForm({
  profile,
  sessionActions,
}: {
  profile: Profile
  sessionActions?: ReactNode
}) {
  const actions = useProfileActions(profile)
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
  async function saveNames() {
    const data = await actions.saveNames({
      displayName: displayName.trim(),
      firstName: firstName.trim(),
      lastName: lastName.trim(),
    })
    setFirst(data.firstName)
    setLast(data.lastName)
    setDisplay(data.displayName)
    setCustomName(false)
    return 'Профиль сохранён.'
  }
  async function upload(file: File) {
    await actions.uploadAvatar(file)
    return 'Аватар обновлён.'
  }
  async function remove() {
    await actions.removeAvatar()
    return 'Аватар удалён.'
  }
  async function requestEmail() {
    await actions.requestEmail(email)
    setEmailStep(true)
    return `Код отправлен на ${email}.`
  }
  async function verifyEmail() {
    await actions.verifyEmail(code, () => {
      setEmailStep(false)
      setCode('')
    })
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
                  if (!customName) setDisplay([e.target.value, lastName].filter(Boolean).join(' '))
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
                  if (!customName) setDisplay([firstName, e.target.value].filter(Boolean).join(' '))
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
        <p className="mt-3 break-words">{profile.email ?? 'Почта пока не привязана'}</p>
        {profile.email && (
          <p className="mt-1 text-sm text-muted-foreground">Подтверждённый адрес для входа</p>
        )}
        <p className="mt-4 text-sm text-muted-foreground">
          Для смены адреса нужен недавний вход и подтверждение нового адреса кодом.
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
              <Label htmlFor="profile-email">{profile.email ? 'Новая почта' : 'Почта'}</Label>
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
        <div className="grace-actions">{sessionActions}</div>
      </section>
    </div>
  )
}
export function ProfilePanel({ sessionActions }: { sessionActions?: ReactNode }) {
  const result = useQuery(userQueries.profile())
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
  return <ProfileForm key={result.data.id} profile={result.data} sessionActions={sessionActions} />
}
