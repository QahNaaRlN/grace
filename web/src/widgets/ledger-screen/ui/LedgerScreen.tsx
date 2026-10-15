import { useQuery } from '@tanstack/react-query'
import { Plus, RefreshCw, WalletCards } from 'lucide-react'
import { useState, type ReactNode } from 'react'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries } from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import { obligationQueries } from '@/entities/obligation'
import { receiptQueries } from '@/entities/receipt'
import { CreateAccountForm } from '@/features/create-account'
import { CreateFinancialEventForm } from '@/features/create-financial-event'
import { browserMonthEnd, today } from '@/shared/lib/date'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { PageHeader } from '@/shared/ui/page-header'
import { LoadingSkeleton } from '@/shared/ui/skeleton'

function useLedger() {
  const accounts = useQuery(accountQueries.list())
  const categories = useQuery(categoryQueries.list())
  const events = useQuery(financialEventQueries.list())
  const receipts = useQuery(receiptQueries.list())
  const obligations = useQuery(obligationQueries.list(today()))
  const cash = useQuery(cashPositionQueries.through(browserMonthEnd()))
  return { accounts, categories, events, receipts, obligations, cash }
}

export type Ledger = ReturnType<typeof useLedger> & {
  /** At least one ledger query failed; the screen already shows the retry banner. */
  incomplete: boolean
}

/**
 * Common frame of the ledger screens: header with the create actions, the partial-load
 * banner, and the loading and empty-ledger states shared by every money page.
 */
export function LedgerScreen({
  title,
  description,
  accountAction = 'never',
  operationAction = false,
  gated = true,
  children,
}: {
  title: string
  description?: string
  /** SPEC-0036: offer an account only where money lives, never on profile or settings pages. */
  accountAction?: 'always' | 'when-empty' | 'never'
  operationAction?: boolean
  /** Wait for the ledger and show the empty-ledger invitation before the content. */
  gated?: boolean
  children: (ledger: Ledger) => ReactNode
}) {
  const [operationOpen, setOperationOpen] = useState(false)
  const [accountOpen, setAccountOpen] = useState(false)
  const queries = useLedger()
  const { accounts, categories, events } = queries
  const all = Object.values(queries)
  const error = all.find((q) => q.isError)?.error
  const pending = all.some((q) => q.isPending)
  const empty = accounts.isSuccess && accounts.data.length === 0
  const ledger = { ...queries, incomplete: Boolean(error) }
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <PageHeader
          title={title}
          description={description}
          status={error ? 'Часть данных недоступна' : pending ? 'Обновляем данные…' : undefined}
        />
        <div className="flex flex-wrap gap-3">
          <Dialog open={accountOpen} onOpenChange={setAccountOpen}>
            {(accountAction === 'always' || (empty && accountAction === 'when-empty')) && (
              <DialogTrigger asChild>
                <Button>
                  <Plus className="size-4" />
                  Добавить {empty ? 'первый ' : ''}счёт
                </Button>
              </DialogTrigger>
            )}
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Новый счёт</DialogTitle>
                <DialogDescription>Название, реквизиты и текущий остаток</DialogDescription>
              </DialogHeader>
              <CreateAccountForm onSaved={() => setAccountOpen(false)} />
            </DialogContent>
          </Dialog>
          {operationAction && !empty && (
            <Dialog open={operationOpen} onOpenChange={setOperationOpen}>
              <DialogTrigger asChild>
                <Button
                  disabled={!accounts.data?.length || categories.isPending || categories.isError}
                >
                  <Plus className="size-4" />
                  Добавить операцию
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Добавить операцию</DialogTitle>
                  <DialogDescription>Проверьте данные перед подтверждением</DialogDescription>
                </DialogHeader>
                {accounts.data && categories.data && (
                  <CreateFinancialEventForm
                    accounts={accounts.data}
                    categories={categories.data}
                    events={events.data ?? []}
                    onSaved={() => setOperationOpen(false)}
                  />
                )}
              </DialogContent>
            </Dialog>
          )}
        </div>
      </div>
      {error && (
        <div
          role="alert"
          className="flex flex-wrap items-center gap-4 rounded-xl border bg-card p-4"
        >
          <div className="min-w-0 flex-1">
            <strong>Обзор загружен частично</strong>
            <p className="mt-1 text-sm text-muted-foreground">{error.message}</p>
          </div>
          <Button
            variant="outline"
            onClick={() => void Promise.all(all.map((query) => query.refetch()))}
          >
            <RefreshCw className="size-4" />
            Повторить
          </Button>
        </div>
      )}
      {!gated ? (
        children(ledger)
      ) : pending ? (
        <LoadingSkeleton />
      ) : empty ? (
        <section className="rounded-2xl border bg-card px-5 py-12 text-center">
          <WalletCards aria-hidden="true" className="mx-auto mb-5 size-10 text-primary" />
          <h2>Начните с первого счёта</h2>
          <p className="mt-3 text-muted-foreground">
            Добавьте дебетовую карту или наличные и текущий остаток.
          </p>
          <div className="grace-actions justify-center">
            <Button onClick={() => setAccountOpen(true)}>Добавить первый счёт</Button>
          </div>
        </section>
      ) : (
        children(ledger)
      )}
    </div>
  )
}
