import { PageHeader } from '@/shared/ui/page-header'
import { BankStatementImport } from '@/widgets/bank-statement-import'
import { ImportReview } from '@/widgets/import-review'

export function ImportsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Импорт"
        description="Загрузите выписку или таблицу, проверьте строки и подтвердите результат."
      />
      <BankStatementImport />
      <ImportReview />
    </div>
  )
}
