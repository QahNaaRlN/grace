import { cn } from '@/shared/lib/utils'

const sizes = {
  // The one display size: the headline amount on Overview (32px → 40px).
  display: 'text-amount-sm font-semibold tracking-tight sm:text-amount',
  lg: 'text-title-1 font-semibold tracking-tight',
  md: 'text-base font-semibold',
  sm: 'text-sm font-medium',
} as const

// SPEC-0035: every amount on screen goes through Money. It formats integer
// minor units with ru-RU rules, carries direction in the sign (colour only
// reinforces it) and shows an unknown value in words — never as 0.
export function Money({
  valueMinor,
  currency,
  direction,
  size = 'md',
  className,
}: {
  valueMinor: number | null
  currency: string
  /** credit adds "+" in success; debit adds "−". */
  direction?: 'credit' | 'debit'
  size?: keyof typeof sizes
  className?: string
}) {
  if (valueMinor === null) {
    return (
      <span
        data-slot="money"
        data-unknown="true"
        className={cn(
          'grace-money border-b border-dashed border-input text-unknown',
          sizes[size],
          'font-medium',
          className,
        )}
      >
        Не сверён
      </span>
    )
  }
  const text = new Intl.NumberFormat('ru-RU', { style: 'currency', currency }).format(
    Math.abs(valueMinor) / 100,
  )
  const negative = direction ? direction === 'debit' : valueMinor < 0
  const sign = direction === 'credit' ? '+' : negative ? '−' : ''
  return (
    <span
      data-slot="money"
      data-direction={direction}
      className={cn(
        'grace-money',
        sizes[size],
        direction === 'credit' && 'text-success',
        className,
      )}
    >
      {sign}
      {text}
    </span>
  )
}
