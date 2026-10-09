import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PasswordForm } from './PasswordEntry'
const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))
function mount(view: 'login' | 'register' | 'forgot' | 'reset') { return render(<MemoryRouter><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><Routes><Route path="/" element={<PasswordForm view={view} />} /><Route path="/login" element={<p>Login destination</p>} /></Routes></QueryClientProvider></MemoryRouter>) }
beforeEach(() => { post.mockReset(); window.history.replaceState(null, '', '/'); })
describe('SPEC-0031 password forms', () => {
  it('logs in with only email and password and clears cached user state', async () => {
    post.mockResolvedValue({ data: { user: { id: 'owner' }, csrfToken: 'synthetic', mode: 'password' } })
    mount('login')
    expect(screen.queryByLabelText('Код приглашения')).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Почта'), 'person@example.test')
    await userEvent.type(screen.getByLabelText('Пароль'), 'a synthetic passphrase')
    await userEvent.click(screen.getByRole('button', { name: 'Войти' }))
    await waitFor(() => expect(post).toHaveBeenCalledWith('/api/auth/password/login', { body: { email: 'person@example.test', password: 'a synthetic passphrase' } }))
  })
  it('prefills invitation from the fragment without carrying it to the login', async () => {
    const invite = 'x'.repeat(43); window.history.replaceState(null, '', '/#invitation=' + invite)
    post.mockResolvedValue({ data: { resendAfterSeconds: 60 } })
    mount('register')
    expect(screen.getByLabelText('Код приглашения')).toHaveValue(invite)
    expect(window.location.hash).toBe('')
    await userEvent.type(screen.getByLabelText('Почта'), 'invited@example.test')
    await userEvent.type(screen.getByLabelText('Пароль'), 'a synthetic passphrase')
    await userEvent.type(screen.getByLabelText('Повторите пароль'), 'a synthetic passphrase')
    await userEvent.click(screen.getByRole('button', { name: 'Зарегистрироваться' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Приглашение пока не использовано')
    expect(post.mock.calls[0][1].body.invitation).toBe(invite)
    expect(screen.getByRole('button', { name: /Повторить через/ })).toBeDisabled()
  })
  it('does not consume a mail token until explicit confirmation', async () => {
    const token = 't'.repeat(43); window.history.replaceState(null, '', '/#token=' + token)
    post.mockResolvedValue({ response: new Response(null, { status: 204 }) })
    mount('register')
    expect(post).not.toHaveBeenCalled()
    await userEvent.click(screen.getByRole('button', { name: 'Подтвердить почту' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Аккаунт готов')
    expect(post).toHaveBeenCalledWith('/api/auth/password/confirm', { body: { token } })
  })
  it('keeps reset secrets out of the URL and rejects mismatching passwords', async () => {
    const token = 'r'.repeat(43); window.history.replaceState(null, '', '/#token=' + token)
    post.mockResolvedValue({ response: new Response(null, { status: 204 }) })
    mount('reset')
    expect(window.location.hash).toBe('')
    await userEvent.type(screen.getByLabelText('Новый пароль'), 'a synthetic passphrase')
    await userEvent.type(screen.getByLabelText('Повторите пароль'), 'another synthetic phrase')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить пароль' }))
    expect(screen.getByRole('alert')).toHaveTextContent('должны совпадать')
    expect(post).not.toHaveBeenCalled()
    await userEvent.clear(screen.getByLabelText('Повторите пароль'))
    await userEvent.type(screen.getByLabelText('Повторите пароль'), 'a synthetic passphrase')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить пароль' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Старые сеансы завершены')
  })
  it('requests recovery with only email and a generic result', async () => {
    post.mockResolvedValue({ data: { resendAfterSeconds: 60 } })
    mount('forgot')
    await userEvent.type(screen.getByLabelText('Почта'), 'person@example.test')
    await userEvent.click(screen.getByRole('button', { name: 'Получить ссылку' }))
    expect(await screen.findByRole('status')).toHaveTextContent('Если почта зарегистрирована')
    expect(post).toHaveBeenCalledWith('/api/auth/password/forgot', { body: { email: 'person@example.test' } })
  })
})
