import type { components } from '@/shared/api/generated/schema'

type Subscription = components['schemas']['Subscription']

/** The card or account a subscription is charged to, as the board names it. */
export function subscriptionCard(s: Subscription, accounts: components['schemas']['Account'][]) {
  const account = accounts.find((a) => a.id === s.accountId)
  if (s.paymentInstrumentId) {
    const card = account?.paymentInstruments?.find((c) => c.id === s.paymentInstrumentId)
    return card ? `${card.bank} *${card.last4}` : 'Карта недоступна'
  }
  return account ? account.name : s.accountId ? 'Счёт недоступен' : 'Карта не указана'
}
