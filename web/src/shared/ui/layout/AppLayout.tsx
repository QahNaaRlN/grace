import type { ReactNode } from 'react'
import { Bot, HeartPulse, WalletCards, Import } from 'lucide-react'
import { NavLink } from 'react-router-dom'

import { cn } from '@/shared/lib/utils'
import { AuthActions } from './AuthActions'

type AppLayoutProps = {
  children: ReactNode
}

/**
 * Компонент, который задаёт семантическую оболочку приложения (header, main, footer).
 */
export function AppLayout({ children }: AppLayoutProps) {
  const navClass = ({ isActive }: { isActive: boolean }) => cn(
    'flex min-h-11 items-center justify-center gap-2 rounded-xl px-3 py-2 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/40',
    isActive
      ? 'bg-white text-primary shadow-sm ring-1 ring-foreground/5'
      : 'text-muted-foreground hover:bg-white/60 hover:text-foreground',
  )

  return (
    <div className="flex min-h-screen flex-col text-foreground">
      <header className="sticky top-0 z-40 border-b border-border/70 bg-background/95">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="grid size-10 place-items-center rounded-2xl bg-primary text-primary-foreground shadow-sm">
              <HeartPulse className="size-5" aria-hidden="true" />
            </div>
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.18em] text-primary">Grace</p>
              <p className="text-sm font-semibold sm:text-base">Финансовое здоровье</p>
            </div>
          </div>
          <AuthActions />
          <nav
            aria-label="Основная навигация"
            className="fixed inset-x-0 bottom-0 z-50 border-t border-border/70 bg-background/90 px-3 pt-2 pb-[calc(0.5rem+env(safe-area-inset-bottom))] backdrop-blur-xl md:static md:border-0 md:bg-transparent md:p-0 md:backdrop-blur-none"
          >
            <div className="mx-auto grid max-w-md grid-cols-3 gap-1 rounded-2xl bg-muted/80 p-1 md:flex">
              <NavLink to="/" end className={navClass}><WalletCards className="size-4" aria-hidden="true" />Финансы</NavLink>
              <NavLink to="/assistant" className={navClass}><Bot className="size-4" aria-hidden="true" />Ассистент</NavLink>
              <NavLink to="/imports" className={navClass}><Import className="size-4" aria-hidden="true" />Импорт</NavLink>
            </div>
          </nav>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 px-4 pt-5 pb-28 sm:px-6 sm:pt-8 md:pb-10">{children}</main>

      <footer className="hidden border-t border-border/70 bg-white/60 md:block">
        <div className="mx-auto max-w-6xl px-6 py-4 text-sm text-muted-foreground">
          Достоверные данные · детерминированные расчёты
        </div>
      </footer>
    </div>
  )
}
