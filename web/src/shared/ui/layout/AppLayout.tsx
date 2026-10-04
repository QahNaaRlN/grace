import type { ReactNode } from 'react'

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
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700">
              Grace
            </p>
            <h1 className="text-lg font-semibold">Финансовое здоровье</h1>
          </div>
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
