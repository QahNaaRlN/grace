import { useQuery } from '@tanstack/react-query'
import { Activity, ArrowDownRight, ArrowRightLeft, WalletCards } from 'lucide-react'

import { CreateFinancialEventForm } from '@/features/create-financial-event'
import { apiClient } from '@/shared/api/client'
import type { Account, FinancialEvent } from '@/shared/api/types'
import { Badge } from '@/shared/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/shared/ui/card'
import { AppLayout } from '@/shared/ui/layout/AppLayout'

const kindLabels: Record<FinancialEvent['kind'], string> = {
  purchase: 'Покупка',
  income_received: 'Доход',
  internal_transfer: 'Перевод',
  refund: 'Возврат',
  debt_payment: 'Платёж по обязательству',
}

function formatMoney(valueMinor: number, currency: string) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency }).format(valueMinor / 100)
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

async function loadEvents() {
  const { data } = await apiClient.GET('/api/v1/financial-events')
  if (!data) throw new Error('Не удалось загрузить финансовые события')
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

function Events({ items }: { items: FinancialEvent[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Финансовые события</CardTitle>
        <CardDescription>
          Каждое событие объясняет одну или несколько связанных проводок.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">Событий пока нет.</p>
        ) : (
          <div className="divide-y">
            {items.map((item) => {
              const posting = item.postings[0]
              const isTransfer = item.kind === 'internal_transfer'
              const positive = posting?.direction === 'credit'
              const sourcePosting = item.postings.find((value) => value.direction === 'debit')
              const destinationPosting = item.postings.find((value) => value.direction === 'credit')
              const accountDescription = isTransfer
                ? `${sourcePosting?.accountName ?? 'Неизвестный счёт'} → ${destinationPosting?.accountName ?? 'Неизвестный счёт'}`
                : posting?.accountName
              return (
                <article key={item.id} className="flex items-center justify-between gap-4 py-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <div className="rounded-full bg-muted p-2">
                      {isTransfer
                        ? <ArrowRightLeft aria-hidden="true" className="size-4" />
                        : <ArrowDownRight aria-hidden="true" className="size-4" />}
                    </div>
                    <div className="min-w-0">
                      <p className="truncate font-medium">
                        {item.merchant || item.description || kindLabels[item.kind]}
                      </p>
                      <p className="text-sm text-muted-foreground">
                        {item.categoryName ?? kindLabels[item.kind]} · {accountDescription}
                      </p>
                    </div>
                  </div>
                  {posting && (
                    <div className="shrink-0 text-right">
                      <p className={positive ? 'font-medium text-emerald-700' : 'font-medium text-destructive'}>
                        {isTransfer ? '' : positive ? '+' : '−'}
                        {formatMoney(posting.amountMinor, posting.currency)}
                      </p>
                      <Badge variant="secondary">{kindLabels[item.kind]}</Badge>
                    </div>
                  )}
                </article>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function HomePage() {
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: loadAccounts })
  const categories = useQuery({ queryKey: ['categories'], queryFn: loadCategories })
  const events = useQuery({ queryKey: ['financial-events'], queryFn: loadEvents })
  const error = accounts.error ?? categories.error ?? events.error
  const isPending = accounts.isPending || categories.isPending || events.isPending

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
          <CreateFinancialEventForm accounts={accounts.data} categories={categories.data} />
        )}
        {events.data && <Events items={events.data} />}
      </section>
    </AppLayout>
  )
}
