import type { Combo, ProviderConnection, ProviderNode } from '../../api/client'
import { getModelCaps, getModelsByProviderId, PROVIDER_ID_TO_ALIAS } from '../../lib/models'
import { PROVIDER_CATALOG } from '../../lib/providers'

export interface PickerModel {
  id: string
  name: string
  value: string
  caps: { vision: boolean; reasoning: boolean }
}

export interface PickerGroup {
  id: string
  name: string
  color?: string
  models: PickerModel[]
}

export function resolveModelPickerGroups(
  connections: ProviderConnection[] = [],
  providerNodes: ProviderNode[] = []
): PickerGroup[] {
  const activeProviderIds = new Set<string>()
  for (const c of connections) {
    if (c.isActive === 1 && c.provider) {
      activeProviderIds.add(c.provider)
    }
  }

  const groups: PickerGroup[] = []
  const seenGroupIds = new Set<string>()

  // 1. Catalog providers (active connections or noAuth)
  for (const catItem of PROVIDER_CATALOG) {
    const isConnected =
      activeProviderIds.has(catItem.id) || (catItem.alias && activeProviderIds.has(catItem.alias))
    const isNoAuth = catItem.noAuth === true
    if (!isConnected && !isNoAuth) {
      continue
    }

    const alias = catItem.alias || PROVIDER_ID_TO_ALIAS[catItem.id] || catItem.id
    const rawModels = getModelsByProviderId(catItem.id)
    if (!rawModels || rawModels.length === 0) {
      continue
    }

    const seenModelIds = new Set<string>()
    const models: PickerModel[] = []

    for (const m of rawModels) {
      if (!m.id || seenModelIds.has(m.id)) continue
      seenModelIds.add(m.id)
      models.push({
        id: m.id,
        name: m.name || m.id,
        value: `${alias}/${m.id}`,
        caps: getModelCaps(m.id, m),
      })
    }

    if (models.length > 0) {
      seenGroupIds.add(catItem.id)
      groups.push({
        id: catItem.id,
        name: catItem.name || catItem.id,
        color: catItem.color,
        models,
      })
    }
  }

  // 2. Compatible nodes in providerNodes
  for (const node of providerNodes) {
    if (!node.id || seenGroupIds.has(node.id)) continue

    const nodeWithModels = node as ProviderNode & {
      models?: Array<string | { id: string; name?: string }>
    }
    const rawModels = nodeWithModels.models || []
    let models: PickerModel[] = []

    if (Array.isArray(rawModels) && rawModels.length > 0) {
      const seen = new Set<string>()
      for (const m of rawModels) {
        const mid = typeof m === 'string' ? m : m.id
        const mname = typeof m === 'string' ? m : m.name || m.id
        if (!mid || seen.has(mid)) continue
        seen.add(mid)
        models.push({
          id: mid,
          name: mname,
          value: `${node.id}/${mid}`,
          caps: getModelCaps(mid),
        })
      }
    } else {
      const catalogModels = getModelsByProviderId(node.id)
      if (catalogModels && catalogModels.length > 0) {
        const seen = new Set<string>()
        for (const m of catalogModels) {
          if (!m.id || seen.has(m.id)) continue
          seen.add(m.id)
          models.push({
            id: m.id,
            name: m.name || m.id,
            value: `${node.id}/${m.id}`,
            caps: getModelCaps(m.id, m),
          })
        }
      }
    }

    if (models.length > 0) {
      seenGroupIds.add(node.id)
      groups.push({
        id: node.id,
        name: node.name || node.id,
        models,
      })
    }
  }

  return groups
}

export function resolveFilteredCombos(
  combos: Combo[] = [],
  currentComboName: string | undefined,
  searchQuery: string,
  target: string
): Combo[] {
  if (target === 'vision' || target === 'audio') {
    return []
  }

  const query = searchQuery.trim().toLowerCase()
  return combos.filter((c) => {
    if (currentComboName && c.name === currentComboName) {
      return false
    }
    if (query) {
      return c.name.toLowerCase().includes(query)
    }
    return true
  })
}

export function resolveFilteredGroups(
  groups: PickerGroup[] = [],
  searchQuery: string,
  target: string
): PickerGroup[] {
  const query = searchQuery.trim().toLowerCase()

  return groups
    .map((group) => {
      let models = group.models

      if (target === 'vision') {
        if (!query) {
          models = models.filter((m) => m.caps.vision)
        }
      }

      if (query) {
        const groupMatches = group.name.toLowerCase().includes(query)
        const matchedModels = models.filter(
          (m) =>
            m.name.toLowerCase().includes(query) ||
            m.id.toLowerCase().includes(query) ||
            m.value.toLowerCase().includes(query)
        )
        models = matchedModels.length > 0 ? matchedModels : groupMatches ? models : []
      }

      if (models.length === 0) return null

      return {
        ...group,
        models,
      }
    })
    .filter((g): g is PickerGroup => g !== null)
}
