import { today } from '@/shared/lib/date'
export function initialInstallmentForm() {
  return {
    name: '',
    provider: '',
    mode: 'standalone' as 'standalone' | 'credit_card',
    currency: 'RUB',
    accountId: '',
    purchaseEventId: '',
    categoryId: '',
    openedOn: today(),
    observedOn: today(),
    principal: '',
    remaining: '',
    historical: false,
    firstAmount: '',
    firstAccount: '',
    firstOn: today(),
    reminders: false,
  }
}
