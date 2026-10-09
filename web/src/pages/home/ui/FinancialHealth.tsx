import { eventTimeText } from '@/shared/lib/finance-display'
import { Link } from 'react-router-dom'
import type {
  Account,
  Category,
  FinancialEvent,
  Obligation,
  Receipt,
} from '@/shared/api/types'
import {
  accountLabel,
  formatMoney,
  dateText,
} from '@/shared/lib/finance-display'
const columns: Record<string, string[]> = {
  Сводка: [
    'Анализируемый месяц',
    'Зарплата',
    'Все расходы',
    'За свои деньги',
    'По кредитным инструментам',
    'Погашено карт',
    'Платежи по другим долгам',
    'Остаток потока месяца',
    'Для grace в выбранном месяце',
    'Уже оплачено',
    'Осталось внести',
    'Срочных / просроченных выписок',
    'Ошибок источника в операциях',
    'Источник расходов',
    'Расходы месяца, ₽',
    'Доля расходов',
    'Комментарий',
    'Категория',
    'Факт, ₽',
    'Лимит, ₽',
    'Отклонение, ₽',
  ],
  Операции: [
    'Дата',
    'Тип',
    'Описание',
    'Категория',
    'Подкатегория',
    'Сумма, ₽',
    'Источник финансирования',
    'Счёт / карта',
    'Приоритет',
    'Регулярный?',
    'Получатель / долг',
    'Комментарий',
    'Месяц',
    'Проверка источника',
    'ID',
  ],
  'Кредитные карты': [
    'Инструмент',
    'Организация',
    'Тип',
    'Лимит, ₽',
    'Текущая задолженность, ₽',
    'Доступно, ₽',
    'Начало текущего периода',
    'Конец текущего периода',
    'Дата платежа',
    'Нужно внести, ₽',
    'Мин. платёж, ₽',
    'Статус',
    'Комментарий',
  ],
  'Выписки карт': [
    'Карта',
    'Начало периода',
    'Конец периода',
    'Дата платежа',
    'Покупки периода, ₽',
    'Для сохранения grace, ₽',
    'Оплачено, ₽',
    'Осталось внести, ₽',
    'Мин. платёж, ₽',
    'Статус',
    'Комментарий',
  ],
  Зарплата: [
    'Часть зарплаты',
    'Базовая дата',
    'Плановая рабочая дата',
    'Плановая сумма, ₽',
    'Фактически получено, ₽',
    'Отклонение, ₽',
    'Официальные нерабочие дни',
  ],
  Задолженности: [
    'Приоритет',
    'Кредитор / долг',
    'Тип',
    'Остаток, ₽',
    'Ставка, %',
    'Мин. платёж, ₽',
    'Доп. платёж, ₽',
    'Плановый платёж, ₽',
    'Дата платежа',
    'Просрочка?',
    'Статус',
    'Оценка месяцев',
    'Комментарий',
  ],
  Категории: [
    'Категория',
    'Типичная роль',
    'Лимит, ₽',
    'Подкатегория (пример)',
    'Приоритет расходов',
    'Пояснение',
  ],
  'Сверка счетов': [
    'Счёт / инструмент',
    'Начальный остаток по выписке, ₽',
    'Поступления, ₽',
    'Списания, ₽',
    'Конечный остаток по выписке, ₽',
    'Контроль',
    'Комментарий',
  ],
  'Первичные документы': [
    '№',
    'Файл',
    'Тип',
    'Группа',
    'Сообщение',
    'Статус',
    'Комментарий',
  ],
  Инструкция: ['Шаг', 'Что сделать'],
}
const eventLabels: Record<FinancialEvent['kind'], string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
  debt_payment: 'Платёж по обязательству',
}
const debtLabels: Record<Obligation['kind'], string> = {
  loan: 'Кредит',
  installment: 'Рассрочка',
  other_fixed: 'Другое обязательство',
}
type RecordRow = { title: string; values: Record<string, string> }
export function FinancialHealth({
  accounts,
  categories,
  events,
  obligations,
  receipts,
}: {
  accounts: Account[]
  categories: Category[]
  events: FinancialEvent[]
  obligations: Obligation[]
  receipts: Receipt[]
}) {
  const sections: Record<string, RecordRow[]> = {
    'Кредитные карты': accounts
      .filter((a) => a.type === 'credit')
      .map((a) => ({
        title: accountLabel(a),
        values: {
          Инструмент: accountLabel(a),
          Организация: a.paymentInstruments?.[0]?.bank ?? 'Не указано',
          Тип: 'Кредитный счёт',
        },
      })),
    Операции: events.map((e) => ({
      title: e.merchant || e.description || e.id,
      values: {
        ID: e.id,
        Дата: eventTimeText(e),
        Тип: eventLabels[e.kind],
        Описание: e.description,
        Категория: e.categoryName ?? 'Не указано',
        'Сумма, ₽': e.postings[0]
          ? formatMoney(e.postings[0].amountMinor, e.postings[0].currency)
          : 'Не указано',
        'Счёт / карта': e.postings
          .map((p) => {
            const a = accounts.find((a) => a.id === p.accountId)
            return a ? accountLabel(a) : p.accountName
          })
          .join(' → '),
        'Получатель / долг': e.merchant,
        Комментарий: e.description,
        Месяц: new Intl.DateTimeFormat('ru-RU', {
          timeZone: 'Asia/Yekaterinburg',
          month: 'long',
          year: 'numeric',
        }).format(new Date(e.occurredAt ?? `${e.occurredDate}T00:00:00Z`)),
      },
    })),
    // A general income event does not establish a salary component.
    Зарплата: [],
    Задолженности: obligations.map((o) => ({
      title: o.name,
      values: {
        'Кредитор / долг': o.name,
        Тип: debtLabels[o.kind],
        'Остаток, ₽': formatMoney(o.remainingMinor, o.currency),
        Статус: o.status === 'completed' ? 'Завершено' : 'Действует',
      },
    })),
    Категории: categories.map((c) => ({
      title: c.name,
      values: {
        Категория: c.name,
        'Типичная роль': c.kind === 'expense' ? 'Расходы' : 'Доходы',
      },
    })),
    'Первичные документы': receipts.map((r) => ({
      title: r.seller,
      values: {
        '№': r.id,
        Тип: 'Подтверждённый чек',
        Статус: 'Подтверждён',
        Комментарий: `${r.seller} · ${r.issuedDate ? `${dateText(r.issuedDate)} · время неизвестно` : r.issuedAt ? dateText(r.issuedAt) : 'Дата неизвестна'} · ${formatMoney(r.totalMinor, r.currency)}`,
      },
    })),
  }
  return (
    <section className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Сведения в структуре таблицы «Финансовое здоровье». Отсутствующие данные
        обозначены «Не указано».
      </p>
      {Object.entries(columns).map(([section, fields]) => (
        <details key={section} className="rounded-2xl border bg-card p-5">
          <summary className="min-h-11 rounded-xl px-3 py-2 font-semibold hover:bg-accent">
            {section}
          </summary>
          <div className="mt-5 space-y-4">
            {section === 'Сводка' && (
              <Link
                className="inline-flex min-h-11 items-center rounded-xl px-3 text-primary hover:bg-accent"
                to="/analytics"
              >
                Открыть сводку и расчёт доступности
              </Link>
            )}
            {(sections[section]?.length
              ? sections[section]
              : [{ title: 'Сведения не указаны', values: {} }]
            ).map((record, i) => (
              <details key={i} open={!sections[section]?.length}>
                <summary className="min-h-11 rounded-xl px-3 py-2 hover:bg-accent">
                  {record.title}
                </summary>
                <dl className="grace-fields mt-3">
                  {fields.map((field) => (
                    <div key={field}>
                      <dt>{field}</dt>
                      <dd>{record.values[field] || 'Не указано'}</dd>
                    </div>
                  ))}
                </dl>
              </details>
            ))}
          </div>
        </details>
      ))}
    </section>
  )
}
