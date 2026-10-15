import { LoadingSkeleton } from '@/shared/ui/skeleton'
import { Categories } from '@/widgets/categories-list'
import { LedgerScreen } from '@/widgets/ledger-screen'

export function CategoriesPage() {
  return (
    <LedgerScreen title="Категории" gated={false}>
      {({ categories }) =>
        categories.isPending ? (
          <LoadingSkeleton label="Загружаем категории…" />
        ) : (
          categories.data && <Categories items={categories.data} />
        )
      }
    </LedgerScreen>
  )
}
