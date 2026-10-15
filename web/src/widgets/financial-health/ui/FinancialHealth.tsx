import { Link } from 'react-router-dom'
import { CircleAlert, ShieldCheck, CalendarClock, BarChart3 } from 'lucide-react'
import type {
  Account,
  Category,
  FinancialEvent,
  Obligation,
  Receipt,
  CashPosition,
} from '@/shared/api'
import { accountLabel } from '@/shared/lib/finance-display'
import { dateText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { AccountIcon } from '@/shared/ui/account-icon'
import { Button } from '@/shared/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/shared/ui/card'
import { Disclosure } from '@/shared/ui/disclosure'
import { Money } from '@/shared/ui/money'
import { Notice, StatusBadge } from '@/shared/ui/status'

export function FinancialHealth({
  accounts,
  categories,
  events,
  obligations,
  receipts,
  cash,
  incomplete = false,
}: {
  accounts: Account[]
  categories: Category[]
  events: FinancialEvent[]
  obligations: Obligation[]
  receipts: Receipt[]
  cash?: CashPosition
  incomplete?: boolean
}) {
  const unknown = accounts.filter((a) => a.balanceMinor === null || !a.balanceReconciled)
  const overdue = obligations.flatMap((o) =>
    o.payments
      .filter((p) => p.isOverdue)
      .map((p) => ({ ...p, name: o.name, currency: o.currency })),
  )
  return (
    <section className="space-y-6">
      <div className="rounded-2xl bg-secondary p-5 text-secondary-foreground">
        <p className="font-medium">
          Хватит ли денег на ближайшие обязательства и можно ли доверять расчёту?
        </p>
        <p className="mt-2 text-sm leading-relaxed">
          Здесь собраны известные платежи и пробелы в данных. Это проверка текущего положения, а не
          оценка в баллах. Отдельные валюты не складываются.
        </p>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CalendarClock aria-hidden="true" className="size-5 text-primary" />
              После обязательных платежей
            </CardTitle>
            <CardDescription>
              По известным платежам до {cash ? dateText(cash.through) : 'выбранной даты'}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {cash?.currencies.length ? (
              cash.currencies.map((c) => (
                <div key={c.currency} className="space-y-2">
                  <p>
                    <Money
                      valueMinor={c.availableAfterObligationsMinor}
                      currency={c.currency}
                      size="lg"
                    />
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Дебетовые счета и наличные: {formatMoney(c.liquidBalanceMinor, c.currency)} ·
                    Известные платежи: {formatMoney(c.obligationsThroughMinor, c.currency)}
                  </p>
                </div>
              ))
            ) : (
              <p>Расчёт пока недоступен. Проверьте счета и даты платежей.</p>
            )}
            <p className="text-sm text-muted-foreground">
              Будущие доходы, незапланированные расходы и кредитный лимит не учитываются.
              Несверенные остатки обозначены явно.
            </p>
            <Button variant="outline" asChild>
              <Link to="/analytics">Посмотреть расчёт</Link>
            </Button>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CircleAlert aria-hidden="true" className="size-5 text-primary" />
              Что требует внимания
            </CardTitle>
            <CardDescription>
              Сведения о просрочке рассчитаны Grace по графику платежей.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {overdue.length ? (
              overdue.map((p) => (
                <div
                  key={p.id}
                  className="flex flex-wrap justify-between gap-2 rounded-xl border p-3"
                >
                  <span className="flex flex-wrap items-center gap-2">
                    <StatusBadge tone="danger">Просрочен</StatusBadge>
                    {p.name} · {dateText(p.dueOn)}
                  </span>
                  <Money valueMinor={p.remainingMinor} currency={p.currency} />
                </div>
              ))
            ) : (
              <p>
                {incomplete
                  ? 'Часть данных недоступна. Статус просрочки нельзя подтвердить.'
                  : 'В загруженном графике просроченных платежей нет.'}
              </p>
            )}
            <p className="text-sm text-muted-foreground">
              Платежи вне Grace здесь не видны. Отметка оплаты создаёт финансовую операцию только
              после вашего подтверждения.
            </p>
            <Button variant="outline" asChild>
              <Link to="/payments">Проверить график</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck aria-hidden="true" className="size-5 text-primary" />
            Насколько полны данные
          </CardTitle>
          <CardDescription>
            Даже точный расчёт зависит от того, какие записи добавлены.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {unknown.length ? (
            <>
              <p>Остатки требуют сверки:</p>
              <div className="space-y-2">
                {unknown.map((a) => (
                  <div key={a.id} className="flex items-center gap-3 rounded-xl border p-3">
                    <AccountIcon type={a.type} />
                    <span className="flex-1">{accountLabel(a)}</span>
                    <StatusBadge tone="unknown">Не сверён</StatusBadge>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <p>
              {incomplete
                ? 'Сведения загружены частично. Проверьте доступность счетов.'
                : accounts.length
                  ? 'Все загруженные остатки отмечены как сверенные.'
                  : 'Счета ещё не добавлены.'}
            </p>
          )}
          <Notice tone="unknown" title="Полнота истории не подтверждена">
            Ноль в отчёте означает отсутствие записей, а не доказанное отсутствие расходов. Импорт в
            черновике не входит в расчёты.
          </Notice>
          <div className="flex flex-wrap gap-3">
            <Button variant="outline" asChild>
              <Link to="/accounts">Проверить счета</Link>
            </Button>
            <Button variant="ghost" asChild>
              <Link to="/imports">Проверить импорт</Link>
            </Button>
          </div>
          <Disclosure title="Какие сведения используются">
            <dl className="grace-fields">
              <div>
                <dt>Операции</dt>
                <dd>{events.length} последних загруженных записей</dd>
              </div>
              <div>
                <dt>Категории</dt>
                <dd>{categories.length}</dd>
              </div>
              <div>
                <dt>Подтверждённые чеки</dt>
                <dd>{receipts.length} последних загруженных записей</dd>
              </div>
            </dl>
            <p className="mt-3 text-sm text-muted-foreground">
              Доход от аренды или другое поступление не считается зарплатой автоматически. Кредитные
              льготные периоды и условия договоров здесь пока не анализируются.
            </p>
          </Disclosure>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 aria-hidden="true" className="size-5 text-primary" />
            Куда уходят деньги
          </CardTitle>
          <CardDescription>
            Сравните расходы по категориям и периодам в статистике. Для объяснения подтверждённых
            итогов можно обратиться к ИИ.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <Button asChild>
            <Link to="/analytics">Открыть статистику</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link to="/assistant">Обсудить расходы</Link>
          </Button>
        </CardContent>
      </Card>
    </section>
  )
}
