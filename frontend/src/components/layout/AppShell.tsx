import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Bell, Boxes, ClipboardList, LayoutDashboard, LogOut, Package, RotateCcw, ScrollText, SlidersHorizontal, Truck, Users } from 'lucide-react'
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
  '/users': Users,
  '/notifications': Bell,
}

export function AppShell() {
  const { user, logout } = useAuth()
  const { t } = useT()
  const navigate = useNavigate()
  const { data: unread } = useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: async () => (await notificationApi.unreadCount()).data.unread_count,
    enabled: Boolean(user),
    refetchInterval: 30_000,
  })
  if (!user) return null
  const items = navItems.filter((item) => item.roles.includes(user.role as Role))
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
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive ? 'bg-brand-50 text-brand-700' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900',
                  )
                }
              >
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
        <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white px-4 md:px-6">
          <div className="flex items-center gap-3 md:hidden">
            <div className="grid h-8 w-8 place-items-center rounded-lg bg-brand-600 text-sm font-bold text-white">W</div>
            <span className="font-semibold">WMS</span>
          </div>
          <div className="hidden text-sm text-slate-500 md:block">
            {t('common.signedInAs', { email: user.email })}
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => navigate('/notifications')}
              className="relative rounded-md p-2 text-slate-500 hover:bg-slate-100"
              aria-label={t('nav.notifications')}
            >
              <Bell className="h-5 w-5" />
              {Boolean(unread) && <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-red-500" />}
            </button>
            <LanguageSwitcher />
            <div className="text-right">
              <p className="text-sm font-medium text-slate-800">{user.full_name}</p>
              <p className="text-xs text-slate-500">
                {t(`role.${user.role}`)}
                {user.department ? ` · ${user.department.name}` : ''}
              </p>
            </div>
            <button
              onClick={() => { logout(); navigate('/login') }}
              className="rounded-md p-2 text-slate-500 hover:bg-slate-100"
              aria-label={t('common.signOut')}
            >
              <LogOut className="h-5 w-5" />
            </button>
          </div>
        </header>
        <main className="flex-1 p-4 md:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
