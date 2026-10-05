import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { apiClient } from '@/shared/api/client'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'

export function LoginPage() {
  const [params] = useSearchParams()
  const [invitation, setInvitation] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState(params.has('error') ? 'Вход не завершён. Проверьте приглашение и попробуйте снова.' : '')

  async function login(event: React.FormEvent) {
    event.preventDefault()
    if (pending) return
    setPending(true)
    setError('')
    try {
      const { data } = await apiClient.POST('/api/auth/login', { body: invitation.trim() ? { invitation: invitation.trim() } : {} })
      if (!data) throw new Error('Вход недоступен. Проверьте приглашение или обратитесь к владельцу.')
      window.location.assign(data.authorizationUrl)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось войти.')
      setPending(false)
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-md items-center px-4 py-8">
      <Card className="w-full">
        <CardHeader><CardTitle>Добро пожаловать в Grace</CardTitle></CardHeader>
        <CardContent>
          <form onSubmit={event => void login(event)} className="space-y-5">
            <p className="text-sm text-muted-foreground">Войдите через Telegram, чтобы открыть свои финансы. Для первого входа нужен код приглашения от владельца.</p>
            <div className="space-y-2"><Label htmlFor="invitation">Код приглашения (при первом входе)</Label><Input id="invitation" autoComplete="off" value={invitation} onChange={event => setInvitation(event.target.value)} maxLength={43} /></div>
            {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
            <Button type="submit" disabled={pending} className="w-full">{pending ? 'Переходим в Telegram…' : 'Войти через Telegram'}</Button>
            <p className="text-xs text-muted-foreground">Grace не запрашивает доступ к переписке или номеру телефона.</p>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
