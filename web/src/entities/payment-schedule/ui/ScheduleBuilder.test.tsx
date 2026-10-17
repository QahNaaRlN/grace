import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ScheduleRow } from '../model/schedule-rows'
import { fakeSchedulePreview } from '../testing/fake-preview'
import { ScheduleBuilder } from './ScheduleBuilder'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

function Harness({
  initial = [] as ScheduleRow[],
  mode,
}: {
  initial?: ScheduleRow[]
  mode?: 'equal' | 'paste' | 'manual'
}) {
  const [rows, setRows] = useState(initial)
  return (
    <ScheduleBuilder
      rows={rows}
      onChange={setRows}
      currency="RUB"
      idPrefix="t"
      initialMode={mode}
    />
  )
}

beforeEach(() => {
  post.mockReset()
  post.mockImplementation(
    async (_path: string, init: { body: Parameters<typeof fakeSchedulePreview>[0] }) =>
      fakeSchedulePreview(init.body),
  )
})

describe('SPEC-0037 schedule builder', () => {
  it('builds 120 equal monthly payments, collapses them and shows the Go total', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.type(screen.getByLabelText('Первый платёж'), '2027-01-15')
    await user.type(screen.getByLabelText('Количество, ежемесячно'), '120')
    await user.type(screen.getByLabelText('Сумма платежа'), '25000')
    await user.click(screen.getByRole('button', { name: 'Построить график' }))
    expect(post.mock.calls[0][1].body).toEqual({
      mode: 'equal',
      firstDueOn: '2027-01-15',
      count: 120,
      interval: 'monthly',
      amountMinor: 2_500_000,
    })
    expect(await screen.findByRole('button', { name: 'Показать все 120' })).toBeVisible()
    expect(screen.getAllByLabelText(/^Дата платежа/)).toHaveLength(6)
    expect(screen.getByLabelText('Дата платежа 120')).toHaveValue('2036-12-15')
    expect(await screen.findByText(/120 платежей .* итог 3\s000\s000,00/)).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Показать все 120' }))
    expect(screen.getAllByLabelText(/^Дата платежа/)).toHaveLength(120)
  })

  it('parses a pasted table and lists skipped lines with reasons', async () => {
    const user = userEvent.setup()
    render(<Harness mode="paste" />)
    await user.click(screen.getByLabelText('Строки графика из банка'))
    await user.paste('Дата\tСумма\n15.01.2027 25 000,00\n15.02.2027 25 000,00\nИтого 50 000,00')
    await user.click(screen.getByRole('button', { name: 'Распознать график' }))
    const skipped = await screen.findByText('Пропущено строк: 2')
    expect(within(skipped.parentElement as HTMLElement).getByText(/Строка 1/)).toBeVisible()
    expect(screen.getAllByLabelText(/^Дата платежа/)).toHaveLength(2)
    expect(post.mock.calls[0][1].body).toMatchObject({ mode: 'paste', amountColumn: 1 })
  })

  it('adds the next month on Enter in the last amount and fills amounts down', async () => {
    const user = userEvent.setup()
    render(
      <Harness
        mode="manual"
        initial={[
          { key: 'a', dueOn: '2027-01-31', amount: '100,00' },
          { key: 'b', dueOn: '2027-02-28', amount: '' },
        ]}
      />,
    )
    await user.click(screen.getByRole('button', { name: 'Сумму платежа 1 — во все следующие' }))
    expect(screen.getByLabelText('Сумма платежа 2')).toHaveValue('100,00')
    await user.type(screen.getByLabelText('Сумма платежа 2'), '{Enter}')
    await waitFor(() => expect(screen.getByLabelText('Дата платежа 3')).toHaveValue('2027-03-28'))
    expect(screen.getByLabelText('Сумма платежа 3')).toHaveValue('100,00')
  })

  it('keeps paid rows locked when a generator replaces the schedule', async () => {
    const user = userEvent.setup()
    render(
      <Harness
        initial={[
          {
            key: 'paid',
            dueOn: '2026-12-15',
            amount: '250,00',
            locked: true,
            sourceId: 'p',
            paidMinor: 100,
          },
        ]}
      />,
    )
    expect(screen.getByLabelText('Дата платежа 1 · есть оплата')).toBeDisabled()
    await user.type(screen.getByLabelText('Первый платёж'), '2027-01-15')
    await user.type(screen.getByLabelText('Количество, ежемесячно'), '2')
    await user.type(screen.getByLabelText('Сумма платежа'), '300')
    await user.click(screen.getByRole('button', { name: 'Построить график' }))
    await waitFor(() => expect(screen.getAllByLabelText(/^Дата платежа/)).toHaveLength(3))
    expect(screen.getByLabelText('Дата платежа 1 · есть оплата')).toHaveValue('2026-12-15')
    expect(screen.getByLabelText('Удалить платёж 1')).toBeDisabled()
  })
})

