import type { ComponentProps } from 'react'

import { currencies } from '@/shared/lib/currencies'
import { ChoiceSelect } from './choice-select'

// SPEC-0047: a currency is chosen from a list, never typed. A stored code that
// is not in the list is kept as its own option so editing does not change it.
export function CurrencySelect({
  value,
  onValueChange,
  name,
  ...props
}: Omit<ComponentProps<typeof ChoiceSelect>, 'children'> & { name?: string }) {
  const known = currencies.some(([code]) => code === value)
  return (
    <>
      <ChoiceSelect value={value} onValueChange={onValueChange} {...props}>
        {!known && value && <option value={value}>{value}</option>}
        {currencies.map(([code, label]) => (
          <option key={code} value={code}>{`${label} · ${code}`}</option>
        ))}
      </ChoiceSelect>
      {name && <input type="hidden" name={name} value={value} />}
    </>
  )
}
