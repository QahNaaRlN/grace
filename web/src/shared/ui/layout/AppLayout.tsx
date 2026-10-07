import { AuthContext } from '@/shared/model/auth-context'
import { env } from '@/shared/config/env'
import { useContext, useEffect, useState, type ReactNode } from 'react'
import {
  Bot,
  HeartPulse,
  LayoutDashboard,
  ListOrdered,
  CalendarClock,
  Menu,
  WalletCards,
  BarChart3,
  UserRound,
  Moon,
  Sun,
  TableProperties,
  Import,
} from 'lucide-react'
import { NavLink, Link } from 'react-router-dom'
import { useTheme } from 'next-themes'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogTrigger,
} from '@/shared/ui/dialog'

const mainLinks = [
  { to: '/', label: 'Обзор', icon: LayoutDashboard },
  { to: '/operations', label: 'Операции', icon: ListOrdered },
  { to: '/payments', label: 'Платежи', icon: CalendarClock },
]
const moreLinks = [
  { to: '/imports', label: 'Импорт', icon: Import },
  { to: '/accounts', label: 'Счета и карты', icon: WalletCards },
  { to: '/analytics', label: 'Подробный расчёт', icon: BarChart3 },
  { to: '/assistant', label: 'Ассистент', icon: Bot },
  { to: '/health', label: 'Финансовое здоровье', icon: TableProperties },
  { to: '/profile', label: 'Профиль и доступ', icon: UserRound },
]
export function AppLayout({ children }: { children: ReactNode }) {
  const auth = useContext(AuthContext)
  const [open, setOpen] = useState(false)
  const [keyboard, setKeyboard] = useState(false)
  const { resolvedTheme, setTheme } = useTheme()
  useEffect(() => {
    const viewport = window.visualViewport
    if (!viewport) return
    const update = () => setKeyboard(window.innerHeight - viewport.height > 150)
    viewport.addEventListener('resize', update)
    return () => viewport.removeEventListener('resize', update)
  }, [])
  const navClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex min-h-11 items-center justify-center gap-2 rounded-xl px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-ring',
      isActive
        ? 'bg-accent text-primary'
        : 'text-muted-foreground hover:bg-accent hover:text-foreground',
    )
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-40 border-b bg-card">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <Link
            to="/"
            aria-label="Grace — обзор"
            className="flex min-h-11 items-center gap-3 text-xl font-semibold"
          >
            <span className="grid size-9 place-items-center rounded-xl bg-primary text-primary-foreground">
              <HeartPulse aria-hidden="true" className="size-5" />
            </span>
            Grace
          </Link>
          <nav
            aria-label="Основная навигация"
            className="mr-auto ml-6 hidden gap-2 md:flex"
          >
            {mainLinks.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.to === '/'}
                className={navClass}
              >
                {link.label}
              </NavLink>
            ))}
          </nav>
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" aria-label="Меню">
                {auth?.session.user.avatarUrl ? (
                  <img
                    src={env.VITE_API_BASE_URL + auth.session.user.avatarUrl}
                    alt=""
                    className="size-7 rounded-full object-cover"
                  />
                ) : (
                  <Menu aria-hidden="true" className="size-4" />
                )}
                <span>Меню</span>
              </Button>
            </DialogTrigger>
            <DialogContent data-motion="menu" className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(350px,calc(100%-1rem))] translate-x-0 translate-y-0 rounded-r-none sm:max-w-[350px]">
              <div>
                <DialogHeader>
                  <DialogTitle>Меню</DialogTitle>
                  <DialogDescription>Личное пространство</DialogDescription>
                </DialogHeader>
                <nav
                  aria-label="Дополнительные разделы"
                  className="mt-6 grid gap-1"
                >
                  {moreLinks.map(({ icon: Icon, ...link }) => (
                    <NavLink
                      key={link.to}
                      to={link.to}
                      onClick={() => setOpen(false)}
                      className={navClass}
                    >
                      <Icon aria-hidden="true" className="size-5" />
                      {link.label}
                    </NavLink>
                  ))}
                </nav>
                <div className="grace-actions">
                  <Button
                    variant="outline"
                    onClick={() =>
                      setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')
                    }
                  >
                    {resolvedTheme === 'dark' ? (
                      <Sun className="size-4" />
                    ) : (
                      <Moon className="size-4" />
                    )}
                    {resolvedTheme === 'dark' ? 'Светлая тема' : 'Тёмная тема'}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </header>
      <main className="grace-page mx-auto w-full max-w-6xl flex-1 px-4 pt-6 pb-[calc(6rem+env(safe-area-inset-bottom))] sm:px-6 md:pb-10">
        {children}
      </main>
      {!keyboard && (
        <nav
          aria-label="Разделы на телефоне"
          className="fixed inset-x-0 bottom-0 z-40 border-t bg-card px-3 pt-2 pb-[calc(.5rem+env(safe-area-inset-bottom))] md:hidden"
        >
          <div className="mx-auto grid max-w-md grid-cols-3 gap-2">
            {mainLinks.map(({ icon: Icon, ...link }) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.to === '/'}
                className={({ isActive }) =>
                  cn(navClass({ isActive }), 'flex-col gap-1 px-1 text-xs')
                }
              >
                <Icon aria-hidden="true" className="size-5" />
                {link.label}
              </NavLink>
            ))}
          </div>
        </nav>
      )}
    </div>
  )
}
