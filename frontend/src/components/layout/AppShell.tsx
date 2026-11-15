import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Bell, Boxes, ClipboardList, History, LayoutDashboard, LogOut, Menu, MoreHorizontal, Package, RotateCcw, ScrollText, SlidersHorizontal, Truck, Users, X } from 'lucide-react'
import { useAuth } from '../../auth/AuthProvider'
import { useT } from '../../i18n/I18nProvider'
import { notificationApi } from '../../api/endpoints'
import { navItems } from '../../lib/constants'
import { cn } from '../../lib/utils'
import { LanguageSwitcher } from '../../i18n/I18nProvider'
import type { Role } from '../../types'

const icons: Record<string, typeof LayoutDashboard> = {
  '/dashboard': LayoutDashboard,
  '/products': Package,
  '/inventory': Boxes,
  '/receiving': Truck,
  '/requests': ClipboardList,
  '/returns': RotateCcw,
  '/adjustments': SlidersHorizontal,
  '/reports': ScrollText,
  '/audit': History,
  '/users': Users,
  '/notifications': Bell,
}

const primaryPaths = ['/dashboard', '/products', '/inventory', '/requests']

export function AppShell() {
  const { user, logout } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const { data: unread } = useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: async () => (await notificationApi.unreadCount()).data.unread_count,
    enabled: Boolean(user),
    refetchInterval: 30_000,
  })

  useEffect(() => {
    setMenuOpen(false)
  }, [location.pathname])

  useEffect(() => {
    if (!menuOpen) return
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMenuOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.body.style.overflow = previous
      document.removeEventListener('keydown', onKey)
    }
  }, [menuOpen])

  if (!user) return null
  const items = navItems.filter((item) => item.roles.includes(user.role as Role))
  const primaryItems = items.filter((item) => primaryPaths.includes(item.to))

  const signOut = () => {
    setMenuOpen(false)
    logout()
    navigate('/login')
  }

  const navLinkClass = (isActive: boolean) =>
    cn(
      'flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors',
      isActive ? 'bg-brand-50 text-brand-700' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900',
    )

  return (
    <div className="flex min-h-full bg-slate-100">
      <aside className="hidden w-60 shrink-0 flex-col border-r border-slate-200 bg-white md:flex">
        <div className="flex h-16 items-center gap-2 border-b border-slate-200 px-5">
          <div className="grid h-8 w-8 place-items-center rounded-lg bg-brand-600 text-sm font-bold text-white">W</div>
          <div>
            <p className="text-sm font-semibold text-slate-900">WMS</p>
            <p className="text-[11px] text-slate-500">{t('common.warehouseManagement')}</p>
          </div>
        </div>
        <nav className="flex-1 space-y-1 overflow-y-auto p-3">
          {items.map((item) => {
            const Icon = icons[item.to] ?? LayoutDashboard
            return (
              <NavLink key={item.to} to={item.to} className={({ isActive }) => navLinkClass(isActive)}>
                <Icon className="h-4 w-4" />
                {t(item.key)}
                {item.to === '/notifications' && Boolean(unread) && (
                  <span className="ml-auto rounded-full bg-red-100 px-1.5 text-xs font-semibold text-red-700">{unread}</span>
                )}
              </NavLink>
            )
          })}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-14 items-center justify-between gap-3 border-b border-slate-200 bg-white px-3 sm:h-16 sm:px-4 md:px-6">
          <div className="flex items-center gap-2 md:hidden">
            <button
              type="button"
              onClick={() => setMenuOpen(true)}
              className="grid h-10 w-10 place-items-center rounded-md text-slate-600 hover:bg-slate-100"
              aria-label={t('actions.openMenu')}
            >
              <Menu className="h-5 w-5" />
            </button>
            <div className="grid h-8 w-8 place-items-center rounded-lg bg-brand-600 text-sm font-bold text-white">W</div>
            <span className="font-semibold">WMS</span>
          </div>
          <div className="hidden text-sm text-slate-500 md:block">
            {t('common.signedInAs', { email: user.email })}
          </div>
          <div className="flex items-center gap-1.5 sm:gap-3">
            <button
              onClick={() => navigate('/notifications')}
              className="relative grid h-10 w-10 place-items-center rounded-md text-slate-500 hover:bg-slate-100"
              aria-label={t('nav.notifications')}
            >
              <Bell className="h-5 w-5" />
              {Boolean(unread) && <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-red-500" />}
            </button>
            <LanguageSwitcher className="hidden md:block" />
            <div className="hidden text-right md:block">
              <p className="text-sm font-medium text-slate-800">{user.full_name}</p>
              <p className="text-xs text-slate-500">
                {t(`role.${user.role}`)}
                {user.department ? ` · ${user.department.name}` : ''}
              </p>
            </div>
            <button
              onClick={signOut}
              className="hidden h-10 w-10 place-items-center rounded-md text-slate-500 hover:bg-slate-100 md:grid"
              aria-label={t('common.signOut')}
            >
              <LogOut className="h-5 w-5" />
            </button>
          </div>
        </header>
        <main className="flex-1 p-3 pb-[calc(4.5rem+env(safe-area-inset-bottom))] sm:p-4 sm:pb-[calc(4.5rem+env(safe-area-inset-bottom))] md:p-6">
          <Outlet />
        </main>
      </div>

      {menuOpen && (
        <div className="fixed inset-0 z-50 md:hidden" role="dialog" aria-modal="true" aria-label={t('nav.mainNavigation')}>
          <button
            type="button"
            aria-label={t('actions.closeMenu')}
            className="absolute inset-0 h-full w-full bg-slate-900/40"
            onClick={() => setMenuOpen(false)}
          />
          <div className="absolute left-0 top-0 flex h-full w-72 max-w-[85vw] flex-col bg-white shadow-xl">
            <div className="flex h-16 items-center justify-between border-b border-slate-200 px-4">
              <div className="flex items-center gap-2">
                <div className="grid h-8 w-8 place-items-center rounded-lg bg-brand-600 text-sm font-bold text-white">W</div>
                <div>
                  <p className="text-sm font-semibold text-slate-900">WMS</p>
                  <p className="text-[11px] text-slate-500">{t('common.warehouseManagement')}</p>
                </div>
              </div>
              <button
                type="button"
                autoFocus
                onClick={() => setMenuOpen(false)}
                className="grid h-10 w-10 place-items-center rounded-md text-slate-500 hover:bg-slate-100"
                aria-label={t('actions.closeMenu')}
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <nav className="flex-1 space-y-1 overflow-y-auto p-3">
              {items.map((item) => {
                const Icon = icons[item.to] ?? LayoutDashboard
                return (
                  <NavLink key={item.to} to={item.to} className={({ isActive }) => navLinkClass(isActive)}>
                    <Icon className="h-4 w-4" />
                    {t(item.key)}
                    {item.to === '/notifications' && Boolean(unread) && (
                      <span className="ml-auto rounded-full bg-red-100 px-1.5 text-xs font-semibold text-red-700">{unread}</span>
                    )}
                  </NavLink>
                )
              })}
            </nav>
            <div className="space-y-3 border-t border-slate-200 p-3">
              <LanguageSwitcher className="w-full" />
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-slate-800">{user.full_name}</p>
                <p className="truncate text-xs text-slate-500">
                  {t(`role.${user.role}`)}
                  {user.department ? ` · ${user.department.name}` : ''}
                </p>
              </div>
              <button
                type="button"
                onClick={signOut}
                className="flex h-10 w-full items-center gap-2 rounded-md px-3 text-sm font-medium text-slate-600 hover:bg-slate-100"
              >
                <LogOut className="h-4 w-4" />
                {t('common.signOut')}
              </button>
            </div>
          </div>
        </div>
      )}

      <nav
        className="fixed inset-x-0 bottom-0 z-40 border-t border-slate-200 bg-white pb-[env(safe-area-inset-bottom)] md:hidden"
        aria-label={t('nav.mainNavigation')}
      >
        <div className="grid grid-cols-5">
          {primaryItems.map((item) => {
            const Icon = icons[item.to] ?? LayoutDashboard
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    'flex min-h-14 flex-col items-center justify-center gap-1 px-1 text-[11px] font-medium',
                    isActive ? 'text-brand-600' : 'text-slate-500',
                  )
                }
              >
                <Icon className="h-5 w-5" />
                <span className="w-full truncate text-center">{t(item.key)}</span>
              </NavLink>
            )
          })}
          <button
            type="button"
            onClick={() => setMenuOpen(true)}
            className={cn(
              'flex min-h-14 flex-col items-center justify-center gap-1 px-1 text-[11px] font-medium',
              menuOpen ? 'text-brand-600' : 'text-slate-500',
            )}
          >
            <MoreHorizontal className="h-5 w-5" />
            <span className="w-full truncate text-center">{t('common.more')}</span>
          </button>
        </div>
      </nav>
    </div>
  )
}
