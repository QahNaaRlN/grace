import { expect, it } from 'vitest'
import { formatMinorText, formatUsdMicros } from '.'

it('formats decimal-string minor units without float, including large and negative values', () => {
  expect(formatMinorText('123456', '₽')).toBe('1 234,56 ₽')
  expect(formatMinorText('-5', 'RUB')).toBe('−0,05 RUB')
  expect(formatMinorText('900719925474099312', 'RUB')).toContain(',12 RUB')
})

it('formats dollar micros with at least two fraction digits', () => {
  expect(formatUsdMicros('1500000')).toBe('$1.50')
  expect(formatUsdMicros('1234')).toBe('$0.001234')
})
