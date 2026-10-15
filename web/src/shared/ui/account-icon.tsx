import { CreditCard, BadgeDollarSign, Wallet, CalendarClock } from 'lucide-react'
import type { Account } from '@/shared/api'
const accountTypeLabel: Record<Account['type'], string> = {
  debit: 'Дебетовая карта / счёт',
  credit: 'Кредитная карта / счёт',
  cash: 'Наличные',
  installment: 'Рассрочка',
}
export function AccountIcon({
  type,
  className = 'size-5 shrink-0 text-primary',
}: {
  type: Account['type']
  className?: string
}) {
  const Icon = {
    debit: CreditCard,
    credit: BadgeDollarSign,
    cash: Wallet,
    installment: CalendarClock,
  }[type]
  return <Icon className={className} role="img" aria-label={accountTypeLabel[type]} />
}
