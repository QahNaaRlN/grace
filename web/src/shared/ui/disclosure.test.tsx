import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import { Disclosure } from './disclosure'
it('preserves form fields across collapse and does not submit the parent form', async () => {
  const submit = vi.fn()
  render(
    <form onSubmit={submit}>
      <Disclosure title="Дополнительно">
        <label>
          Комментарий
          <input />
        </label>
      </Disclosure>
    </form>,
  )
  const trigger = screen.getByRole('button', { name: 'Дополнительно' })
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
  await userEvent.click(trigger)
  await userEvent.type(screen.getByLabelText('Комментарий'), 'Сохранить ввод')
  await userEvent.click(trigger)
  expect(screen.getByLabelText('Комментарий').closest('[inert]')).not.toBeNull()
  await userEvent.click(trigger)
  expect(screen.getByLabelText('Комментарий')).toHaveValue('Сохранить ввод')
  expect(submit).not.toHaveBeenCalled()
})
