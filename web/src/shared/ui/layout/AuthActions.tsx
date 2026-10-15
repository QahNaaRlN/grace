import { useContext, useState } from 'react'

import { apiClient } from '@/shared/api/client'
import { AuthContext } from '@/shared/model/auth-context'
import { csrfToken } from '@/shared/model/auth-session'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'

export function AuthActions() {
  const auth = useContext(AuthContext)
  const [name, setName] = useState('')
  const [invitation, setInvitation] = useState('')
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteUrl, setInviteUrl] = useState('')
  const [boundEmail, setBoundEmail] = useState('')
  const [inviteExpires, setInviteExpires] = useState('')
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  if (!auth) return null

  async function action(run: () => Promise<void>) {
    if (pending) return
    setPending(true)
    setError('')
    try {
      await run()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Действие недоступно.')
    } finally {
      setPending(false)
    }
  }
  async function invite() {
    if (auth?.session.mode === 'password') {
      const { data, error } = await apiClient.POST('/api/auth/password/invitations', {
        body: { name, email: inviteEmail },
        params: { header: { 'X-CSRF-Token': csrfToken } },
      })
      if (!data) throw new Error(error?.message ?? 'Не удалось создать приглашение.')
      setInvitation(data.token)
      setInviteUrl(data.url)
      setBoundEmail(data.email)
      setInviteExpires(data.expiresAt)
      setNotice('Приглашение готово. Отправьте ссылку или код человеку самостоятельно.')
      return
    }
    const { data } = await apiClient.POST('/api/auth/invitations', {
      body: { name },
      params: { header: { 'X-CSRF-Token': csrfToken } },
    })
    if (!data) throw new Error('Не удалось создать приглашение.')
    setInvitation(data.token)
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Dialog
        onOpenChange={() => {
          setInvitation('')
          setInviteUrl('')
          setBoundEmail('')
          setInviteExpires('')
          setNotice('')
          setError('')
        }}
      >
        <DialogTrigger asChild>
          <Button variant="outline">Доступ</Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Доступ к Grace</DialogTitle>
          </DialogHeader>
          <p>{auth.session.user.name}</p>
          {auth.session.mode === 'development' && (
            <p className="text-sm text-muted-foreground">Локальный режим разработки</p>
          )}
          {auth.session.user.role === 'admin' && (
            <div className="space-y-3">
              <Label htmlFor="invite-name">Имя приглашённого пользователя</Label>
              <Input
                id="invite-name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                maxLength={160}
              />
              {auth.session.mode === 'password' && (
                <>
                  <Label htmlFor="invite-email">Почта приглашённого</Label>
                  <Input
                    id="invite-email"
                    type="email"
                    autoComplete="off"
                    maxLength={254}
                    value={inviteEmail}
                    onChange={(event) => setInviteEmail(event.target.value)}
                  />
                </>
              )}
              <Button
                disabled={
                  pending ||
                  !name.trim() ||
                  (auth.session.mode === 'password' && !inviteEmail.trim())
                }
                onClick={() => void action(invite)}
              >
                Создать приглашение
              </Button>
              {inviteUrl && (
                <div className="space-y-2">
                  <Label htmlFor="invite-url">Ссылка для регистрации</Label>
                  <Input id="invite-url" readOnly value={inviteUrl} />
                  <Button
                    variant="outline"
                    disabled={pending}
                    onClick={() =>
                      void action(async () => {
                        await navigator.clipboard.writeText(inviteUrl)
                        setNotice('Ссылка скопирована.')
                      })
                    }
                  >
                    Скопировать ссылку
                  </Button>
                  <Button
                    variant="outline"
                    disabled={pending}
                    onClick={() =>
                      void action(async () => {
                        await navigator.clipboard.writeText(invitation)
                        setNotice('Код скопирован.')
                      })
                    }
                  >
                    Скопировать код
                  </Button>
                  <p className="break-words text-sm text-muted-foreground">
                    Только для {boundEmail}. Код нужен один раз при регистрации. Действует до{' '}
                    {new Date(inviteExpires).toLocaleString('ru-RU')}.
                  </p>
                </div>
              )}
              {invitation && (
                <div className="space-y-2">
                  <Label htmlFor="invite-code">Одноразовый код (действует 24 часа)</Label>
                  <Input id="invite-code" readOnly value={invitation} />
                  <p className="text-sm text-muted-foreground">
                    Передайте код пользователю лично. Его финансовые данные будут отдельными.
                  </p>
                </div>
              )}
            </div>
          )}
          <Button
            variant="outline"
            disabled={pending}
            onClick={() => void action(auth.revokeSessions)}
          >
            Завершить все мои сеансы
          </Button>
          <Button disabled={pending} onClick={() => void action(auth.logout)}>
            Выйти
          </Button>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          )}
          {notice && (
            <p role="status" className="text-sm">
              {notice}
            </p>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
