import type { components } from '@/shared/api/generated/schema'
import { APP_UTC_OFFSET } from '@/shared/config'
import { eventTimeText } from '@/shared/lib/date'
import { accountLabel } from '@/shared/lib/finance-display'
import { parseMinor } from '@/shared/lib/money'
import { importLabels } from '@/entities/import-batch'
import { Checkbox } from '@/shared/ui/checkbox'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { CurrencySelect } from '@/shared/ui/currency-select'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'

type Record = components['schemas']['ImportRecord']
type Fields = components['schemas']['ImportFields']
type Account = components['schemas']['Account']
type Category = components['schemas']['Category']
type FinancialEvent = components['schemas']['FinancialEvent']

/** The operation an import row creates: kind, amount, date, accounts, category, texts. */
export function ImportFieldsEditor({
  record,
  fields,
  setFields,
  amountText,
  setAmountText,
  accounts,
  categories,
  events,
}: {
  record: Record
  fields: Fields
  setFields: (next: Fields) => void
  amountText: string
  setAmountText: (text: string) => void
  accounts: Account[]
  categories: Category[]
  events: FinancialEvent[]
}) {
  function field<K extends keyof Fields>(name: K, value: Fields[K]) {
    setFields({ ...fields, [name]: value })
  }
  const chooseAccount = (name: 'accountId' | 'destinationAccountId', label: string) => (
    <div>
      <Label htmlFor={name}>{label}</Label>
      <ChoiceSelect
        id={name}
        value={fields[name] || ''}
        onValueChange={(value) => field(name, value)}
      >
        <option value="">Выберите счёт</option>
        {accounts.map((a) => (
          <option key={a.id} value={a.id}>
            {accountLabel(a)} ({a.currency})
          </option>
        ))}
      </ChoiceSelect>
    </div>
  )
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <div>
        <Label htmlFor="kind">Вид операции</Label>
        <ChoiceSelect
          id="kind"
          value={fields.kind}
          onValueChange={(value) => field('kind', value as Fields['kind'])}
        >
          {['purchase', 'income_received', 'debt_payment', 'internal_transfer', 'refund'].map(
            (k) => (
              <option key={k} value={k}>
                {importLabels[k]}
              </option>
            ),
          )}
        </ChoiceSelect>
      </div>
      <div>
        <Label htmlFor="amount">Сумма</Label>
        <MoneyInput
          id="amount"
          currency={/^[A-Z]{3}$/.test(fields.currency) ? fields.currency : undefined}
          value={amountText}
          onChange={(e) => {
            setAmountText(e.target.value)
            field('amountMinor', parseMinor(e.target.value) ?? Number.NaN)
          }}
        />
      </div>
      <div>
        <Label htmlFor="currency">Валюта</Label>
        <CurrencySelect
          id="currency"
          value={fields.currency}
          onValueChange={(value) => field('currency', value)}
        />
      </div>
      <div className="sm:col-span-2">
        <label className="flex min-h-11 items-center gap-3">
          <Checkbox
            checked={fields.occurredDate !== undefined}
            onCheckedChange={(checked) =>
              setFields({
                ...fields,
                occurredDate: checked
                  ? record.source.normalized.date?.local.slice(0, 10) || ''
                  : undefined,
                occurredAt: checked ? undefined : '',
              })
            }
          />
          Известен только день, время неизвестно
        </label>
      </div>
      {fields.occurredDate !== undefined ? (
        <div>
          <Label htmlFor="occurredDate">Дата операции</Label>
          <Input
            id="occurredDate"
            type="date"
            value={fields.occurredDate}
            onChange={(e) => field('occurredDate', e.target.value)}
          />
        </div>
      ) : (
        <>
          <div>
            <Label htmlFor="occurredAt">
              Дата и время (с часовым поясом, например {APP_UTC_OFFSET})
            </Label>
            <Input
              id="occurredAt"
              placeholder={`2026-10-06T12:30:00${APP_UTC_OFFSET}`}
              value={fields.occurredAt || ''}
              onChange={(e) => field('occurredAt', e.target.value)}
            />
          </div>
          <div>
            <Label htmlFor="timeReason">Основание уточнения времени</Label>
            <Input
              id="timeReason"
              value={fields.timeReason || ''}
              onChange={(e) => field('timeReason', e.target.value)}
            />
          </div>
        </>
      )}
      {chooseAccount('accountId', 'Счёт операции')}
      {fields.kind === 'internal_transfer' &&
        chooseAccount('destinationAccountId', 'Счёт получателя')}
      <div>
        <Label htmlFor="categoryId">Категория</Label>
        <ChoiceSelect
          id="categoryId"
          value={fields.categoryId || ''}
          onValueChange={(value) => field('categoryId', value || undefined)}
        >
          <option value="">Без категории</option>
          {categories
            .filter((c) => !c.hidden || c.id === fields.categoryId)
            .map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
        </ChoiceSelect>
      </div>
      {fields.kind === 'refund' && (
        <div>
          <Label htmlFor="originalEventId">Исходная покупка</Label>
          <ChoiceSelect
            id="originalEventId"
            value={fields.originalEventId || ''}
            onValueChange={(value) => field('originalEventId', value)}
          >
            <option value="">Выберите покупку</option>
            {events
              .filter((e) => e.kind === 'purchase')
              .map((e) => (
                <option key={e.id} value={e.id}>
                  {eventTimeText(e)} · {e.merchant || e.description}
                </option>
              ))}
          </ChoiceSelect>
        </div>
      )}
      <div>
        <Label htmlFor="merchant">Получатель</Label>
        <Input
          id="merchant"
          value={fields.merchant}
          onChange={(e) => field('merchant', e.target.value)}
        />
      </div>
      <div>
        <Label htmlFor="description">Описание</Label>
        <Input
          id="description"
          value={fields.description}
          onChange={(e) => field('description', e.target.value)}
        />
      </div>
    </div>
  )
}
