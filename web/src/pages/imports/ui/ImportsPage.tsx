import { PageHeader } from '@/shared/ui/page-header'
import { ImportReview } from '@/widgets/import-review'

export function ImportsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Импорт"
        description="Загрузите таблицу, проверьте строки и подтвердите результат."
      />
      <ImportReview />
    </div>
  )
}
