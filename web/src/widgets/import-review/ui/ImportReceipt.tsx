import { Link } from 'react-router-dom'

import { importLabels } from '@/entities/import-batch'
import type { ImportReceipt as Receipt } from '@/features/import/confirm'
import { eventTimeText } from '@/shared/lib/date'
import { formatMoney } from '@/shared/lib/money'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Disclosure } from '@/shared/ui/disclosure'

/** The outcomes of a confirmation, with a link to every operation it wrote. */
export function ImportReceipt({ batchId, receipt }: { batchId: string; receipt: Receipt }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Подтверждение сохранено</CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="space-y-3">
          {receipt.outcomes.map((o) => (
            <li key={o.recordId}>
              {importLabels[o.action]}{' '}
              {o.event && (
                <Disclosure
                  id={`event-${o.eventId}`}
                  className="rounded-md border p-3"
                  triggerClassName="cursor-pointer py-2"
                  title={
                    <>
                      {importLabels[o.event.kind]}: {o.event.merchant || o.event.description} ·{' '}
                      {eventTimeText(o.event)}
                    </>
                  }
                >
                  <p>{o.event.description}</p>
                  {o.event.postings.map((p) => (
                    <p key={p.id}>
                      {p.accountName}: {p.direction === 'debit' ? 'Списание' : 'Зачисление'}{' '}
                      {formatMoney(p.amountMinor, p.currency)}
                    </p>
                  ))}
                  <Link
                    className="underline"
                    to={`/imports?batch=${batchId}&confirmation=${receipt.id}#event-${o.eventId}`}
                  >
                    Ссылка на подтверждённую операцию
                  </Link>
                </Disclosure>
              )}
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
