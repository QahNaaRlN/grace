import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account } from '@/shared/api'
import { Accounts } from './Accounts'

const { get, post, patch } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: post, PATCH: patch } }))

const base: Account = {
  id: 'sber',
  name: 'Сбербанк — счёт 3021',
  type: 'debit',
  currency: 'RUB',
  balanceMinor: null,
  balanceReconciled: false,
  createdAt: '2026-10-01T00:00:00Z',
  revision: 1,
  hasOperations: true,
  lastReconciliation: null,
  lastImportConfirmedAt: '2026-10-09T06:00:00Z',
  paymentInstruments: [{ id: 'c1', name: 'Карта', bank: 'Сбербанк', last4: '3021' }],
}
const reconciled: Account = {
  ...base,
  id: 'tinkoff',
  name: 'Т-Банк',
  balanceMinor: 975000,
  balanceReconciled: true,
  revision: 3,
  lastReconciliation: {
    asOf: '2026-10-13T09:20:00Z',
    outcome: 'adjusted',
    statedMinor: 975000,
    differenceMinor: -10000,
    createdAt: '2026-10-13T09:21:00Z',
    stale: false,
  },
  paymentInstruments: [
    {
      id: 'old',
      name: 'Старая',
      bank: 'Т-Банк',
      last4: '1111',
      disabledAt: '2026-10-10T00:00:00Z',
    },
    { id: 'new', name: 'Новая', bank: 'Т-Банк', last4: '2222' },
  ].reverse(),
}
const stale: Account = {
  ...reconciled,
  id: 'stale',
  name: 'Наличные',
  type: 'cash',
  paymentInstruments: [],
  lastImportConfirmedAt: null,
  lastReconciliation: { ...reconciled.lastReconciliation!, stale: true },
}

function show(items: Account[]) {
  render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <Accounts items={items} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  get.mockReset()
  post.mockReset()
  patch.mockReset()
  get.mockResolvedValue({
    data: {
      items: [
        {
          id: 'r1',
          accountId: 'tinkoff',
          asOf: '2026-10-13T09:20:00Z',
          statedMinor: 975000,
          computedMinor: 985000,
          differenceMinor: -10000,
          outcome: 'adjusted',
          adjustmentEventId: 'e1',
          createdAt: '2026-10-13T09:21:00Z',
        },
      ],
    },
  })
})

