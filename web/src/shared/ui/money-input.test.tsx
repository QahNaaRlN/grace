import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'

import { minorToInput, parseMinor } from '@/shared/lib/finance-display'
import { Label } from './label'
import { formatAmountText, NBSP } from '@/shared/lib/amount-text'
import { MoneyInput } from './money-input'

describe('SPEC-0036 amount input', () => {
  it('groups thousands and keeps the kopeck separator', () => {
    expect(formatAmountText('20000000')).toBe(`20${NBSP}000${NBSP}000`)
    expect(formatAmountText('1234.5')).toBe(`1${NBSP}234,5`)
    expect(formatAmountText(`20 000 000,00 ₽`)).toBe(`20${NBSP}000${NBSP}000,00`)
    expect(formatAmountText('−5')).toBe('-5')
  })

  it('never silently repairs an invalid amount', () => {
    expect(parseMinor(formatAmountText('1,2,3'))).toBeNull()
    expect(parseMinor(formatAmountText('1,234'))).toBeNull()
    expect(parseMinor(formatAmountText(''))).toBeNull()
  })

  it('parses grouped text into integer minor units', () => {
    expect(parseMinor(`20${NBSP}000${NBSP}000`)).toBe(2_000_000_000)
    expect(parseMinor('1 234,5')).toBe(123_450)
    expect(parseMinor('1234.56')).toBe(123_456)
    expect(parseMinor(formatAmountText('20 000 000,00 ₽'))).toBe(2_000_000_000)
    expect(parseMinor('−5')).toBe(-500)
    expect(parseMinor('9007199254740992')).toBeNull()
  })

  it('turns minor units into editable text without float division', () => {
    expect(minorToInput(1897)).toBe('18,97')
    expect(minorToInput(5)).toBe('0,05')
    expect(minorToInput(-120000)).toBe('-1200,00')
    expect(minorToInput(null)).toBe('')
  })

  it('formats while typing in an uncontrolled field read through FormData', async () => {
    render(<form aria-label="f"><Label htmlFor="sum">Сумма</Label><MoneyInput id="sum" name="amount" currency="RUB" /></form>)
    const input = screen.getByLabelText('Сумма') as HTMLInputElement
    await userEvent.type(input, '20000000')
    expect(input.value).toBe(`20${NBSP}000${NBSP}000`)
    expect(input).toHaveAttribute('inputmode', 'decimal')
    expect(input).not.toHaveAttribute('type', 'number')
    const form = screen.getByRole('form') as HTMLFormElement
    expect(parseMinor(String(new FormData(form).get('amount')))).toBe(2_000_000_000)
    expect(screen.getByText('₽')).toHaveAttribute('aria-hidden', 'true')
    expect(input).toHaveAccessibleDescription('в рублях')
  })

  it('keeps the caret in place when a separator is inserted', async () => {
    render(<><Label htmlFor="c">Сумма</Label><MoneyInput id="c" /></>)
    const input = screen.getByLabelText('Сумма') as HTMLInputElement
    await userEvent.type(input, '1000')
    expect(input.value).toBe(`1${NBSP}000`)
    expect(input.selectionStart).toBe(input.value.length)
  })

  it('works as a controlled field', async () => {
    function Controlled() {
      const [value, setValue] = useState('')
      return <><Label htmlFor="v">Сумма</Label><MoneyInput id="v" value={value} onChange={e => setValue(e.target.value)} /><output>{parseMinor(value)}</output></>
    }
    render(<Controlled />)
    await userEvent.type(screen.getByLabelText('Сумма'), '150000,5')
    expect(screen.getByLabelText('Сумма')).toHaveValue(`150${NBSP}000,5`)
    expect(screen.getByRole('status')).toHaveTextContent('15000050')
  })
})
