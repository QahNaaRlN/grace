import type { InstallmentInput } from '../api/use-installments'
import { formatMoney } from '@/shared/lib/money'
import { Notice } from '@/shared/ui/status'
export function InstallmentReview({ review }: { review: InstallmentInput }) {
  return (
    <Notice tone="info" title="Проверьте перед сохранением">
      {review.historical ? 'Исходный остаток' : 'Покупка'}:{' '}
      {formatMoney(review.initialRemainingMinor ?? review.principalMinor, review.currency)}.{' '}
      {review.mode === 'standalone' && !review.accountId
        ? 'Будет создан отдельный долговой счёт.'
        : 'Будет использован выбранный долговой счёт.'}{' '}
      {review.purchaseEventId
        ? 'Связываем существующую покупку.'
        : review.historical
          ? 'Прошлая покупка не создаётся.'
          : 'Будет записана одна покупка.'}{' '}
      {review.firstPayment &&
        `Будет записан перевод первого взноса ${formatMoney(review.firstPayment.amountMinor, review.currency)}.`}
    </Notice>
  )
}