describe('SPEC-0040 accounts', () => {
  it('shows the last reconciliation and import on every row', () => {
    show([base, reconciled, stale])
    const rows = screen.getAllByRole('listitem')
    expect(rows[0]).toHaveTextContent('Не сверён')
    expect(rows[0]).toHaveTextContent('импорт 9 окт., 11:00')
    expect(rows[1]).toHaveTextContent('Сверен 13 окт., 14:20')
    expect(rows[1]).toHaveTextContent('Т-Банк *2222')
    expect(rows[2]).toHaveTextContent('Нужна повторная сверка')
  })

  it('reconciles an unreconciled account: preview, then confirm with the hash and a key', async () => {
    post.mockImplementation(async (path: string) =>
      path.endsWith('/preview')
        ? {
            data: {
              asOf: '2026-10-05T20:00:00+05:00',
              statedMinor: 1000000,
              currency: 'RUB',
              computedMinor: 350000,
              differenceMinor: 650000,
              outcome: 'opening_set',
              proposedOpeningMinor: 650000,
              previewHash: 'h1',
            },
          }
        : {
            data: {
              reconciliation: {
                id: 'r',
                accountId: 'sber',
                asOf: '2026-10-05T15:00:00Z',
                statedMinor: 1000000,
                computedMinor: 350000,
                differenceMinor: 650000,
                outcome: 'opening_set',
                adjustmentEventId: null,
                createdAt: '',
              },
              account: base,
            },
          },
    )
    show([base])
    await userEvent.click(screen.getByRole('button', { name: /Сбербанк \*3021/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Сверить остаток' }))
    const dialog = screen.getByRole('dialog', { name: 'Сверить остаток' })
    await userEvent.type(within(dialog).getByLabelText('Остаток в банке'), '10000')
    const when = within(dialog).getByLabelText('На дату и время')
    await userEvent.clear(when)
    await userEvent.type(when, '2026-10-05T20:00')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Проверить' }))
    expect(post.mock.calls[0][1].body).toEqual({
      asOf: '2026-10-05T20:00:00+05:00',
      statedMinor: 1000000,
    })
    expect(await within(dialog).findByText('Начальный остаток')).toBeVisible()
    expect(within(dialog).getByText(/6\s500,00\s₽/)).toBeVisible()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Подтвердить сверку' }))
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
    const [path, init] = post.mock.calls[1]
    expect(path).toBe('/api/v1/accounts/{accountId}/reconciliations')
    expect(init.body).toMatchObject({ statedMinor: 1000000, previewHash: 'h1' })
    expect(init.params.header['Idempotency-Key']).toBeTruthy()
    expect(await within(dialog).findByText(/Начальный остаток установлен/)).toBeVisible()
  })

  it('explains a stale preview and asks to check again', async () => {
    post.mockImplementation(async (path: string) =>
      path.endsWith('/preview')
        ? {
            data: {
              asOf: '2026-10-13T12:00:00+05:00',
              statedMinor: 970000,
              currency: 'RUB',
              computedMinor: 975000,
              differenceMinor: -5000,
              outcome: 'adjusted',
              proposedOpeningMinor: null,
              previewHash: 'h2',
            },
          }
        : {
            error: {
              code: 'stale_preview',
              message: 'Операции по счёту изменились. Проверьте сверку ещё раз.',
            },
          },
    )
    show([reconciled])
    await userEvent.click(screen.getByRole('button', { name: /Т-Банк \*2222/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Сверить остаток' }))
    const dialog = screen.getByRole('dialog', { name: 'Сверить остаток' })
    await userEvent.type(within(dialog).getByLabelText('Остаток в банке'), '9700')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Проверить' }))
    expect(await within(dialog).findByText(/«Корректировка остатка»/)).toBeVisible()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Подтвердить сверку' }))
    expect(
      await within(dialog).findByText('Операции по счёту изменились. Проверьте сверку ещё раз.'),
    ).toBeVisible()
    expect(within(dialog).getByRole('button', { name: 'Проверить' })).toBeVisible()
  })

  it('locks the type of an account with operations and keeps disabled cards visible', async () => {
    show([reconciled])
    await userEvent.click(screen.getByRole('button', { name: /Т-Банк \*2222/ }))
    const sheet = screen.getByRole('dialog')
    expect(within(sheet).getByText('Старая · отключена')).toBeVisible()
    expect(within(sheet).getByText('Последний импорт')).toBeVisible()
    await userEvent.click(within(sheet).getByRole('button', { name: /История сверок/ }))
    expect(await within(sheet).findByText(/разница -100,00\s₽/)).toBeVisible()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Изменить сведения' }))
    const edit = screen.getByRole('dialog', { name: 'Сведения счёта' })
    expect(within(edit).getByText(/По счёту есть операции/)).toBeVisible()
    expect(within(edit).getByRole('combobox')).toBeDisabled()
  })

  it('renames an account with its revision and disables a card', async () => {
    patch.mockResolvedValue({ data: reconciled })
    show([reconciled])
    await userEvent.click(screen.getByRole('button', { name: /Т-Банк \*2222/ }))
    const sheet = screen.getByRole('dialog')
    await userEvent.click(within(sheet).getAllByRole('button', { name: 'Отключить' })[0])
    await waitFor(() => expect(patch).toHaveBeenCalledOnce())
    expect(patch.mock.calls[0][1]).toMatchObject({
      params: { path: { accountId: 'tinkoff', cardId: 'new' } },
      body: { disabled: true },
    })
    await userEvent.click(within(sheet).getByRole('button', { name: 'Изменить сведения' }))
    const edit = screen.getByRole('dialog', { name: 'Сведения счёта' })
    const name = within(edit).getByLabelText('Название')
    await userEvent.clear(name)
    await userEvent.type(name, 'Т-Банк зарплатный')
    await userEvent.click(within(edit).getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(patch).toHaveBeenCalledTimes(2))
    expect(patch.mock.calls[1][1].body).toEqual({ revision: 3, name: 'Т-Банк зарплатный' })
  })
})
