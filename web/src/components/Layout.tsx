import { useEffect } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Flame, ImageIcon, Plus, Search, Trophy } from './Icons'
import { cn } from '../lib/utils'

const navItems = [
  { to: '/', label: 'Feed', end: true },
  { to: '/templates', label: 'Templates' },
  { to: '/top', label: 'Leaderboard' },
]

const mobileNav = [
  { to: '/', label: 'Feed', icon: Flame, end: true },
  { to: '/templates', label: 'Templates', icon: ImageIcon },
  { to: '/create', label: 'Create', icon: Plus, accent: true },
  { to: '/top', label: 'Top', icon: Trophy },
]

export default function Layout() {
  const navigate = useNavigate()
  const { pathname } = useLocation()

  useEffect(() => {
    window.scrollTo(0, 0)
  }, [pathname])

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== '/' || event.metaKey || event.ctrlKey || event.altKey) return
      const target = event.target as HTMLElement | null
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.tagName === 'SELECT' ||
          target.isContentEditable)
      ) {
        return
      }
      event.preventDefault()
      navigate('/search')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [navigate])

  return (
    <div className="relative flex min-h-dvh flex-col">
      <div className="grain" aria-hidden="true" />

      <header className="sticky top-0 z-50 border-b border-white/[0.06] bg-ink-950/80 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-[2000px] items-center gap-4 px-4 sm:px-6 lg:px-8">
          <NavLink to="/" className="group flex shrink-0 items-center gap-2.5">
            <span className="grid size-9 place-items-center rounded-xl bg-gradient-to-br from-acid-400 via-ice-400 to-hot-500 font-display text-xl leading-none text-ink-950 shadow-[0_0_24px_-6px_rgb(200_255_77/0.55)] transition group-hover:rotate-[-4deg]">
              M
            </span>
            <span className="text-[17px] font-bold tracking-tight text-white">
              meme<span className="text-acid-400">bank</span>
            </span>
          </NavLink>

          <nav className="hidden flex-1 items-center gap-0.5 md:flex">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) => cn('nav-link', isActive && 'nav-link-active')}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>

          <div className="ml-auto flex shrink-0 items-center gap-2 md:ml-0">
            <NavLink
              to="/search"
              title="Search memes and templates (press /)"
              aria-label="Search memes and templates"
              className="flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-2.5 py-2 text-xs font-medium text-zinc-400 transition hover:border-white/25 hover:bg-white/[0.08] hover:text-white"
            >
              <Search className="size-4" />
              <span className="hidden lg:inline">Search</span>
              <kbd className="hidden rounded border border-white/10 px-1 text-[10px] text-zinc-600 lg:inline">
                /
              </kbd>
            </NavLink>
            <div className="hidden sm:block">
              <NavLink to="/create" className="btn-primary px-4 py-2 sm:px-5 sm:py-2.5">
                <Plus className="size-4" />
                New meme
              </NavLink>
            </div>
          </div>
        </div>
      </header>

      <main className="relative z-10 mx-auto w-full max-w-[2000px] flex-1 px-4 pt-6 pb-28 sm:px-6 md:pb-20 lg:px-8">
        <Outlet />
      </main>

      <footer className="relative z-10 border-t border-white/[0.06] pt-8 pb-28 md:pb-8">
        <div className="mx-auto flex max-w-[2000px] flex-col items-center justify-between gap-3 px-4 text-xs text-zinc-600 sm:flex-row sm:px-6 lg:px-8">
          <p>
            <span className="font-semibold text-zinc-400">memebank</span> — make it, post it, get
            roasted.
          </p>
          <p className="flex items-center gap-2">
            <span>Go + React</span>
            <span className="size-1 rounded-full bg-zinc-700" />
            <span>No accounts. No mercy.</span>
          </p>
        </div>
      </footer>

      <nav className="fixed inset-x-0 bottom-0 z-40 border-t border-white/[0.08] bg-ink-950/95 pb-[env(safe-area-inset-bottom)] backdrop-blur-xl md:hidden">
        <div className="mx-auto grid max-w-md grid-cols-4 px-2 pt-2 pb-2">
          {mobileNav.map((item) => {
            const Icon = item.icon
            return (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  cn(
                    'flex flex-col items-center gap-1 rounded-xl py-0.5 text-[10px] font-semibold transition',
                    item.accent
                      ? 'text-acid-400'
                      : isActive
                        ? 'text-acid-400'
                        : 'text-zinc-500 hover:text-zinc-300',
                  )
                }
              >
                <span
                  className={cn(
                    'grid size-8 place-items-center rounded-full transition',
                    item.accent
                      ? 'bg-acid-400 text-ink-950 shadow-[0_6px_20px_-4px_rgb(200_255_77/0.7)]'
                      : 'bg-white/[0.04]',
                  )}
                >
                  <Icon className="size-4" />
                </span>
                {item.label}
              </NavLink>
            )
          })}
        </div>
      </nav>
    </div>
  )
}
