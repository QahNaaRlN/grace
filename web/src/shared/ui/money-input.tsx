import { useId, type ChangeEvent, type ComponentProps } from 'react'

import { caretAfterFormat, formatAmountText } from '@/shared/lib/amount-text'
import { cn } from '@/shared/lib/utils'
import { Input } from './input'

// SPEC-0036: an amount field that groups thousands as the person types
// ("20000000" → "20 000 000"). Works uncontrolled (FormData, react-hook-form
// register) and controlled; the value it reports is the grouped text, which
// parseMinor turns into integer minor units.
export function MoneyInput({
  className,
  currency,
  allowNegative = false,
  value,
  defaultValue,
  onChange,
  ...props
}: Omit<ComponentProps<'input'>, 'type' | 'inputMode'> & {
  /** ISO code; RUB renders as a ₽ suffix and is announced to screen readers. */
  currency?: string
  /** Shows the full keyboard on phones, where decimal pads have no minus. */
  allowNegative?: boolean
}) {
  const code = currency?.trim().toUpperCase()
  const suffix = code ? (code === 'RUB' ? '₽' : code) : null
  const suffixId = useId()
  const describedBy =
    [props['aria-describedby'], suffix ? suffixId : null].filter(Boolean).join(' ') || undefined
  function change(event: ChangeEvent<HTMLInputElement>) {
    const input = event.target
    const raw = input.value
    const next = formatAmountText(raw)
    if (next !== raw) {
      const position = caretAfterFormat(raw, input.selectionStart ?? raw.length, next)
      input.value = next
      input.setSelectionRange(position, position)
    }
    onChange?.(event)
  }
  return (
    <div className="relative">
      <Input
        {...props}
        aria-describedby={describedBy}
        type="text"
        inputMode={allowNegative ? 'text' : 'decimal'}
        autoComplete="off"
        spellCheck={false}
        value={value === undefined ? undefined : formatAmountText(String(value))}
        defaultValue={
          defaultValue === undefined ? undefined : formatAmountText(String(defaultValue))
        }
        onChange={change}
        className={cn('grace-money tabular-nums', suffix && 'pr-10', className)}
      />
      {suffix && (
        <>
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-muted-foreground"
          >
            {suffix}
          </span>
          <span id={suffixId} className="sr-only">
            {code === 'RUB' ? 'в рублях' : `в валюте ${code}`}
          </span>
        </>
      )}
    </div>
  )
}
