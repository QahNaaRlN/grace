import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { ChoiceSelect } from './choice-select'
import { Checkbox } from './checkbox'
import { PasswordInput } from './password-input'
import { Label } from './label'

function SelectExample({ changed }: { changed: (value: string) => void }) {
  const [value, setValue] = useState('RUB')
  return (
    <>
      <Label htmlFor="currency">Валюта</Label>
      <ChoiceSelect
        id="currency"
        value={value}
        onValueChange={(value) => {
          setValue(value)
          changed(value)
        }}
      >
        <option value="">Все валюты</option>
        <option value="RUB">Рубли</option>
        <option value="USD">Доллары</option>
      </ChoiceSelect>
    </>
  )
}
describe('SPEC-0032 shared library controls', () => {
  it('keeps an empty selection meaningful and never sends the internal sentinel', async () => {
    const changed = vi.fn()
    render(<SelectExample changed={changed} />)
    await userEvent.click(screen.getByRole('combobox', { name: 'Валюта' }))
    await userEvent.click(screen.getByRole('option', { name: 'Все валюты' }))
    expect(changed).toHaveBeenLastCalledWith('')
    expect(screen.getByRole('combobox')).toHaveTextContent('Все валюты')
    await userEvent.click(screen.getByRole('combobox'))
    await userEvent.click(screen.getByRole('option', { name: 'Доллары' }))
    expect(changed).toHaveBeenLastCalledWith('USD')
  })
  it('toggles a checkbox without submitting the containing form and respects disabled', async () => {
    const changed = vi.fn(),
      submit = vi.fn()
    render(
      <form onSubmit={submit}>
        <label>
          <Checkbox checked={false} onCheckedChange={changed} />
          Подтверждаю
        </label>
        <Checkbox aria-label="Недоступно" checked={false} disabled onCheckedChange={changed} />
      </form>,
    )
    await userEvent.click(screen.getByRole('checkbox', { name: 'Подтверждаю' }))
    expect(changed).toHaveBeenCalledWith(true)
    await userEvent.click(screen.getByRole('checkbox', { name: 'Недоступно' }))
    expect(changed).toHaveBeenCalledTimes(1)
    expect(submit).not.toHaveBeenCalled()
  })
  it('opens and closes the password eye without losing text or submitting', async () => {
    const submit = vi.fn()
    render(
      <form onSubmit={submit}>
        <Label htmlFor="password">Пароль</Label>
        <PasswordInput id="password" autoComplete="current-password" />
      </form>,
    )
    const input = screen.getByLabelText('Пароль')
    await userEvent.type(input, 'a synthetic password')
    expect(input).toHaveAttribute('type', 'password')
    await userEvent.click(screen.getByRole('button', { name: 'Показать пароль' }))
    expect(input).toHaveAttribute('type', 'text')
    expect(input).toHaveValue('a synthetic password')
    expect(screen.getByRole('button', { name: 'Скрыть пароль' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Скрыть пароль' }))
    expect(input).toHaveAttribute('type', 'password')
    expect(submit).not.toHaveBeenCalled()
    expect(input).toHaveAttribute('autocomplete', 'current-password')
  })
})
