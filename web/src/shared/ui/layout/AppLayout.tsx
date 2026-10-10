import { AuthContext } from '@/shared/model/auth-context'
import { env } from '@/shared/config/env'
import { useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import {
  Bot,
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
  Tags,
  Repeat2,
} from 'lucide-react'
import { NavLink, Link, useLocation } from 'react-router-dom'
import { useTheme } from 'next-themes'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { GraceMark } from '@/shared/ui/GraceMark'
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
  { to: '/subscriptions', label: 'Подписки', icon: Repeat2 },
  { to: '/categories', label: 'Категории', icon: Tags },
  { to: '/imports', label: 'Импорт', icon: Import },
  { to: '/accounts', label: 'Счета и карты', icon: WalletCards },
  { to: '/analytics', label: 'Статистика', icon: BarChart3 },
  { to: '/assistant', label: 'Ассистент', icon: Bot },
  { to: '/health', label: 'Финансовое здоровье', icon: TableProperties },
  { to: '/profile', label: 'Профиль и доступ', icon: UserRound },
]
// Keep in sync with Tailwind's lg breakpoint used by the shell and bottom bar.
const desktopQuery = '(min-width: 64rem)'
export function AppLayout({ children }: { children: ReactNode }) {
  const auth = useContext(AuthContext)
  const [open, setOpen] = useState(false)
  const [desktop, setDesktop] = useState(() => window.matchMedia?.(desktopQuery).matches ?? false)
  const [keyboard, setKeyboard] = useState(false)
  const { resolvedTheme, setTheme } = useTheme()
  const { pathname } = useLocation()
  const mainRef = useRef<HTMLElement>(null)
  useEffect(() => {
    const query = window.matchMedia?.(desktopQuery)
    if (!query) return
    const update = () => {
      setDesktop(query.matches)
      // Unmount the Radix portal too: hiding its panel would retain the modal lock.
      setOpen(false)
    }
    query.addEventListener('change', update)
    return () => query.removeEventListener('change', update)
  }, [])
  useEffect(() => {
    window.scrollTo(0, 0)
    mainRef.current?.focus({ preventScroll: true })
  }, [pathname])
  useEffect(() => {
    const viewport = window.visualViewport
    if (!viewport) return
    const update = () => setKeyboard(window.innerHeight - viewport.height > 150)
    viewport.addEventListener('resize', update)
    return () => viewport.removeEventListener('resize', update)
  }, [])
  const allLinks = [...mainLinks, ...moreLinks]
  const navClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex min-h-11 items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-ring',
      isActive
        ? 'bg-accent text-primary'
        : 'text-muted-foreground hover:bg-accent hover:text-foreground',
    )
  return (
    <div className="min-h-dvh lg:pl-60">
      <a href="#main-content" className="sr-only fixed top-3 left-3 z-[100] rounded-xl bg-primary px-4 py-3 text-primary-foreground focus:not-sr-only">Перейти к содержимому</a>
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-60 flex-col overflow-y-auto border-r border-border/70 bg-card p-5 lg:flex">
        <Link to="/" aria-label="Grace — обзор" className="mb-8 flex min-h-11 items-center gap-3 px-2 text-xl font-semibold tracking-tight"><GraceMark className="size-9 text-primary" aria-hidden="true"/>Grace</Link>
        <p className="mb-3 px-3 text-xs font-medium tracking-wide text-muted-foreground">Личное пространство</p>
        <nav aria-label="Навигация" className="grid gap-1">{allLinks.map(({icon:Icon,...link})=><NavLink key={link.to} to={link.to} end={link.to==='/'} className={navClass}><Icon className="size-[18px]" aria-hidden="true"/>{link.label}</NavLink>)}</nav>
        <div className="mt-auto border-t pt-4"><Button variant="ghost" className="w-full justify-start text-muted-foreground" onClick={()=>setTheme(resolvedTheme==='dark'?'light':'dark')}>{resolvedTheme==='dark'?<Sun className="size-[18px]"/>:<Moon className="size-[18px]"/>}{resolvedTheme==='dark'?'Светлая тема':'Тёмная тема'}</Button>{auth && <p className="mt-3 truncate px-3 text-sm text-muted-foreground" title={auth.session.user.name}>{auth.session.user.name}</p>}</div>
      </aside>
      <header className="sticky top-0 z-30 border-b border-border/70 bg-background/95 backdrop-blur-sm">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2.5 sm:px-6">
          <Link to="/" aria-label="Grace — обзор" className="flex min-h-11 items-center gap-2.5 text-xl font-semibold tracking-tight lg:hidden"><GraceMark aria-hidden="true" className="size-8 text-primary"/>Grace</Link>
          <span className="hidden text-sm text-muted-foreground lg:block">Мои финансы</span>
          {!desktop && <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" aria-label="Меню" className="lg:hidden">
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
            <DialogContent data-motion="menu" onCloseAutoFocus={event => {
              if (window.matchMedia?.(desktopQuery).matches) {
                event.preventDefault()
                mainRef.current?.focus({ preventScroll: true })
              }
            }} className="left-auto right-0 top-0 h-dvh max-h-dvh max-w-[min(350px,calc(100%-1rem))] translate-x-0 translate-y-0 rounded-r-none sm:max-w-[350px]">
              <div>
                <DialogHeader>
                  <DialogTitle>Меню</DialogTitle>
                  <DialogDescription>Личное пространство</DialogDescription>
                </DialogHeader>
                <nav
                  aria-label="Дополнительные разделы"
                  className="mt-6 grid gap-1"
                >
                  {allLinks.map(({ icon: Icon, ...link }) => (
                    <NavLink
                      key={link.to}
                      to={link.to}
                      end={link.to === '/'}
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
          </Dialog>}
        </div>
      </header>
      <main ref={mainRef} tabIndex={-1} id="main-content" className="grace-page mx-auto w-full max-w-6xl px-4 pt-7 pb-[calc(6rem+env(safe-area-inset-bottom))] outline-none sm:px-6 sm:pt-8 lg:pb-12">
        {children}
      </main>
      {!keyboard && (
        <nav
          aria-label="Разделы на телефоне"
          className="fixed inset-x-0 bottom-0 z-40 border-t bg-card px-3 pt-2 pb-[calc(.5rem+env(safe-area-inset-bottom))] lg:hidden"
        >
          <div className="mx-auto grid max-w-md grid-cols-3 gap-2">
            {mainLinks.map(({ icon: Icon, ...link }) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.to === '/'}
                className={({ isActive }) =>
                  cn(navClass({ isActive }), 'flex-col justify-center gap-1 px-1 text-xs')
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
