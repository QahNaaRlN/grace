import type { components } from '@/shared/api/generated/schema'
import { ungroupAmount } from '@/shared/lib/amount-text'
import { accountLabel } from '@/shared/lib/finance-display'
import { Button } from '@/shared/ui/button'
import { ChoiceSelect } from '@/shared/ui/choice-select'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { MoneyInput } from '@/shared/ui/money-input'
import { Textarea } from '@/shared/ui/textarea'
import type { ReceiptDraftFields as Fields } from '../api/receipt-drafts'

type Account = components['schemas']['Account']
type Category = components['schemas']['Category']

const emptyReceiptItem = {
  name: null,
  quantity: null,
  unitPrice: null,
  lineTotal: null,
  discount: null,
  brand: null,
  package: null,
  broadGroup: null,
}

/** The editable receipt draft: shop, date, total, account, category, items and the action. */
export function ReceiptDraftEditor({
  fields,
  change,
  accounts,
  categories,
  busy,
  onPrepare,
  onCancel,
}: {
  fields: Fields
  change: <K extends keyof Fields>(key: K, value: Fields[K]) => void
  accounts: Account[]
  categories: Category[]
  busy: boolean
  onPrepare: () => void
  onCancel: () => void
}) {
  return (
    <fieldset disabled={busy} className="space-y-4">
      <div className="flex flex-wrap gap-2"></div>
      <div>
        <Label htmlFor="receipt-seller">Магазин</Label>
        <Input
          id="receipt-seller"
          value={fields.seller ?? ''}
          onChange={(e) => change('seller', e.target.value || null)}
        />
      </div>
      <div>
        <Label htmlFor="receipt-date">Дата чека · время неизвестно</Label>
        <Input
          id="receipt-date"
          type="date"
          value={fields.issuedDate ?? ''}
          onChange={(e) => {
            change('issuedDate', e.target.value || null)
            change('issuedAt', null)
          }}
        />
      </div>
      <div>
        <Label htmlFor="receipt-currency">Валюта чека</Label>
        <ChoiceSelect
          id="receipt-currency"
          value={fields.currency ?? ''}
          onValueChange={(value) => change('currency', value || null)}
        >
          <option value="">Не определена</option>
          <option value="RUB">Российский рубль · RUB</option>
          {fields.currency && fields.currency !== 'RUB' && (
            <option value={fields.currency} disabled>
              {fields.currency} · пока не поддерживается
            </option>
          )}
        </ChoiceSelect>
      </div>
      <div>
        <Label htmlFor="receipt-total">Итог</Label>
        <MoneyInput
          id="receipt-total"
          currency={fields.currency ?? undefined}
          value={fields.total ?? ''}
          onChange={(e) => change('total', ungroupAmount(e.target.value) || null)}
        />
      </div>
      <div>
        <Label htmlFor="receipt-account">Счёт оплаты</Label>
        <ChoiceSelect
          id="receipt-account"
          value={fields.accountId}
          onValueChange={(value) => change('accountId', value)}
        >
          <option value="">Выбери счёт</option>
          {accounts
            .filter((a) => a.currency === 'RUB')
            .map((a) => (
              <option key={a.id} value={a.id}>
                {accountLabel(a)}
              </option>
            ))}
        </ChoiceSelect>
      </div>
      <div>
        <Label htmlFor="receipt-category">Категория</Label>
        <ChoiceSelect
          id="receipt-category"
          value={fields.categoryId ?? ''}
          onValueChange={(value) => change('categoryId', value || null)}
        >
          <option value="">Выбери категорию расхода</option>
          {categories
            .filter((c) => c.kind === 'expense' && (!c.hidden || c.id === fields.categoryId))
            .map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
        </ChoiceSelect>
      </div>
      <p className="text-sm font-medium">Позиции · неизвестные поля заполни по чеку</p>
      {fields.items.map((item, index) => (
        <div key={index} className="grid gap-2 rounded-lg border p-3 sm:grid-cols-2">
          {(
            [
              'name',
              'quantity',
              'unitPrice',
              'lineTotal',
              'discount',
              'brand',
              'package',
              'broadGroup',
            ] as const
          ).map((key, i) => (
            <div key={key}>
              <Label htmlFor={`receipt-${index}-${key}`}>
                {
                  [
                    'Название',
                    'Количество',
                    'Цена',
                    'Итог позиции',
                    'Скидка позиции',
                    'Бренд',
                    'Упаковка',
                    'Группа товара',
                  ][i]
                }
              </Label>
              {key === 'unitPrice' || key === 'lineTotal' || key === 'discount' ? (
                <MoneyInput
                  id={`receipt-${index}-${key}`}
                  currency="RUB"
                  value={item[key] ?? ''}
                  onChange={(e) =>
                    change(
                      'items',
                      fields.items.map((v, j) =>
                        j === index ? { ...v, [key]: ungroupAmount(e.target.value) || null } : v,
                      ),
                    )
                  }
                />
              ) : (
                <Input
                  id={`receipt-${index}-${key}`}
                  inputMode={key === 'quantity' ? 'decimal' : undefined}
                  value={item[key] ?? ''}
                  onChange={(e) =>
                    change(
                      'items',
                      fields.items.map((v, j) =>
                        j === index ? { ...v, [key]: e.target.value || null } : v,
                      ),
                    )
                  }
                />
              )}
            </div>
          ))}
          <Button
            variant="ghost"
            onClick={() =>
              change(
                'items',
                fields.items.filter((_, j) => j !== index),
              )
            }
          >
            Удалить позицию
          </Button>
        </div>
      ))}
      <Button
        variant="outline"
        disabled={fields.items.length >= 50}
        onClick={() => change('items', [...fields.items, { ...emptyReceiptItem }])}
      >
        Добавить позицию
      </Button>
      <div>
        <Label htmlFor="receipt-action">Как сохранить</Label>
        <ChoiceSelect
          id="receipt-action"
          value={fields.action}
          onValueChange={(value) => change('action', value)}
        >
          <option value="">Выбери действие</option>
          <option value="create">Создать новый расход</option>
          <option value="link_existing">Добавить чек к существующей покупке</option>
        </ChoiceSelect>
      </div>
      {fields.action === 'link_existing' && (
        <p className="text-sm">
          Выбери покупку из совпадений ниже. Для поиска сначала покажи результат с действием
          «Создать новый расход».
        </p>
      )}
      <div>
        <Label htmlFor="receipt-candidate-reason">
          Если похожая покупка уже есть, почему нужен новый расход
        </Label>
        <Textarea
          id="receipt-candidate-reason"
          value={fields.candidateReason}
          onChange={(e) => change('candidateReason', e.target.value)}
        />
      </div>
      <div className="flex flex-wrap gap-2">
        <Button onClick={onPrepare}>Проверить и показать результат</Button>
        <Button variant="ghost" onClick={onCancel}>
          Отменить черновик
        </Button>
      </div>
    </fieldset>
  )
}
