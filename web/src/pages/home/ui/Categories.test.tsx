import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, expect, it, vi } from 'vitest'
import { Categories } from './Categories'
const api = vi.hoisted(() => ({ POST: vi.fn(), PATCH: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: api }))
beforeEach(() => vi.resetAllMocks())
function show() { render(<QueryClientProvider client={new QueryClient()}><Categories items={[{id:'global',name:'Продукты',kind:'expense',createdAt:'',editable:false},{id:'own',name:'Спорт',kind:'expense',createdAt:'',editable:true}]} /></QueryClientProvider>) }
it('creates a personal category explicitly and never offers to rename a global one', async () => {
  show(); expect(screen.queryByRole('button',{name:'Переименовать Продукты'})).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button',{name:'Добавить категорию'}))
  expect(api.POST).not.toHaveBeenCalled()
  await userEvent.type(screen.getByLabelText('Название категории'),' Тренировки ')
  api.POST.mockResolvedValue({data:{id:'new'}})
  await userEvent.click(screen.getByRole('button',{name:'Сохранить'}))
  await waitFor(() => expect(api.POST).toHaveBeenCalledOnce())
  expect(api.POST.mock.calls[0][1].body).toEqual({name:'Тренировки'})
})
it('renames the same ID and keeps the form available on a conflict', async () => {
  show(); await userEvent.click(screen.getByRole('button',{name:'Переименовать Спорт'}))
  await userEvent.clear(screen.getByLabelText('Название категории')); await userEvent.type(screen.getByLabelText('Название категории'),'Кафе')
  api.PATCH.mockResolvedValue({error:{message:'Такая категория уже существует.'}})
  await userEvent.click(screen.getByRole('button',{name:'Сохранить'}))
  expect(await screen.findByRole('alert')).toHaveTextContent('уже существует')
  expect(api.PATCH.mock.calls[0][1].params.path).toEqual({categoryId:'own'})
  expect(screen.getByLabelText('Название категории')).toHaveValue('Кафе')
})
