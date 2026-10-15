import { Disclosure } from '@/shared/ui/disclosure'
import { useRef, useState } from 'react'
import { toast } from 'sonner'
import type { Account, Category, FinancialEvent } from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { formatMoney, parseMinor } from '@/shared/lib/money'
import { createIdempotencyKey } from '@/shared/lib/idempotency'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { MoneyInput } from '@/shared/ui/money-input'
import { Label } from '@/shared/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'
import { useCreateFinancialEvent } from '../api/use-create-financial-event'

const kinds = {
  purchase: 'Покупка',
  income: 'Доход',
  transfer: 'Перевод',
  refund: 'Возврат',
  debt_payment: 'Платёж по обязательству',
}
type Kind = keyof typeof kinds
type Review = {
  amountMinor: number
  merchant: string
  description: string
  occurredAt: string
  key: string
}
export function CreateFinancialEventForm({
  accounts,
  categories,
  events = [],
  onSaved,
}: {
  accounts: Account[]
  categories: Category[]
  events?: FinancialEvent[]
  onSaved?: () => void
}) {
  const create = useCreateFinancialEvent()
  const [initialDate] = useState(() => {
    const now = new Date()
    return new Date(now.valueOf() - now.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
  })
  const [kind, setKind] = useState<Kind>('purchase')
  const [account, setAccount] = useState(accounts[0]?.id ?? '')
  const [target, setTarget] = useState(accounts[1]?.id ?? '')
  const [category, setCategory] = useState(
    categories.find((c) => c.kind === 'expense')?.id ?? 'none',
  )
  const [original, setOriginal] = useState('')
  const [review, setReview] = useState<Review | null>(null)
  const [failure, setFailure] = useState('')
  const [pending, setPending] = useState(false)
  const inFlight = useRef(false)
  const selected = accounts.find((a) => a.id === account)
  const originalPurchase = events.find((e) => e.id === original)
  const currency = kind === 'refund' ? originalPurchase?.postings[0]?.currency : selected?.currency
  const categoriesVisible = categories.filter(
    // SPEC-0042: hidden categories are not offered for new operations.
    (c) => c.kind === (kind === 'income' ? 'income' : 'expense') && !c.hidden,
  )
  const eventAccount =
    kind === 'refund'
      ? accounts.find((a) => a.id === originalPurchase?.postings[0]?.accountId)
      : selected
  async function save() {
    if (!review || inFlight.current) return
    inFlight.current = true
    setPending(true)
    setFailure('')
    try {
      const { error } = await create.mutateAsync({
        kind,
        key: review.key,
        common: {
          amountMinor: review.amountMinor,
          merchant: review.merchant,
          description: review.description,
          occurredAt: review.occurredAt,
        },
        accountId: account,
        categoryId: category === 'none' ? null : category,
        originalEventId: original,
        destinationAccountId: target,
      })
      if (error) {
        setFailure(error.message)
        return
      }
      toast.success('Операция сохранена')
      onSaved?.()
    } catch {
      setFailure(
        'Не удалось получить ответ. Повтор этого подтверждения использует тот же ключ и не создаст вторую операцию.',
      )
    } finally {
      inFlight.current = false
      setPending(false)
    }
  }
  return (
    <>
      <form
        hidden={!!review}
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault()
          const f = new FormData(e.currentTarget)
          const amountMinor = parseMinor(String(f.get('amount'))),
            merchant = String(f.get('merchant') ?? '').trim(),
            description = String(f.get('description') ?? '').trim(),
            date = new Date(String(f.get('occurredAt')))
          if (
            amountMinor === null ||
            amountMinor <= 0 ||
            !currency ||
            Number.isNaN(date.valueOf())
          ) {
            setFailure('Проверьте сумму, счёт и дату.')
            return
          }
          if (kind === 'purchase' && (category === 'none' || (!merchant && !description))) {
            setFailure('Выберите категорию и укажите описание или получателя.')
            return
          }
          if (kind === 'transfer' && (account === target || !target)) {
            setFailure('Выберите два разных счёта.')
            return
          }
          if (kind === 'refund' && !originalPurchase) {
            setFailure('Выберите исходную покупку.')
            return
          }
          setFailure('')
          setReview({
            amountMinor,
            merchant,
            description,
            occurredAt: date.toISOString(),
            key: createIdempotencyKey(),
          })
        }}
      >
        <div className="space-y-2">
          <Label htmlFor="event-kind">Тип операции</Label>
          <Select
            value={kind}
            onValueChange={(v) => {
              setKind(v as Kind)
              setCategory('none')
            }}
          >
            <SelectTrigger id="event-kind" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {Object.entries(kinds).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="event-amount">Сумма</Label>
          <MoneyInput
            name="amount"
            id="event-amount"
            currency={currency || undefined}
            required
            placeholder="0,00"
          />
        </div>
        {kind !== 'refund' && (
          <div className="space-y-2">
            <Label htmlFor="event-account">
              {kind === 'transfer' ? 'Со счёта' : 'Счёт / карта'}
            </Label>
            <Select value={account} onValueChange={setAccount}>
              <SelectTrigger id="event-account" className="w-full">
                <SelectValue placeholder="Выберите счёт" />
              </SelectTrigger>
              <SelectContent>
                {accounts.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {accountLabel(a)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {kind === 'transfer' && (
          <div className="space-y-2">
            <Label htmlFor="event-target">На счёт</Label>
            <Select value={target} onValueChange={setTarget}>
              <SelectTrigger id="event-target" className="w-full">
                <SelectValue placeholder="Выберите счёт" />
              </SelectTrigger>
              <SelectContent>
                {accounts.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {accountLabel(a)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {kind === 'refund' && (
          <div className="space-y-2">
            <Label htmlFor="event-original">Исходная покупка</Label>
            <Select value={original} onValueChange={setOriginal}>
              <SelectTrigger id="event-original" className="w-full">
                <SelectValue placeholder="Выберите покупку" />
              </SelectTrigger>
              <SelectContent>
                {events
                  .filter((e) => e.kind === 'purchase')
                  .map((e) => (
                    <SelectItem key={e.id} value={e.id}>
                      {e.merchant || e.description || 'Покупка'} ·{' '}
                      {formatMoney(e.postings[0].amountMinor, e.postings[0].currency)}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <div className="space-y-2">
          <Label htmlFor="event-date">Дата и время (часовой пояс устройства)</Label>
          <Input
            name="occurredAt"
            id="event-date"
            type="datetime-local"
            required
            defaultValue={initialDate}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="event-description">Описание / комментарий</Label>
          <Input name="description" id="event-description" maxLength={500} />
        </div>
        <Disclosure
          className="rounded-xl border p-3"
          triggerClassName="min-h-11 rounded-lg px-3 py-2 font-medium hover:bg-accent"
          title={<>Дополнительные сведения</>}
        >
          <div className="mt-4 space-y-4">
            {kind !== 'transfer' && kind !== 'refund' && (
              <div className="space-y-2">
                <Label htmlFor="event-category">Категория</Label>
                <Select value={category} onValueChange={setCategory}>
                  <SelectTrigger id="event-category" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Не указана</SelectItem>
                    {categoriesVisible.map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            {kind !== 'transfer' && (
              <div className="space-y-2">
                <Label htmlFor="event-merchant">
                  {kind === 'income' ? 'Источник' : 'Получатель'}
                </Label>
                <Input name="merchant" id="event-merchant" maxLength={160} />
              </div>
            )}
          </div>
        </Disclosure>
        <div className="grace-actions">
          <Button type="submit" disabled={accounts.length === 0}>
            Посмотреть подтверждение
          </Button>
        </div>
      </form>
      {review && (
        <>
          <dl className="grace-fields">
            <div>
              <dt>Тип</dt>
              <dd>{kinds[kind]}</dd>
            </div>
            <div>
              <dt>Сумма</dt>
              <dd>{formatMoney(review.amountMinor, currency!)}</dd>
            </div>
            <div>
              <dt>Счёт / карта</dt>
              <dd>{eventAccount ? accountLabel(eventAccount) : 'Не указано'}</dd>
            </div>
            {kind === 'transfer' && (
              <div>
                <dt>На счёт</dt>
                <dd>
                  {accounts.find((a) => a.id === target)
                    ? accountLabel(accounts.find((a) => a.id === target)!)
                    : 'Не указано'}
                </dd>
              </div>
            )}
            <div>
              <dt>Дата</dt>
              <dd>{new Date(review.occurredAt).toLocaleString('ru-RU')}</dd>
            </div>
            <div>
              <dt>Категория</dt>
              <dd>{categories.find((c) => c.id === category)?.name ?? 'Не указано'}</dd>
            </div>
            <div>
              <dt>Получатель</dt>
              <dd>{review.merchant || 'Не указано'}</dd>
            </div>
            <div>
              <dt>Комментарий</dt>
              <dd>{review.description || 'Не указано'}</dd>
            </div>
          </dl>
          <div className="grace-actions">
            <Button disabled={pending} onClick={() => void save()}>
              {pending ? 'Сохранение…' : 'Подтвердить и сохранить'}
            </Button>
            <Button variant="outline" disabled={pending} onClick={() => setReview(null)}>
              Изменить
            </Button>
          </div>
        </>
      )}
      {failure && (
        <p role="alert" className="mt-4 text-destructive">
          {failure}
        </p>
      )}
    </>
  )
}
