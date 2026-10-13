import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { NewDraftCard } from './NewDraftCard'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const label = 'Таблица CSV или снимок JSON (до 16 MiB)'
const csv = () => new File(['date;amount\n'.repeat(60)], 'Операции.csv', { type: 'text/csv' })

beforeEach(() => post.mockReset())

describe('SPEC-0038 import file feedback', () => {
  it('AC1: shows the chosen file with name, type, size and a ready status, and can replace or remove it', async () => {
    render(<NewDraftCard onSaved={() => {}} />)
    const user = userEvent.setup()
    await user.upload(screen.getByLabelText(label), csv())
    expect(screen.getByText('Операции.csv')).toBeVisible()
    expect(screen.getByText(/Таблица CSV · 0,7 КБ/)).toBeVisible()
    expect(screen.getByRole('status')).toHaveTextContent('Файл выбран — готов к сохранению')
    expect(screen.getByRole('button', { name: 'Заменить файл' })).toBeEnabled()
    await user.click(screen.getByRole('button', { name: 'Убрать' }))
    expect(screen.queryByText('Операции.csv')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Сохранить черновик' })).toBeDisabled()
  })

  it('AC2: rejects a wrong type or an oversized file in words and sends nothing', async () => {
    render(<NewDraftCard onSaved={() => {}} />)
    const user = userEvent.setup({ applyAccept: false })
    await user.upload(screen.getByLabelText(label), new File(['x'], 'Выписка.xlsx'))
    expect(screen.getByText('Подходят только файлы CSV или JSON.')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Сохранить черновик' })).toBeDisabled()
    const big = new File(['x'], 'big.json', { type: 'application/json' })
    Object.defineProperty(big, 'size', { value: 16 * 1024 * 1024 + 1 })
    await user.click(screen.getByRole('button', { name: 'Заменить файл' }))
    await user.upload(screen.getByLabelText(label), big)
    expect(screen.getByText(/больше 16 МБ/)).toBeVisible()
    expect(screen.getByLabelText(label)).toHaveAttribute('aria-invalid', 'true')
    expect(post).not.toHaveBeenCalled()
  })

  it('AC3/AC4: saves once, says so while saving and confirms in the card without claiming operations', async () => {
    let finish: (value: unknown) => void = () => {}
    post.mockReturnValue(new Promise(resolve => { finish = resolve }))
    const onSaved = vi.fn()
    render(<NewDraftCard onSaved={onSaved} />)
    const user = userEvent.setup()
    await user.upload(screen.getByLabelText(label), csv())
    await user.dblClick(screen.getByRole('button', { name: 'Сохранить черновик' }))
    expect(screen.getByRole('button', { name: 'Сохраняем…' })).toBeDisabled()
    expect(screen.getByText('Сохраняем черновик на сервере…')).toBeVisible()
    expect(screen.getByLabelText(label)).toBeDisabled()
    finish({ data: { id: 'batch-1' } })
    expect(await screen.findByText('Черновик сохранён', { selector: 'p.font-semibold' })).toBeVisible()
    expect(screen.getByText(/Финансовые операции ещё не созданы/)).toBeVisible()
    expect(post).toHaveBeenCalledOnce()
    expect(onSaved).toHaveBeenCalledWith('batch-1')
  })

  it('AC4: shows a failure in the card and retries with the same key', async () => {
    post.mockResolvedValueOnce({ error: { code: 'invalid_csv' } }).mockResolvedValueOnce({ data: { id: 'batch-2' } })
    render(<NewDraftCard onSaved={() => {}} />)
    const user = userEvent.setup()
    await user.upload(screen.getByLabelText(label), csv())
    await user.click(screen.getByRole('button', { name: 'Сохранить черновик' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Не удалось прочитать CSV')
    await user.click(screen.getByRole('button', { name: 'Повторить' }))
    await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
    const key = (call: unknown[]) => (call[1] as { params: { header: Record<string, string> } }).params.header['Idempotency-Key']
    expect(key(post.mock.calls[0])).toBe(key(post.mock.calls[1]))
    expect(await screen.findByText('Черновик сохранён', { selector: 'p.font-semibold' })).toBeVisible()
  })

  it('AC5/AC6: announces states politely and shows the right cursor', async () => {
    render(<NewDraftCard onSaved={() => {}} />)
    const input = screen.getByLabelText(label)
    expect(screen.getByRole('status')).toHaveAttribute('aria-live', 'polite')
    expect(input).toHaveClass('cursor-pointer', 'file:cursor-pointer', 'disabled:cursor-not-allowed', 'disabled:file:cursor-not-allowed')
  })
})