describe('SPEC-0045 schedule by contract term', () => {
  it('builds payments from a day of month and a from–to month range', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('radio', { name: 'По сроку договора' }))
    expect(screen.queryByLabelText('Количество, ежемесячно')).not.toBeInTheDocument()
    await user.type(screen.getByLabelText('С (месяц и год)'), '2027-01')
    await user.type(screen.getByLabelText('По (включительно)'), '2036-12')
    await user.type(screen.getByLabelText('Сумма платежа'), '25000')
    await user.click(screen.getByRole('button', { name: 'Построить график' }))
    expect(post.mock.calls[0][1].body).toEqual({
      mode: 'period',
      dayOfMonth: 15,
      fromMonth: '2027-01',
      toMonth: '2036-12',
      amountMinor: 2_500_000,
    })
    expect(await screen.findByRole('button', { name: 'Показать все 120' })).toBeVisible()
    expect(screen.getByLabelText('Дата платежа 120')).toHaveValue('2036-12-15')
  })

  it('asks for both months before building', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    await user.click(screen.getByRole('radio', { name: 'По сроку договора' }))
    await user.type(screen.getByLabelText('Сумма платежа'), '100')
    await user.click(screen.getByRole('button', { name: 'Построить график' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Укажите месяц начала и месяц окончания.')
    expect(post).not.toHaveBeenCalled()
  })
})

describe('SPEC-0047 schedule layout', () => {
  it('keeps every row in its grid columns: date, amount, fill down, delete', () => {
    const initial: ScheduleRow[] = [
      { key: 'a', dueOn: '2025-05-14', amount: '5 586,00', locked: false },
      { key: 'b', dueOn: '2025-06-14', amount: '5 627,16', locked: false },
    ]
    render(<Harness initial={initial} mode="manual" />)
    const grid = screen.getByText('Дата').parentElement!
    const cells = Array.from(grid.children)
    // No absolutely positioned (sr-only) direct child may sit in the grid.
    expect(cells.some((cell) => cell.classList.contains('sr-only'))).toBe(false)
    // Header: Дата, Сумма, one cell spanning both action columns.
    expect(cells[2]).toHaveClass('col-span-2')
    expect(cells[3]).toBe(screen.getByLabelText('Дата платежа 1'))
    expect(cells[4]).toContainElement(screen.getByLabelText('Сумма платежа 1'))
    expect(cells[7]).toBe(screen.getByLabelText('Дата платежа 2'))
    expect(cells[8]).toContainElement(screen.getByLabelText('Сумма платежа 2'))
  })
})

describe('SPEC-0053 loan parts', () => {
  function Loan() {
    const [rows, setRows] = useState<ScheduleRow[]>([])
    return (
      <ScheduleBuilder
        rows={rows}
        onChange={setRows}
        currency="RUB"
        idPrefix="loan"
        initialMode="paste"
        withParts
      />
    )
  }

  it('maps bank columns of principal and interest and shows the parts of each row', async () => {
    const user = userEvent.setup()
    post.mockImplementation(async (_path: string, init: { body: { mode: string } }) =>
      init.body.mode === 'paste'
        ? {
            data: {
              payments: [
                {
                  dueOn: '2027-01-15',
                  amountMinor: 2_500_000,
                  principalMinor: 1_750_000,
                  interestMinor: 700_000,
                  otherMinor: 50_000,
                },
              ],
              totalMinor: 2_500_000,
              rejectedLines: [],
            },
          }
        : fakeSchedulePreview(init.body as Parameters<typeof fakeSchedulePreview>[0]),
    )
    render(<Loan />)
    await user.click(screen.getByLabelText('Строки графика из банка'))
    await user.paste('15.01.2027;7000,00;17500,00;500,00;25000,00')
    await user.click(screen.getByRole('combobox', { name: 'Сумма платежа — колонка после даты' }))
    await user.click(await screen.findByRole('option', { name: '4-я' }))
    await user.click(screen.getByRole('combobox', { name: 'Основной долг — колонка' }))
    await user.click(await screen.findByRole('option', { name: '2-я' }))
    await user.click(screen.getByRole('button', { name: 'Распознать график' }))
    expect(await screen.findByText('Укажите обе колонки: основной долг и проценты.')).toBeVisible()
    await user.click(screen.getByRole('combobox', { name: 'Проценты — колонка' }))
    await user.click(await screen.findByRole('option', { name: '1-я' }))
    await user.click(screen.getByRole('combobox', { name: 'Прочее — колонка' }))
    await user.click(await screen.findByRole('option', { name: '3-я' }))
    await user.click(screen.getByRole('button', { name: 'Распознать график' }))
    await waitFor(() =>
      expect(post.mock.calls[0][1].body).toEqual({
        mode: 'paste',
        text: '15.01.2027;7000,00;17500,00;500,00;25000,00',
        amountColumn: 4,
        principalColumn: 2,
        interestColumn: 1,
        otherColumn: 3,
      }),
    )
    const principal = await screen.findByLabelText<HTMLInputElement>('Тело, платёж 1')
    expect(principal.value).toMatch(/^17\s500,00$/)
    expect(screen.getByLabelText<HTMLInputElement>('Прочее, платёж 1').value).toBe('500,00')
    expect(screen.getByRole('checkbox', { name: 'Указать тело и проценты' })).toBeChecked()
  })
})
