import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { AuthContext } from '@/shared/model/auth-context'
import { AuthActions } from './AuthActions'
const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))
function mount(role: 'admin' | 'member') { return render(<AuthContext.Provider value={{ session: { mode: 'password', csrfToken: 'synthetic', user: { id: 'synthetic', name: 'Synthetic', role } }, logout: vi.fn(), revokeSessions: vi.fn() }}><AuthActions /></AuthContext.Provider>) }
describe('SPEC-0031 bound invitations', () => {
  it('creates a bound code/link and copies only on request, retaining the real recipient', async () => {
    const user = userEvent.setup()
    const copy = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
    post.mockResolvedValue({ data: { token: 'i'.repeat(43), url: 'https://grace.test/register#invitation=' + 'i'.repeat(43), email: 'person@example.test', expiresAt: '2026-10-10T00:00:00Z' } })
    mount('admin')
    await user.click(screen.getByRole('button', { name: 'Доступ' }))
    await user.type(screen.getByLabelText('Имя приглашённого пользователя'), 'Person')
    await user.type(screen.getByLabelText('Почта приглашённого'), 'person@example.test')
    await user.click(screen.getByRole('button', { name: 'Создать приглашение' }))
    expect(await screen.findByLabelText('Ссылка для регистрации')).toHaveValue('https://grace.test/register#invitation=' + 'i'.repeat(43))
    expect(copy).not.toHaveBeenCalled()
    await user.clear(screen.getByLabelText('Почта приглашённого'))
    await user.type(screen.getByLabelText('Почта приглашённого'), 'another@example.test')
    expect(screen.getByText(/Только для person@example.test/)).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Скопировать ссылку' }))
    expect(copy).toHaveBeenCalledWith('https://grace.test/register#invitation=' + 'i'.repeat(43))
    expect(post).toHaveBeenCalledTimes(1)
    copy.mockRestore()
  })
  it('does not show invitation management to members', async () => {
    mount('member'); await userEvent.click(screen.getByRole('button', { name: 'Доступ' }))
    expect(screen.queryByRole('button', { name: 'Создать приглашение' })).not.toBeInTheDocument()
  })
})
