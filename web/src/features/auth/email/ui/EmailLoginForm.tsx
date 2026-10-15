import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { requestEmailCode, useVerifyEmailCode } from '../api/email-login'

/** Passwordless sign-in: the email, then the one-time code from the letter. */
export function EmailLoginForm({ onError }: { onError: (message: string) => void }) {
  const navigate = useNavigate()
  const verifyCode = useVerifyEmailCode()
  const [email, setEmail] = useState(''),
    [invitation, setInvitation] = useState(''),
    [code, setCode] = useState('')
  const [step, setStep] = useState<'email' | 'code'>('email'),
    [pending, setPending] = useState(false)
  const [cooldown, setCooldown] = useState(0)
  const sending = useRef(false)
  useEffect(() => {
    if (cooldown <= 0) return
    const id = window.setTimeout(() => setCooldown((v) => Math.max(v - 1, 0)), 1000)
    return () => clearTimeout(id)
  }, [cooldown])
  async function requestCode() {
    if (sending.current) return
    sending.current = true
    setPending(true)
    onError('')
    try {
      const resendAfter = await requestEmailCode(email.trim(), invitation.trim())
      setStep('code')
      setCode('')
      setCooldown(resendAfter)
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Отправка временно недоступна.')
    } finally {
      sending.current = false
      setPending(false)
    }
  }
  async function verify() {
    if (sending.current) return
    sending.current = true
    setPending(true)
    onError('')
    try {
      await verifyCode(code)
      void navigate('/', { replace: true })
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Вход временно недоступен.')
    } finally {
      sending.current = false
      setPending(false)
    }
  }
  return step === 'email' ? (
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
            onError('')
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
  )
}
