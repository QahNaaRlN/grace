import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AppLayout } from './AppLayout'

afterEach(() => vi.unstubAllGlobals())

function viewport(initialDesktop: boolean) {
  const query = new EventTarget() as EventTarget & { matches: boolean }
  query.matches = initialDesktop
  vi.stubGlobal('matchMedia', vi.fn(() => query))
  return (desktop: boolean) => act(() => {
    query.matches = desktop
    query.dispatchEvent(new Event('change'))
  })
}

function layout() {
  render(<MemoryRouter><AppLayout><p>Содержимое страницы</p></AppLayout></MemoryRouter>)
}

describe('SPEC-0032 responsive navigation', () => {
  it('does not offer the mobile menu in desktop mode', () => {
    viewport(true)
    layout()
    expect(screen.queryByRole('button', { name: 'Меню' })).not.toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: 'Навигация' })).toBeInTheDocument()
  })

  it('releases an open modal on desktop transition and returns to mobile with it closed', async () => {
    const resize = viewport(false)
    layout()
    await userEvent.click(screen.getByRole('button', { name: 'Меню' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(document.body.style.pointerEvents).toBe('none')
    resize(true)
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(document.body.style.pointerEvents).not.toBe('none')
    expect(document.querySelector('[data-slot="dialog-overlay"]')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Меню' })).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('main')).toHaveFocus())
    resize(false)
    expect(screen.getByRole('button', { name: 'Меню' })).toHaveAttribute('aria-expanded', 'false')
    await userEvent.click(screen.getByRole('button', { name: 'Меню' }))
    const menu = screen.getByRole('dialog')
    await userEvent.click(within(menu).getByRole('link', { name: 'Счета и карты' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByRole('navigation', { name: 'Навигация' }).querySelector('a[aria-current="page"]')).toHaveAttribute('href', '/accounts')
  })
})
