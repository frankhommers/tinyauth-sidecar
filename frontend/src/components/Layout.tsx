import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { ThemeToggle } from './theme-toggle'
import { LanguageSelector } from './language-toggle'
import { cn } from '@/lib/utils'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/context/AuthContext'

type NavItem = {
  label: string
  path?: string
  href?: string
  onClick?: () => void
}

function NavItemRender({ item, className }: { item: NavItem; className: string }) {
  if (item.onClick) {
    return <button onClick={item.onClick} className={className}>{item.label}</button>
  }
  if (item.href) {
    return <a href={item.href} className={className}>{item.label}</a>
  }
  return (
    <NavLink
      to={item.path!}
      className={({ isActive }) =>
        cn(className, isActive && 'bg-primary text-primary-foreground')
      }
    >
      {item.label}
    </NavLink>
  )
}

export function Layout({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  const { loggedIn } = useAuth()

  const handleLogout = async () => {
    await fetch('/api/user/logout', { method: 'POST', credentials: 'include' }).catch(() => {})
    window.location.href = '/'
  }

  const navItems: NavItem[] = [
    ...(loggedIn
      ? [
          { label: t('nav.account'), path: '/account' },
          { label: t('nav.logout'), onClick: handleLogout },
        ]
      : [
          { label: t('nav.login'), href: '/' },
        ]),
    { label: t('nav.reset'), path: '/reset-password' },
  ]

  return (
    <div
      className="relative min-h-svh bg-cover bg-center"
      style={{ backgroundImage: 'url(/background.jpg)' }}
    >
      <div className="absolute inset-0 bg-black/45 dark:bg-black/55" />

      <div className="fixed top-4 right-4 z-20 flex items-center gap-2">
        <LanguageSelector />
        <ThemeToggle />
      </div>

      <header className="relative z-10">
        <div className="mx-auto flex max-w-5xl items-center justify-center px-4 py-4">
          <nav className="hidden sm:flex items-center gap-1 rounded-md border bg-card/75 p-1 backdrop-blur-md">
            {navItems.map((item) => (
              <NavItemRender
                key={item.label}
                item={item}
                className="rounded-sm px-3 py-1.5 text-sm transition-colors hover:bg-accent cursor-pointer"
              />
            ))}
          </nav>
        </div>
        <div className="mx-auto max-w-5xl px-4 sm:hidden">
          <nav className="flex items-center gap-1 rounded-md border bg-card/75 p-1 backdrop-blur-md">
            {navItems.map((item) => (
              <NavItemRender
                key={item.label}
                item={item}
                className="flex-1 rounded-sm px-2 py-1.5 text-center text-xs transition-colors hover:bg-accent cursor-pointer"
              />
            ))}
          </nav>
        </div>
      </header>

      <main className="relative z-10 mx-auto flex min-h-[calc(100svh-72px)] max-w-5xl items-center justify-center px-4 pb-8">
        <div className="w-full max-w-md">{children}</div>
      </main>
    </div>
  )
}
