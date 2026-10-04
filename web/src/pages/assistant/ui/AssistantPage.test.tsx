import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import { AssistantRequestError, type SubmitAssistant } from '@/features/ask-assistant'
import type { AssistantResult } from '@/shared/api/types'

import { AssistantPage } from './AssistantPage'

const accountResult = {
  status: 'completed',
  route: 'get_accounts',
  confidence: 0.94,
  data: {
    name: 'get_accounts',
    data: {
      items: [{
        id: '00000000-0000-0000-0000-000000000001',
        name: 'Основной счёт',
        type: 'debit',
        currency: 'RUB',
        balanceMinor: 125_050,
        createdAt: '2026-10-01T10:00:00+05:00',
      }],
    },
  },
} satisfies AssistantResult

const obligationsResult = {
  status: 'completed',
  route: 'get_obligations',
  confidence: 0.91,
  data: {
    name: 'get_obligations',
    data: {
      asOf: '2026-10-05',
      items: [{
        id: '00000000-0000-0000-0000-000000000002',
        name: 'Рассрочка',
        kind: 'installment',
        currency: 'RUB',
        totalMinor: 100_000,
        paidMinor: 25_000,
        remainingMinor: 75_000,
        status: 'active',
        createdAt: '2026-09-01T10:00:00+05:00',
        payments: [{
          id: '00000000-0000-0000-0000-000000000003',
          position: 1,
          dueOn: '2026-10-20',
          amountMinor: 25_000,
          paidMinor: 0,
          remainingMinor: 25_000,
          status: 'planned',
          isOverdue: false,
          settlements: [],
        }],
      }],
    },
  },
} satisfies AssistantResult

const receiptsResult = {
  status: 'completed',
  route: 'get_receipts',
  confidence: 0.88,
  data: {
    name: 'get_receipts',
    data: {
      items: [{
        id: '00000000-0000-0000-0000-000000000004',
        purchaseEventId: '00000000-0000-0000-0000-000000000005',
        seller: 'Магазин',
        issuedAt: '2026-10-04T14:00:00+05:00',
        currency: 'RUB',
        totalMinor: 350_00,
        fiscalDocumentNumber: '42',
        createdAt: '2026-10-04T14:05:00+05:00',
        items: [{
          id: '00000000-0000-0000-0000-000000000006',
          position: 1,
          name: 'Продукты',
          quantityMilli: 1_000,
          unitPriceMinor: 350_00,
          lineTotalMinor: 350_00,
        }],
      }],
    },
  },
} satisfies AssistantResult

const incomePeriodResult = {
  status: 'completed',
  route: 'get_period_since_income',
  confidence: 0.9,
  data: {
    name: 'get_period_since_income',
    data: {
      anchor: {
        eventId: '00000000-0000-0000-0000-000000000007',
        categoryId: null,
        categoryName: 'Зарплата',
        occurredAt: '2026-10-01T09:00:00+05:00',
        postings: [{
          accountId: '00000000-0000-0000-0000-000000000001',
          accountName: 'Основной счёт',
          amountMinor: 200_000,
          currency: 'RUB',
        }],
      },
      summary: {
        from: '2026-10-01T09:00:00+05:00',
        to: '2026-10-05T18:00:00+05:00',
        currencies: [{
          currency: 'RUB',
          openingBalanceMinor: 10_000,
          closingBalanceMinor: 125_050,
          incomeMinor: 200_000,
          purchasesMinor: 70_000,
          refundsMinor: 0,
          debtPaymentsMinor: 14_950,
          netExternalCashFlowMinor: 115_050,
          internalTransferVolumeMinor: 0,
          expenseCategories: [],
        }],
      },
    },
  },
} satisfies AssistantResult

function renderPage(submitAssistant: SubmitAssistant) {
  return render(
    <MemoryRouter initialEntries={['/assistant']}>
      <AssistantPage submitAssistant={submitAssistant} />
    </MemoryRouter>,
  )
}

async function ask(submitAssistant: SubmitAssistant, text = '  Покажи мои счета  ') {
  const user = userEvent.setup()
  renderPage(submitAssistant)
  await user.type(screen.getByLabelText('Вопрос'), text)
  await user.click(screen.getByRole('button', { name: 'Спросить Grace' }))
  return user
}

