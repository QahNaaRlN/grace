import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { CreateObligationForm } from './CreateObligationForm'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

function renderForm() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <CreateObligationForm />
    </QueryClientProvider>,
  )
}

beforeEach(() => post.mockReset())

describe('SPEC-0036 obligation amounts', () => {
  it('groups a mortgage-sized total and sends exact minor units', async () => {
    post.mockResolvedValue({ data: { id: 'o' } })
    renderForm()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Название'), 'Ипотека')
    await user.type(screen.getByLabelText('Общая сумма'), '20000000')
    expect(screen.getByLabelText('Общая сумма')).toHaveValue('20 000 000')
    expect(screen.getByLabelText('Общая сумма')).not.toHaveAttribute('type', 'number')
    await user.type(screen.getByLabelText('Дата #1'), '2027-01-15')
    await user.type(screen.getByLabelText('Сумма'), '20000000,00')
    await user.click(screen.getByRole('button', { name: 'Сохранить обязательство' }))
    await waitFor(() => expect(post).toHaveBeenCalledOnce())
    expect(post.mock.calls[0][1].body).toMatchObject({
      totalMinor: 2_000_000_000,
      payments: [{ dueOn: '2027-01-15', amountMinor: 2_000_000_000 }],
    })
  })

  it('rejects three decimals instead of rounding them', async () => {
    renderForm()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Название'), 'Рассрочка')
    await user.type(screen.getByLabelText('Общая сумма'), '10,005')
    await user.type(screen.getByLabelText('Дата #1'), '2027-01-15')
    await user.type(screen.getByLabelText('Сумма'), '10,005')
    await user.click(screen.getByRole('button', { name: 'Сохранить обязательство' }))
    expect(await screen.findAllByText('Не больше двух знаков после запятой')).toHaveLength(2)
    expect(post).not.toHaveBeenCalled()
  })
})
