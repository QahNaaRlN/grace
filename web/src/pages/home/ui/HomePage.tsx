import { useQuery } from '@tanstack/react-query'
import { Activity, ArrowDownRight, WalletCards } from 'lucide-react'

import { CreateTransactionForm } from '@/features/create-transaction'
import { apiClient } from '@/shared/api/client'
import type { Account, Transaction } from '@/shared/api/types'
import { Badge } from '@/shared/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/shared/ui/card'
import { AppLayout } from '@/shared/ui/layout/AppLayout'

function formatMoney(valueMinor: number, currency: string) {
  return new Intl.NumberFormat('ru-RU', {
    style: 'currency',
    currency,
  }).format(valueMinor / 100)
}

async function loadAccounts() {
  const { data } = await apiClient.GET('/api/v1/accounts')
  if (!data) throw new Error('Не удалось загрузить счета')
  return data.items
}

async function loadCategories() {
  const { data } = await apiClient.GET('/api/v1/categories')
  if (!data) throw new Error('Не удалось загрузить категории')
  return data.items
}

async function loadTransactions() {
  const { data } = await apiClient.GET('/api/v1/transactions')
  if (!data) throw new Error('Не удалось загрузить операции')
  return data.items
}

function Accounts({ items }: { items: Account[] }) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {items.map((account) => (
        <Card key={account.id}>
          <CardHeader className="pb-2">
            <CardDescription>{account.type}</CardDescription>
            <CardTitle className="text-lg">{account.name}</CardTitle>
          </CardHeader>
          <CardContent className="text-2xl font-semibold">
            {formatMoney(account.balanceMinor, account.currency)}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function Transactions({ items }: { items: Transaction[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Последние операции</CardTitle>
        <CardDescription>
          Техническая операция хранится вместе с её финансовым смыслом.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">Операций пока нет.</p>
        ) : (
          <div className="divide-y">
            {items.map((item) => (
              <article key={item.id} className="flex items-center justify-between gap-4 py-3">
                <div className="flex min-w-0 items-center gap-3">
                  <div className="rounded-full bg-muted p-2">
                    <ArrowDownRight aria-hidden="true" className="size-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate font-medium">{item.merchant || item.description}</p>
                    <p className="text-sm text-muted-foreground">
                      {item.categoryName ?? item.kind} · {item.accountName}
                    </p>
                  </div>
                </div>
                <div className="shrink-0 text-right">
                  <p className="font-medium text-destructive">
                    −{formatMoney(item.amountMinor, item.currency)}
                  </p>
                  <Badge variant="secondary">{item.kind}</Badge>
                </div>
              </article>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function HomePage() {
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: loadAccounts })
  const categories = useQuery({ queryKey: ['categories'], queryFn: loadCategories })
  const transactions = useQuery({ queryKey: ['transactions'], queryFn: loadTransactions })

  const error = accounts.error ?? categories.error ?? transactions.error
  const isPending = accounts.isPending || categories.isPending || transactions.isPending

  return (
    <AppLayout>
      <section className="space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-muted-foreground">Grace</p>
            <h1 className="text-3xl font-semibold tracking-tight">Финансовая картина</h1>
          </div>
          <Badge variant={error ? 'destructive' : 'secondary'}>
            <Activity aria-hidden="true" className="mr-1 size-3" />
            {error ? 'API недоступен' : isPending ? 'Загрузка…' : 'Данные актуальны'}
          </Badge>
        </div>

        {error && (
          <Card>
            <CardContent className="flex items-center gap-3 pt-6 text-destructive">
              <WalletCards aria-hidden="true" className="size-5" />
              <p>{error instanceof Error ? error.message : 'Не удалось связаться с API'}</p>
            </CardContent>
          </Card>
        )}

        {accounts.data && <Accounts items={accounts.data} />}

        {accounts.data && categories.data && (
          <CreateTransactionForm accounts={accounts.data} categories={categories.data} />
        )}

        {transactions.data && <Transactions items={transactions.data} />}
      </section>
    </AppLayout>
  )
}
