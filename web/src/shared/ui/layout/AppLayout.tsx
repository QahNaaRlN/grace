import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'

import { cn } from '@/shared/lib/utils'

type AppLayoutProps = {
  children: ReactNode
}

/**
 * Компонент, который задаёт семантическую оболочку приложения (header, main, footer).
 */
export function AppLayout({ children }: AppLayoutProps) {
  return (
    <div className="flex min-h-screen flex-col bg-slate-50 text-slate-900">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-4 px-4 py-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700">
              Grace
            </p>
            <h1 className="text-lg font-semibold">Финансовое здоровье</h1>
          </div>
          <nav aria-label="Основная навигация" className="flex rounded-lg bg-slate-100 p-1 text-sm font-medium">
            <NavLink
              to="/"
              end
              className={({ isActive }) => cn('rounded-md px-3 py-2 transition-colors', isActive ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-600 hover:text-slate-950')}
            >
              Финансы
            </NavLink>
            <NavLink
              to="/assistant"
              className={({ isActive }) => cn('rounded-md px-3 py-2 transition-colors', isActive ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-600 hover:text-slate-950')}
            >
              Ассистент
            </NavLink>
          </nav>
        </div>
      </header>

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-8">{children}</main>

      <footer className="border-t border-slate-200 bg-white">
        <div className="mx-auto max-w-5xl px-4 py-4 text-sm text-slate-500">
          Достоверные данные · детерминированные расчёты
        </div>
      </footer>
    </div>
  )
}
