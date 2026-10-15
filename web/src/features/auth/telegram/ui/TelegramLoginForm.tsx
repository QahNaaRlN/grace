import { useState } from 'react'

import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { startTelegramLogin } from '../api/telegram-login'

/** Sign-in through Telegram; the first sign-in needs an invitation code. */
export function TelegramLoginForm({ onError }: { onError: (message: string) => void }) {
  const [invitation, setInvitation] = useState('')
  const [pending, setPending] = useState(false)
  async function submit() {
    setPending(true)
    onError('')
    try {
      window.location.assign(await startTelegramLogin(invitation.trim()))
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Вход недоступен.')
      setPending(false)
    }
  }
  return (
    <form
      className="space-y-5"
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
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
  )
}
