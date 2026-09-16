export type ActiveTab = 'analytics' | 'combos' | 'connections' | 'settings' | 'keys' | 'terminal'

export const TAB_ROUTES: Record<ActiveTab, string> = {
  analytics: '/dashboard/usage',
  connections: '/dashboard/providers',
  combos: '/dashboard/combos',
  keys: '/dashboard/keys',
  terminal: '/dashboard/terminal',
  settings: '/dashboard/quota',
}

const ROUTE_TO_TAB: Record<string, ActiveTab> = {
  // analytics
  '/dashboard/usage': 'analytics',
  '/dashboard/analytics': 'analytics',
  '/usage': 'analytics',
  '/analytics': 'analytics',

  // connections
  '/dashboard/providers': 'connections',
  '/dashboard/connections': 'connections',
  '/providers': 'connections',
  '/connections': 'connections',

  // combos
  '/dashboard/combos': 'combos',
  '/combos': 'combos',

  // keys
  '/dashboard/keys': 'keys',
  '/keys': 'keys',

  // terminal
  '/dashboard/terminal': 'terminal',
  '/dashboard/logs': 'terminal',
  '/terminal': 'terminal',

  // settings
  '/dashboard/quota': 'settings',
  '/dashboard/settings': 'settings',
  '/quota': 'settings',
  '/settings': 'settings',
}

export function pathToTab(pathname: string): ActiveTab {
  if (!pathname) return 'connections'
  const clean = pathname.trim().split('?')[0].split('#')[0]
  const normalized = clean.replace(/\/+$/, '') || '/'
  return ROUTE_TO_TAB[normalized.toLowerCase()] || 'connections'
}
