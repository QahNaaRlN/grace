import { useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/shared/api/client'
import { csrfToken } from '@/shared/model/auth-session'
import { parseMinor, formatMoney } from '@/shared/lib/finance-display'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/shared/ui/select'

export function CreateAccountForm({ onSaved }: { onSaved: () => void }) {
  const query = useQueryClient()
  const [type, setType] = useState('debit')
  const [card, setCard] = useState(true)
  const [review, setReview] = useState<{
    name: string
    bank: string
    last4: string
    balance: number
  } | null>(null)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const saving = useRef(false)
  async function save() {
    if (!review || saving.current) return
    saving.current = true
    setPending(true)
    setError('')
    try {
      const { data, error: failure } = await apiClient.POST(
        '/api/v1/accounts',
        {
          params: { header: { 'X-CSRF-Token': csrfToken } },
          body: {
            name: review.name,
            type: type as 'debit' | 'credit' | 'cash' | 'installment',
            currency: 'RUB',
            openingBalanceMinor: review.balance,
            ...(card
              ? {
                  card: {
                    name: review.name,
                    bank: review.bank,
                    last4: review.last4,
                  },
                }
              : {}),
          },
        },
      )
      if (!data) {
        setError(failure?.message ?? 'Не удалось создать счёт.')
        return
      }
      await query.invalidateQueries({ queryKey: ['accounts'] })
      await query.invalidateQueries({ queryKey: ['cash-position'] })
      onSaved()
    } catch {
      setError(
        'Не удалось получить ответ. Перед повтором проверьте список счетов: запись могла сохраниться.',
      )
    } finally {
      saving.current = false
      setPending(false)
    }
  }
  return (
    <>
      <form
        hidden={!!review}
        onSubmit={(e) => {
          e.preventDefault()
          const f = new FormData(e.currentTarget)
          const balance = parseMinor(String(f.get('balance')))
          const name = String(f.get('name')).trim(),
            bank = String(f.get('bank') ?? '').trim(),
            last4 = String(f.get('last4') ?? '')
          if (
            !name ||
            balance === null ||
            (card && (!bank || !/^\d{4}$/.test(last4)))
          ) {
            setError('Проверьте название, остаток и реквизиты карты.')
            return
          }
          setError('')
          setReview({ name, bank, last4, balance })
        }}
        className="space-y-4"
      >
        <div className="space-y-2">
          <Label htmlFor="new-account-name">Название</Label>
          <Input id="new-account-name" name="name" required maxLength={160} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="new-account-type">Тип счёта</Label>
          <Select
            value={type}
            onValueChange={(v) => {
              setType(v)
              setCard(v === 'debit' || v === 'credit')
            }}
          >
            <SelectTrigger id="new-account-type" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="debit">Дебетовый счёт</SelectItem>
              <SelectItem value="credit">Кредитный счёт</SelectItem>
              <SelectItem value="cash">Наличные</SelectItem>
              <SelectItem value="installment">Рассрочка</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {(type === 'debit' || type === 'credit') && (
          <label className="flex min-h-11 cursor-pointer items-center gap-3">
            <input
              type="checkbox"
              checked={card}
              onChange={(e) => setCard(e.target.checked)}
            />
            Есть карта
          </label>
        )}
        {card && (
          <>
            <div className="space-y-2">
              <Label htmlFor="new-bank">Банк</Label>
              <Input id="new-bank" name="bank" required maxLength={160} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-last4">Последние четыре цифры карты</Label>
              <Input
                id="new-last4"
                name="last4"
                inputMode="numeric"
                pattern="[0-9]{4}"
                maxLength={4}
                required
                placeholder="Например, 0012"
              />
            </div>
          </>
        )}
        <div className="space-y-2">
          <Label htmlFor="new-balance">Текущий остаток, ₽</Label>
          <Input
            id="new-balance"
            name="balance"
            inputMode="decimal"
            required
            placeholder="0,00"
          />
          <p className="text-sm text-muted-foreground">
            Для кредитного счёта задолженность обозначается отрицательным
            балансом.
          </p>
        </div>
        <div className="grace-actions">
          <Button type="submit">Проверить данные</Button>
        </div>
      </form>
      {review && (
        <>
          <dl className="grace-fields">
            <div>
              <dt>Название</dt>
              <dd>{review.name}</dd>
            </div>
            {card && (
              <>
                <div>
                  <dt>Банк</dt>
                  <dd>{review.bank}</dd>
                </div>
                <div>
                  <dt>Последние четыре цифры</dt>
                  <dd>{review.last4}</dd>
                </div>
              </>
            )}
            <div>
              <dt>Текущий остаток</dt>
              <dd>{formatMoney(review.balance, 'RUB')}</dd>
            </div>
          </dl>
          <div className="grace-actions">
            <Button disabled={pending} onClick={() => void save()}>
              {pending ? 'Сохранение…' : 'Создать счёт'}
            </Button>
            <Button
              variant="outline"
              disabled={pending}
              onClick={() => setReview(null)}
            >
              Изменить
            </Button>
          </div>
        </>
      )}
      {error && (
        <p role="alert" className="mt-4 text-destructive">
          {error}
        </p>
      )}
    </>
  )
}
