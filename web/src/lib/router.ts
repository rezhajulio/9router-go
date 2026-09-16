export type ActiveTab = 'analytics' | 'combos' | 'connections' | 'settings' | 'keys' | 'terminal' | 'media-web'

export const TAB_ROUTES: Record<ActiveTab, string> = {
  analytics: '/dashboard/usage',
  connections: '/dashboard/providers',
  combos: '/dashboard/combos',
  keys: '/dashboard/keys',
  terminal: '/dashboard/terminal',
  settings: '/dashboard/quota',
  'media-web': '/dashboard/media-providers/web',
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

  // media-web
  '/dashboard/media-providers/web': 'media-web',
  '/dashboard/media-providers': 'media-web',
  '/media-providers/web': 'media-web',
  '/media': 'media-web',
}
export function pathToTab(pathname: string): ActiveTab {
  if (!pathname) return 'connections'
  const clean = pathname.trim().split('?')[0].split('#')[0]
  const normalized = clean.replace(/\/+$/, '') || '/'
  if (normalized.toLowerCase().includes('media')) {
    return 'media-web'
  }
  return ROUTE_TO_TAB[normalized.toLowerCase()] || 'connections'
}
