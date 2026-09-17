import { api, type ProviderConnection } from '../../api/client'
import { getModelCaps, getModelKind } from '../../lib/models'

export const MEDIA_KINDS: Record<string, true> = {
  image: true,
  tts: true,
  stt: true,
  embedding: true,
  video: true,
}

export function isChatModel(m: unknown): boolean {
  const kind = getModelKind(m)
  if (!kind || kind === 'llm') {
    const obj = typeof m === 'object' && m !== null ? (m as { kind?: string; type?: string }) : null
    return !(obj?.kind && MEDIA_KINDS[obj.kind]) && !(obj?.type && MEDIA_KINDS[obj.type])
  }
  return false
}

export interface ProviderStats {
  total: number
  connected: number
  errorCount: number
  allDisabled: boolean
  latestError?: string | null
  connections: ProviderConnection[]
}

export interface ModelItem {
  id: string
  name?: string
  isCustom?: boolean
  caps: { vision: boolean; reasoning: boolean }
  kind?: string
}

export interface CustomModelData {
  id: string
  name?: string
  providerAlias?: string
  type?: string
}

export function getIconPath(id: string, apiType?: string): string {
  if (id.startsWith('openai-compatible')) {
    return apiType === 'responses' ? '/providers/oai-r.png' : '/providers/oai-cc.png'
  }
  if (id.startsWith('anthropic-compatible')) {
    return '/providers/anthropic-m.png'
  }
  return `/providers/${id}.png`
}

export function getProviderStats(
  connections: ProviderConnection[],
  providerId: string,
  authTypes?: string[]
): ProviderStats {
  const list = connections.filter((c) => {
    if (c.provider !== providerId) return false
    if (authTypes && authTypes.length > 0) {
      return authTypes.includes(c.authType)
    }
    return true
  })

  const total = list.length
  const connected = list.filter((c) => c.isActive === 1 && c.testStatus !== 'error' && !c.lastError).length
  const errorCount = list.filter((c) => c.testStatus === 'error' || !!c.lastError).length
  const allDisabled = total > 0 && list.every((c) => c.isActive === 0)
  const latestError = list.find((c) => !!c.lastError)?.lastError

  return { total, connected, errorCount, allDisabled, latestError, connections: list }
}

export function matchesFilter(stats: ProviderStats, statusFilter: string, noAuth?: boolean): boolean {
  if (statusFilter === 'all') return true
  if (statusFilter === 'connected') return stats.connected > 0 || (!!noAuth && stats.total === 0)
  if (statusFilter === 'error') return stats.errorCount > 0
  if (statusFilter === 'disabled') return stats.allDisabled
  if (statusFilter === 'not_connected') return stats.total === 0 && !noAuth
  return true
}

export function matchesSearch(name: string, searchQuery: string): boolean {
  if (!searchQuery.trim()) return true
  return name.toLowerCase().includes(searchQuery.trim().toLowerCase())
}

export async function fetchProviderModelsData(
  providerId: string,
  storageAlias: string
): Promise<{ customModels: CustomModelData[]; disabledModelIds: string[] }> {
  try {
    const [customs, disabled] = await Promise.all([
      api.getCustomModels().catch(() => ({})),
      api.getDisabledModels().catch(() => ({})),
    ])
    let customModels: CustomModelData[] = []
    if (Array.isArray(customs)) {
      customModels = customs
    } else if (customs && typeof customs === 'object') {
      customModels = Object.entries(customs).map(([k, v]: [string, any]) =>
        typeof v === 'object' && v !== null ? { id: k, ...v } : { id: k, name: String(v) }
      )
    }
    const disArr = (disabled && (disabled as any)[storageAlias]) || (disabled && (disabled as any)[providerId]) || []
    return {
      customModels,
      disabledModelIds: Array.isArray(disArr) ? disArr : [],
    }
  } catch {
    return { customModels: [], disabledModelIds: [] }
  }
}

export function buildAvailableModels(
  builtInModels: Array<{ id: string; name?: string; kind?: string; type?: string }>,
  providerCustomModels: CustomModelData[]
): ModelItem[] {
  const list: ModelItem[] = []
  const seen = new Set<string>()

  for (const cm of providerCustomModels) {
    if (!cm.id || seen.has(cm.id)) continue
    seen.add(cm.id)
    list.push({
      id: cm.id,
      name: cm.name || cm.id,
      isCustom: true,
      caps: getModelCaps(cm.id, cm),
      kind: getModelKind(cm),
    })
  }
  for (const bm of builtInModels) {
    if (!bm.id || seen.has(bm.id)) continue
    seen.add(bm.id)
    list.push({
      id: bm.id,
      name: bm.name,
      isCustom: false,
      caps: getModelCaps(bm.id, bm),
      kind: getModelKind(bm),
    })
  }
  return list
}
