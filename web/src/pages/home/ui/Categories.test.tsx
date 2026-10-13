import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Category } from '@/shared/api/types'
import { Categories } from './Categories'

const api = vi.hoisted(() => ({ POST: vi.fn(), PATCH: vi.fn(), PUT: vi.fn(), DELETE: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: api }))
beforeEach(() => vi.resetAllMocks())

const base = { kind: 'expense' as const, createdAt: '', hidden: false, operationCount: 0, icon: null }
const items: Category[] = [
  { ...base, id: 'global', name: 'Продукты', editable: false, icon: 'shopping-cart', operationCount: 12 },
  { ...base, id: 'own', name: 'Спорт', editable: true, icon: 'dumbbell', operationCount: 3 },
  { ...base, id: 'empty', name: 'Пустая', editable: true },
]
function show(list = items) { render(<QueryClientProvider client={new QueryClient()}><Categories items={list} /></QueryClientProvider>) }

describe('SPEC-0042 categories', () => {
  it('creates a personal category with an icon and never offers to edit a global one', async () => {
    show()
    expect(screen.queryByRole('button', { name: 'Изменить Продукты' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Удалить Продукты' })).not.toBeInTheDocument()
    expect(screen.getByText('Общая категория · 12 операций')).toBeVisible()
    await userEvent.click(screen.getByRole('button', { name: 'Добавить категорию' }))
    await userEvent.type(screen.getByLabelText('Название категории'), ' Тренировки ')
    await userEvent.click(screen.getByRole('radio', { name: 'Спорт' }))
    api.POST.mockResolvedValue({ data: { id: 'new' } })
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))
    await waitFor(() => expect(api.POST).toHaveBeenCalledOnce())
    expect(api.POST.mock.calls[0][1].body).toEqual({ name: 'Тренировки', icon: 'dumbbell' })
  })

  it('changes name and icon of the same ID and keeps the form on a conflict', async () => {
    show()
    await userEvent.click(screen.getByRole('button', { name: 'Изменить Спорт' }))
    expect(screen.getByRole('radio', { name: 'Спорт' })).toHaveAttribute('aria-checked', 'true')
    await userEvent.clear(screen.getByLabelText('Название категории'))
    await userEvent.type(screen.getByLabelText('Название категории'), 'Кафе')
    await userEvent.click(screen.getByRole('radio', { name: 'Кафе и рестораны' }))
    api.PATCH.mockResolvedValue({ error: { message: 'Такая категория уже существует.' } })
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('уже существует')
    expect(api.PATCH.mock.calls[0][1]).toMatchObject({ params: { path: { categoryId: 'own' } }, body: { name: 'Кафе', icon: 'utensils' } })
    expect(screen.getByLabelText('Название категории')).toHaveValue('Кафе')
  })

  it('deletes a category only after a move target is chosen and passes the preview count', async () => {
    show()
    await userEvent.click(screen.getByRole('button', { name: 'Удалить Спорт' }))
    const confirm = screen.getByRole('button', { name: 'Удалить категорию' })
    expect(confirm).toBeDisabled()
    await userEvent.click(screen.getByRole('combobox'))
    await userEvent.click(screen.getByRole('option', { name: 'Продукты' }))
    expect(screen.getByText('Перенесём 3 операции в «Продукты». Суммы и даты не изменятся.')).toBeVisible()
    api.POST.mockResolvedValue({ data: { operations: 3 } })
    api.DELETE.mockResolvedValue({ data: { movedOperations: 3 } })
    await userEvent.click(confirm)
    await waitFor(() => expect(api.DELETE).toHaveBeenCalledOnce())
    expect(api.DELETE.mock.calls[0][1].params).toMatchObject({ path: { categoryId: 'own' }, query: { expectedOperations: 3, moveTo: 'global' } })
  })

  it('stops a deletion when the operation count changed since the list loaded', async () => {
    show()
    await userEvent.click(screen.getByRole('button', { name: 'Удалить Спорт' }))
    await userEvent.click(screen.getByRole('combobox'))
    await userEvent.click(screen.getByRole('option', { name: 'Продукты' }))
    api.POST.mockResolvedValue({ data: { operations: 4 } })
    await userEvent.click(screen.getByRole('button', { name: 'Удалить категорию' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Количество операций изменилось')
    expect(api.DELETE).not.toHaveBeenCalled()
  })

  it('deletes an empty category without a move target', async () => {
    show()
    await userEvent.click(screen.getByRole('button', { name: 'Удалить Пустая' }))
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    api.POST.mockResolvedValue({ data: { operations: 0 } })
    api.DELETE.mockResolvedValue({ data: { movedOperations: 0 } })
    await userEvent.click(screen.getByRole('button', { name: 'Удалить категорию' }))
    await waitFor(() => expect(api.DELETE.mock.calls[0][1].params.query).toEqual({ expectedOperations: 0 }))
  })

  it('hides and shows a shared category', async () => {
    show([{ ...items[0] }, { ...base, id: 'shown', name: 'Транспорт', editable: false, hidden: true }])
    expect(screen.getByText('Скрыта из выбора')).toBeVisible()
    api.PUT.mockResolvedValue({ data: {} })
    await userEvent.click(screen.getByRole('button', { name: 'Скрыть' }))
    await waitFor(() => expect(api.PUT.mock.calls[0][1].params.path).toEqual({ categoryId: 'global' }))
    api.DELETE.mockResolvedValue({ data: {} })
    await userEvent.click(screen.getByRole('button', { name: 'Показать' }))
    await waitFor(() => expect(api.DELETE.mock.calls[0][0]).toBe('/api/v1/categories/{categoryId}/hidden'))
  })
})
