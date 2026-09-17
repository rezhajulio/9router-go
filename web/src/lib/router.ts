export type ActiveTab =
  | 'analytics'
  | 'combos'
  | 'connections'
  | 'settings'
  | 'keys'
  | 'terminal'
  | 'media-embedding'
  | 'media-image'
  | 'media-tts'
  | 'media-stt'
  | 'media-video'
  | 'media-web'

export const TAB_ROUTES: Record<ActiveTab, string> = {
  analytics: '/dashboard/usage',
  connections: '/dashboard/providers',
  combos: '/dashboard/combos',
  keys: '/dashboard/keys',
  terminal: '/dashboard/terminal',
  settings: '/dashboard/quota',
  'media-embedding': '/dashboard/media-providers/embedding',
  'media-image': '/dashboard/media-providers/image',
  'media-tts': '/dashboard/media-providers/tts',
  'media-stt': '/dashboard/media-providers/stt',
  'media-video': '/dashboard/media-providers/video',
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

  // media-embedding
  '/dashboard/media-providers/embedding': 'media-embedding',
  '/media-providers/embedding': 'media-embedding',
  '/media/embedding': 'media-embedding',

  // media-image
  '/dashboard/media-providers/image': 'media-image',
  '/media-providers/image': 'media-image',
  '/media/image': 'media-image',

  // media-tts
  '/dashboard/media-providers/tts': 'media-tts',
  '/media-providers/tts': 'media-tts',
  '/media/tts': 'media-tts',

  // media-stt
  '/dashboard/media-providers/stt': 'media-stt',
  '/media-providers/stt': 'media-stt',
  '/media/stt': 'media-stt',

  // media-video
  '/dashboard/media-providers/video': 'media-video',
  '/media-providers/video': 'media-video',
  '/media/video': 'media-video',

  // media-web
  '/dashboard/media-providers/web': 'media-web',
  '/dashboard/media-providers': 'media-web',
  '/media-providers/web': 'media-web',
  '/media/web': 'media-web',
  '/media': 'media-web',
}
export function pathToTab(pathname: string): ActiveTab {
  if (!pathname) return 'connections'
  const clean = pathname.trim().split('?')[0].split('#')[0]
  const normalized = (clean.replace(/\/+$/, '') || '/').toLowerCase()
  if (ROUTE_TO_TAB[normalized]) {
    return ROUTE_TO_TAB[normalized]
  }
  if (normalized.includes('embedding')) return 'media-embedding'
  if (normalized.includes('image')) return 'media-image'
  if (normalized.includes('tts')) return 'media-tts'
  if (normalized.includes('stt')) return 'media-stt'
  if (normalized.includes('video')) return 'media-video'
  if (normalized.includes('media')) return 'media-web'
  return 'connections'
}
