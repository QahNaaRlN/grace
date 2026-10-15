import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { fakeSchedulePreview } from '@/entities/payment-schedule/index.testing'
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
const createCalls = () => post.mock.calls.filter((call) => call[0] === '/api/v1/obligations')

beforeEach(() => {
  post.mockReset()
  post.mockImplementation(
    async (path: string, init: { body: Parameters<typeof fakeSchedulePreview>[0] }) =>
      path === '/api/v1/obligations/schedule-preview'
        ? fakeSchedulePreview(init.body)
        : { data: { id: 'o' } },
  )
})

describe('SPEC-0037 obligation with a long schedule', () => {
  it('creates a 120-payment mortgage without typing the total', async () => {
    renderForm()
    const user = userEvent.setup()
    expect(screen.queryByLabelText('Общая сумма')).not.toBeInTheDocument()
    await user.type(screen.getByLabelText('Название'), 'Ипотека')
    await user.type(screen.getByLabelText('Первый платёж'), '2027-01-15')
    await user.type(screen.getByLabelText('Количество, ежемесячно'), '120')
    await user.type(screen.getByLabelText('Сумма платежа'), '20000000')
    expect(screen.getByLabelText('Сумма платежа')).toHaveValue('20 000 000')
    await user.click(screen.getByRole('button', { name: 'Построить график' }))
    await screen.findByRole('button', { name: 'Показать все 120' })
    await user.click(screen.getByRole('button', { name: 'Сохранить обязательство' }))
    await waitFor(() => expect(createCalls()).toHaveLength(1))
    const body = createCalls()[0][1].body
    expect(body.totalMinor).toBe(120 * 2_000_000_000)
    expect(body.payments).toHaveLength(120)
    expect(body.payments[119]).toEqual({ dueOn: '2036-12-15', amountMinor: 2_000_000_000 })
  })

  it('does not save a schedule with a broken row', async () => {
    renderForm()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Название'), 'Рассрочка')
    await user.click(screen.getByRole('button', { name: 'Вручную' }))
    await user.click(screen.getByRole('button', { name: 'Добавить строку' }))
    await user.click(screen.getByRole('button', { name: 'Сохранить обязательство' }))
    expect(await screen.findByText(/Исправьте график: строка 1/)).toBeVisible()
    expect(createCalls()).toHaveLength(0)
  })

  it('asks for at least one payment', async () => {
    renderForm()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Название'), 'Рассрочка')
    await user.click(screen.getByRole('button', { name: 'Сохранить обязательство' }))
    expect(await screen.findByText('Добавьте хотя бы один платёж.')).toBeVisible()
    expect(post).not.toHaveBeenCalled()
  })
})
