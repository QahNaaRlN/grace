import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'

import { importBatchQueries } from '@/entities/import-batch'
import { accountQueries } from '@/entities/account'
import {
  bankStatementQueries,
  statementAttemptQueries,
  SourceText,
  StatementReconciliation,
} from '@/entities/bank-statement'
import {
  StatementManualDraft,
  StatementAI,
  StatementOwnerReview,
  StatementUpload,
} from '@/features/import/bank-statement'
import { env } from '@/shared/config'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Disclosure } from '@/shared/ui/disclosure'

export function BankStatementImport() {
  const [params, setParams] = useSearchParams()
  const selectedBatch = params.get('batch') ?? ''
  const batch = useQuery({
    ...importBatchQueries.batch(selectedBatch),
    enabled: !!selectedBatch && !params.get('statement'),
  })
  const id = params.get('statement') ?? batch.data?.statementId ?? ''
  const accounts = useQuery(accountQueries.importOptions())
  const source = useQuery({ ...bankStatementQueries.detail(id), enabled: !!id })
  const attempts = useQuery({
    ...statementAttemptQueries.list(id),
    enabled: !!id,
    refetchInterval: (query) =>
      query.state.data?.some((a) => a.status === 'running') ? 2000 : false,
  })
  const s = source.data
  return (
    <Card>
      <CardHeader>
        <CardTitle>PDF-выписка</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        <Disclosure title="Загрузить текстовый PDF" defaultOpen={!id}>
          <StatementUpload
            accounts={(accounts.data?.items ?? []).filter(
              (a) => a.type === 'debit' && a.currency === 'RUB',
            )}
            onSaved={(next) => {
              const search = new URLSearchParams(params)
              search.set('statement', next)
              search.delete('batch')
              search.delete('confirmation')
              setParams(search)
            }}
          />
        </Disclosure>
        {source.isError && <p role="alert">{source.error.message}</p>}
        {s && (
          <>
            <p className="break-all font-medium">
              {s.fileName} · {s.pageCount} стр.
            </p>
            {s.review && <StatementReconciliation review={s.review} />}
            <StatementOwnerReview key={`owner:${s.id}`} statement={s} />
            {!s.deletedAt && (
              <Button asChild variant="outline">
                <a href={`${env.VITE_API_BASE_URL}/api/imports/statements/${s.id}/original`}>
                  Скачать исходный PDF
                </a>
              </Button>
            )}
            {s.layout && (
              <Disclosure title="Текст исходника">
                <SourceText layout={s.layout} />
              </Disclosure>
            )}
            <StatementAI key={`ai:${s.id}:${s.revision}`} statement={s} />
            {(attempts.data?.length ?? 0) > 0 && (
              <p className="text-sm">
                Попытки распознавания:{' '}
                {attempts.data
                  ?.map(
                    (a) =>
                      ({
                        running: 'Выполняется',
                        completed: 'Черновик сохранён',
                        failed: 'Не завершено',
                        uncertain: 'Стоимость требует проверки',
                        cancelled: 'Отменено',
                      })[a.status] ?? a.status,
                  )
                  .join('; ')}
              </p>
            )}
            <StatementManualDraft key={`manual:${s.id}`} statement={s} />
            {s.batchId && (
              <Button
                onClick={() => {
                  const search = new URLSearchParams(params)
                  search.set('batch', s.batchId!)
                  setParams(search)
                }}
              >
                Проверить строки и подготовить подтверждение
              </Button>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
