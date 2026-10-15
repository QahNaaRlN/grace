import { Disclosure } from '@/shared/ui/disclosure'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { importBatchQueries, useImportRefresh } from '@/entities/import-batch'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'

type Row = components['schemas']['ImportMonthRow']
const labels: Record<string, string> = {
  unresolved: 'Не проверено',
  create: 'Новая операция',
  link_existing: 'Связь с существующей операцией',
  duplicate_of: 'Дубль другой строки',
  exclude: 'Исключено',
  kind_changed: 'Изменён вид операции',
  amount_changed: 'Изменена сумма',
  date_changed: 'Изменена дата',
  currency_changed: 'Изменена валюта',
  target_unknown: 'Целевые данные неизвестны',
  purchase: 'Покупка',
  income_received: 'Доход',
  internal_transfer: 'Перевод между своими счетами',
  refund: 'Возврат',
  debt_payment: 'Погашение долга',
}
function money(value: number | null, currency: string) {
  if (value === null) return 'Неизвестно'
  if (!Number.isSafeInteger(value)) return 'Точная сумма доступна в исходнике'
  return `${(value / 100).toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${currency || 'валюта неизвестна'}`
}

export function MonthPanel({
  batchId,
  onOpenRecord,
}: {
  batchId: string
  onOpenRecord: (id: string) => void
}) {
  const refreshImports = useImportRefresh()
  const [params, setParams] = useSearchParams()
  const month = params.get('month') ?? ''
  const report = useQuery({ ...importBatchQueries.month(batchId, month), enabled: !!month })
  const showRow = (row: Row) => (
    <li key={row.recordId} className="space-y-1 rounded-lg border p-3">
      <Button
        className="h-auto min-h-11 max-w-full whitespace-normal text-left"
        variant="ghost"
        onClick={() => onOpenRecord(row.recordId)}
      >
        {row.sheet}, строка {row.row}: {row.description || 'Без описания'}
      </Button>
      <p className="text-sm">
        {row.sourceDate || 'Дата неизвестна'} · {money(row.sourceAmountMinor, row.sourceCurrency)}
      </p>
      <p className="text-sm">
        {labels[row.action]} · {row.published ? 'Подтверждено' : 'Ещё не подтверждено'}
      </p>
      {row.targetKind && (
        <p className="text-sm">
          После проверки: {row.targetDate} · {money(row.targetAmountMinor, row.targetCurrency)} ·{' '}
          {labels[row.targetKind] ?? row.targetKind}
        </p>
      )}
      {row.differences.length > 0 && (
        <p className="text-sm">{row.differences.map((d) => labels[d]).join(' · ')}</p>
      )}
      {row.reason && <p className="break-words text-sm">Причина: {row.reason}</p>}
    </li>
  )
  return (
    <Card>
      <CardHeader>
        <CardTitle>Перенос месяца расходов</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <Label htmlFor="import-month">Месяц для сверки</Label>
        <Input
          id="import-month"
          type="month"
          value={month}
          onChange={(event) => {
            const next = new URLSearchParams(params)
            if (event.target.value) next.set('month', event.target.value)
            else next.delete('month')
            setParams(next)
          }}
        />
        {!month && (
          <p>
            Выберите месяц из таблицы. Сверка учитывает весь черновик, включая дополнительные
            строки.
          </p>
        )}
        {report.isFetching && <p role="status">Обновляем сверку…</p>}
        {report.isError && (
          <p role="alert">
            Не удалось загрузить сверку.{' '}
            <Button variant="outline" onClick={() => void refreshImports.months()}>
              Повторить сверку
            </Button>
          </p>
        )}
        {report.data && !report.isError && (
          <>
            <p role="status" className="font-medium">
              {report.data.complete
                ? 'Все строки месяца подтверждены'
                : 'Перенос месяца ещё не завершён'}
            </p>
            <p>
              Строк месяца: {report.data.rows.length} · Не подтверждено: {report.data.pendingRows} ·
              Вне месяца: {report.data.outsidePeriod} · Без даты:{' '}
              {report.data.unknownDateRows.length}
            </p>
            <p className="text-sm text-muted-foreground">
              Исходный итог включает возможные дубли. Подготовленные решения ещё не меняют расходы.
              Подтверждённый итог учитывает уникальные покупки из строк этого месяца; переводы и
              погашения не добавляются к потреблению.
            </p>
            {report.data.totals.map((t) => (
              <div key={t.currency} className="space-y-1 rounded-lg bg-muted p-3">
                <p className="font-medium">
                  {t.currency || 'Валюта неизвестна'} · Исходных покупок: {t.sourceRows}
                </p>
                <p>В таблице: {money(t.sourceMinor, t.currency)}</p>
                <p>Подготовлено: {money(t.preparedMinor, t.currency)}</p>
                <p>Подтверждено: {money(t.confirmedMinor, t.currency)}</p>
              </div>
            ))}
            {report.data.rows.length === 0 && (
              <p>В выбранном месяце нет строк с известной датой.</p>
            )}
            {report.data.unknownDateRows.length > 0 && (
              <div>
                <p className="font-medium">
                  {report.data.unknownDateRows.some(
                    (row) => !row.published || row.action !== 'exclude',
                  )
                    ? 'Сначала уточните строки без даты: они могут относиться к этому месяцу.'
                    : 'Строки без даты исключены с подтверждёнными причинами и сохранены для проверки.'}
                </p>
                <ul className="mt-3 space-y-2">{report.data.unknownDateRows.map(showRow)}</ul>
              </div>
            )}
            <Disclosure
              triggerClassName="min-h-11 cursor-pointer py-3"
              title={<>Все строки месяца и причины расхождений ({report.data.rows.length})</>}
            >
              <ul className="space-y-2">{report.data.rows.map(showRow)}</ul>
            </Disclosure>
          </>
        )}
      </CardContent>
    </Card>
  )
}
