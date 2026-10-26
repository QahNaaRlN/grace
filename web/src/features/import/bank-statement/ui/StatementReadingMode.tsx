import { useState } from 'react'
import type { components } from '@/shared/api/generated/schema'
import { Button } from '@/shared/ui/button'
import { useStatementMutations } from '../api/mutations'

export function StatementReadingMode({
  statement: s,
}: {
  statement: components['schemas']['BankStatement']
}) {
  const { edit } = useStatementMutations(s.id)
  const [error, setError] = useState('')
  if (s.readingMode === 'plain_review' || s.deletedAt || s.publishedRows > 0 || !s.pages?.length)
    return null
  return (
    <section className="space-y-2">
      <Button
        variant="outline"
        disabled={edit.isPending}
        onClick={async () => {
          try {
            await edit.mutateAsync({
              expectedRevision: s.revision,
              accountId: s.accountId,
              retention: s.retention,
              readingMode: 'plain_review',
              rolesConfirmed: false,
              reviewedPages: [],
              reason: 'Перейти к простому чтению; предыдущие сверка и согласие отменены',
            })
            setError('')
          } catch (e) {
            setError(e instanceof Error ? e.message : 'Не удалось изменить режим.')
          }
        }}
      >
        Перейти к простой редактируемой таблице
      </Button>
      <p className="text-sm">
        Предыдущие сверка и согласие будут отменены. Денежные записи не изменятся.
      </p>
      {error && <p role="alert">{error}</p>}
    </section>
  )
}
