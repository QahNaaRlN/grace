import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { Button } from './button'
import { Field } from './field'
import { Input } from './input'
import { Money } from './money'
import { Notice, StatusBadge } from './status'

describe('SPEC-0035 Money', () => {
  it('formats minor units, carries direction in the sign and shows unknown in words', () => {
    render(<>
      <Money valueMinor={12_845_000} currency="RUB" size="display" />
      <Money valueMinor={6_400_000} currency="RUB" direction="credit" />
      <Money valueMinor={189_700} currency="RUB" direction="debit" />
      <Money valueMinor={null} currency="RUB" />
    </>)
    const [display, credit, debit, unknown] = document.querySelectorAll('[data-slot="money"]')
    expect(display.textContent).toMatch(/^128\s450,00\s₽$/)
    expect(display).toHaveClass('text-amount-sm', 'sm:text-amount', 'grace-money')
    expect(credit.textContent).toMatch(/^\+64\s000,00\s₽$/)
    expect(credit).toHaveClass('text-success')
    expect(debit.textContent).toMatch(/^−1\s897,00\s₽$/)
    expect(unknown).toHaveTextContent('Не сверён')
    expect(unknown).toHaveAttribute('data-unknown', 'true')
    expect(screen.queryByText(/^0,00/)).not.toBeInTheDocument()
  })
})

describe('SPEC-0035 Field', () => {
  it('labels the control and describes it with the hint and the error', () => {
    render(<Field label="Сумма" hint="В рублях" error="Такой суммы нет"><Input /></Field>)
    const input = screen.getByLabelText('Сумма')
    expect(input).toHaveAccessibleDescription('В рублях Такой суммы нет')
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })
  it('keeps an existing id and description', () => {
    render(<><p id="extra">Пояснение</p><Field label="Имя"><Input id="name" aria-describedby="extra" /></Field></>)
    expect(screen.getByLabelText('Имя')).toHaveAttribute('id', 'name')
    expect(screen.getByLabelText('Имя')).toHaveAccessibleDescription('Пояснение')
  })
})

describe('SPEC-0035 status', () => {
  it('always shows a word and an icon; danger is an alert', () => {
    render(<>
      <Notice tone="unknown" title="Покрытие неполное">Кредитные карты не учтены.</Notice>
      <Notice tone="danger" title="Ассистент не ответил" />
      <StatusBadge tone="warning">Скоро платёж</StatusBadge>
    </>)
    expect(screen.getByRole('status')).toHaveTextContent('Покрытие неполное')
    expect(screen.getByRole('alert')).toHaveTextContent('Ассистент не ответил')
    const badge = screen.getByText('Скоро платёж')
    expect(badge.querySelector('svg')).not.toBeNull()
    expect(badge).toHaveClass('bg-warning-subtle', 'text-warning')
  })
  it('renders an unexpected tone as unknown instead of breaking the page', () => {
    render(<StatusBadge tone={'pending' as never}>Статус неизвестен</StatusBadge>)
    expect(screen.getByText('Статус неизвестен')).toHaveClass('bg-unknown-subtle')
  })
})

describe('SPEC-0035 Button', () => {
  it('offers a solid destructive variant and no shrunken sizes', async () => {
    render(<Button variant="destructive-solid">Удалить</Button>)
    const button = screen.getByRole('button', { name: 'Удалить' })
    expect(button).toHaveClass('bg-destructive', 'text-destructive-foreground')
    expect(button.className).not.toMatch(/translate-y|ring-3|ring-\[3px\]/)
    await userEvent.tab()
    expect(button).toHaveFocus()
  })
})
