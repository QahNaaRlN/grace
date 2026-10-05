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
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  if (!auth) return null

  async function action(run: () => Promise<void>) {
    if (pending) return
    setPending(true)
    setError('')
    try { await run() } catch (err) { setError(err instanceof Error ? err.message : 'Действие недоступно.') } finally { setPending(false) }
  }
  async function invite() {
    const { data } = await apiClient.POST('/api/auth/invitations', { body: { name }, params: { header: { 'X-CSRF-Token': csrfToken } } })
    if (!data) throw new Error('Не удалось создать приглашение.')
    setInvitation(data.token)
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Dialog onOpenChange={() => { setInvitation(''); setError('') }}>
        <DialogTrigger asChild><Button variant="outline" size="sm">Доступ</Button></DialogTrigger>
        <DialogContent>
          <DialogHeader><DialogTitle>Доступ к Grace</DialogTitle></DialogHeader>
          <p>{auth.session.user.name}</p>
          {auth.session.mode === 'development' && <p className="text-sm text-muted-foreground">Локальный режим разработки</p>}
          {auth.session.user.role === 'admin' && <div className="space-y-3">
            <Label htmlFor="invite-name">Имя приглашённого пользователя</Label>
            <Input id="invite-name" value={name} onChange={event => setName(event.target.value)} maxLength={160} />
            <Button disabled={pending || !name.trim()} onClick={() => void action(invite)}>Создать приглашение</Button>
            {invitation && <div className="space-y-2"><Label htmlFor="invite-code">Одноразовый код (действует 24 часа)</Label><Input id="invite-code" readOnly value={invitation} /><p className="text-sm text-muted-foreground">Передайте код пользователю лично. Его финансовые данные будут отдельными.</p></div>}
          </div>}
          <Button variant="outline" disabled={pending} onClick={() => void action(auth.revokeSessions)}>Завершить все мои сеансы</Button>
          <Button disabled={pending} onClick={() => void action(auth.logout)}>Выйти</Button>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
        </DialogContent>
      </Dialog>
    </div>
  )
}