describe('AssistantPage', () => {
  it('не отправляет запрос при открытии и валидирует пустой ввод', async () => {
    const submit = vi.fn<SubmitAssistant>()
    const user = userEvent.setup()
    renderPage(submit)

    expect(submit).not.toHaveBeenCalled()
    expect(screen.getByRole('link', { name: 'Ассистент' })).toHaveAttribute('aria-current', 'page')
    await user.click(screen.getByRole('button', { name: 'Спросить Grace' }))

    expect(await screen.findByText('Введите вопрос.')).toBeInTheDocument()
    expect(submit).not.toHaveBeenCalled()
  })

  it('отправляет trimmed текст, ru и reference time с явным offset', async () => {
    const submit = vi.fn<SubmitAssistant>().mockResolvedValue(accountResult)
    await ask(submit)

    expect(submit).toHaveBeenCalledOnce()
    const [query, signal] = submit.mock.calls[0]
    expect(query.utterance).toBe('Покажи мои счета')
    expect(query.language).toBe('ru')
    expect(query.referenceTime).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{2}:\d{2}$/)
    expect(signal).toBeInstanceOf(AbortSignal)
    expect(await screen.findByText('Основной счёт')).toBeInTheDocument()
    expect(screen.getByText(/250,50/)).toBeInTheDocument()
  })

  it.each([
    [obligationsResult, 'Рассрочка'],
    [receiptsResult, 'Магазин'],
    [incomePeriodResult, 'Точка отсчёта'],
  ] as const)('показывает typed completed result %#', async (response, expectedText) => {
    const submit = vi.fn<SubmitAssistant>().mockResolvedValue(response)
    await ask(submit)
    expect(await screen.findByText(expectedText)).toBeInTheDocument()
  })

  it('показывает clarify с требуемыми аргументами', async () => {
    const submit = vi.fn<SubmitAssistant>().mockResolvedValue({
      status: 'clarify',
      route: 'get_period_summary',
      confidence: 0.82,
      clarificationCode: 'arguments_required',
      requiredArguments: ['from', 'to'],
    })
    await ask(submit)

    expect(await screen.findByText('Нужно уточнение')).toBeInTheDocument()
    expect(screen.getByText(/дата начала, дата окончания/)).toBeInTheDocument()
  })

  it.each([
    [{ status: 'unsupported', route: 'unsupported', confidence: 0.96 } satisfies AssistantResult, 'Запрос пока не поддерживается'],
    [{ status: 'unavailable', errorCode: 'assistant_unavailable' } satisfies AssistantResult, 'Ассистент недоступен'],
    [{ status: 'unavailable', errorCode: 'dependency_unavailable' } satisfies AssistantResult, 'Финансовые данные временно недоступны'],
  ] as const)('показывает terminal state %#', async (response, expectedText) => {
    const submit = vi.fn<SubmitAssistant>().mockResolvedValue(response)
    await ask(submit)
    expect(await screen.findByText(expectedText)).toBeInTheDocument()
  })

  it.each([
    [new AssistantRequestError('validation'), 'Проверьте текст вопроса'],
    [new AssistantRequestError('malformed'), 'Не удалось обработать ответ'],
    [new AssistantRequestError('network'), 'Не удалось связаться с ассистентом'],
  ] as const)('показывает безопасную ошибку transport %#', async (failure, expectedText) => {
    const submit = vi.fn<SubmitAssistant>().mockRejectedValue(failure)
    await ask(submit)
    expect(await screen.findByText(new RegExp(expectedText))).toBeInTheDocument()
  })

  it('считает UTF-8 bytes и не отправляет oversized вопрос', async () => {
    const submit = vi.fn<SubmitAssistant>()
    await ask(submit, 'я'.repeat(1001))

    expect(await screen.findByText('Вопрос длиннее 2000 байт. Сократите его.')).toBeInTheDocument()
    expect(screen.getByText('2002 / 2000 байт')).toBeInTheDocument()
    expect(submit).not.toHaveBeenCalled()
  })

  it('блокирует двойной submit и заменяет предыдущий результат новым', async () => {
    let finishFirst: ((result: AssistantResult) => void) | undefined
    const first = new Promise<AssistantResult>((resolve) => { finishFirst = resolve })
    const submit = vi.fn<SubmitAssistant>()
      .mockReturnValueOnce(first)
      .mockResolvedValueOnce(receiptsResult)
    const user = userEvent.setup()
    renderPage(submit)
    const input = screen.getByLabelText('Вопрос')

    await user.type(input, 'Счета')
    const button = screen.getByRole('button', { name: 'Спросить Grace' })
    await user.click(button)
    await user.click(button)
    expect(submit).toHaveBeenCalledTimes(1)

    await act(async () => finishFirst?.(accountResult))
    expect(await screen.findByText('Основной счёт')).toBeInTheDocument()
    await user.clear(input)
    await user.type(input, 'Чеки')
    await user.click(screen.getByRole('button', { name: 'Спросить Grace' }))
    expect(await screen.findByText('Магазин')).toBeInTheDocument()
    expect(screen.queryByText('Основной счёт')).not.toBeInTheDocument()
  })

  it('отправляет по Ctrl+Enter, а обычный Enter оставляет новую строку', async () => {
    const submit = vi.fn<SubmitAssistant>().mockResolvedValue(accountResult)
    const user = userEvent.setup()
    renderPage(submit)
    const input = screen.getByLabelText('Вопрос')

    await user.type(input, 'Покажи{enter}счета')
    expect(submit).not.toHaveBeenCalled()
    await user.keyboard('{Control>}{Enter}{/Control}')
    expect(submit).toHaveBeenCalledOnce()
  })

  it('отменяет активный request при уходе со страницы', async () => {
    let capturedSignal: AbortSignal | undefined
    const submit = vi.fn<SubmitAssistant>((_query, signal) => {
      capturedSignal = signal
      return new Promise(() => undefined)
    })
    const user = userEvent.setup()
    const view = renderPage(submit)
    await user.type(screen.getByLabelText('Вопрос'), 'Покажи счета')
    await user.click(screen.getByRole('button', { name: 'Спросить Grace' }))

    expect(capturedSignal?.aborted).toBe(false)
    view.unmount()
    expect(capturedSignal?.aborted).toBe(true)
  })
})
