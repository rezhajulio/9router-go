import type { Combo, ProviderNode } from '../../api/client'

export interface ComboStrategyInfo {
  fallbackStrategy?: string
  judgeModel?: string
}

export interface AdapterPool {
  enabled: boolean
  roundRobin: boolean
  models: string[]
}

export interface CapacityAdapterState {
  vision: AdapterPool
  audioInput: AdapterPool
}

export interface PickerModelItem {
  value: string
  label: string
  provider: string
  vision: boolean
  reasoning: boolean
}

export function hasVision(model: string): boolean {
  const m = model.toLowerCase()
  return (
    m.includes('vision') ||
    m.includes('gemini') ||
    m.includes('claude-3') ||
    m.includes('claude-sonnet') ||
    m.includes('claude-opus') ||
    m.includes('gpt-4o') ||
    m.includes('spark') ||
    m.includes('vl') ||
    m.includes('flash')
  )
}

export function hasReasoning(model: string): boolean {
  const m = model.toLowerCase()
  return (
    m.includes('reason') ||
    m.includes('think') ||
    m.includes('r1') ||
    m.includes('deepseek') ||
    m.includes('spark') ||
    m.includes('high') ||
    m.includes('o1') ||
    m.includes('o3') ||
    m.includes('pro-agent')
  )
}

export function getComboModels(c: Combo): string[] {
  if (Array.isArray(c.models)) return c.models
  if (typeof c.models === 'string') {
    try {
      const parsed = JSON.parse(c.models)
      if (Array.isArray(parsed)) return parsed
      return [c.models]
    } catch {
      return c.models ? [c.models] : []
    }
  }
  return []
}

export function computeAvailableModels(
  providerNodes: ProviderNode[],
  combos: Combo[]
): PickerModelItem[] {
  const list: PickerModelItem[] = []
  const seen = new Set<string>()

  for (const node of providerNodes) {
    const p = node.id
    const nodeWithModels = node as ProviderNode & { models?: string[] }
    if (nodeWithModels.models && Array.isArray(nodeWithModels.models)) {
      for (const m of nodeWithModels.models) {
        const val = `${p}/${m}`
        if (!seen.has(val)) {
          seen.add(val)
          list.push({
            value: val,
            label: m,
            provider: node.name || p,
            vision: hasVision(m),
            reasoning: hasReasoning(m),
          })
        }
      }
    }
  }

  for (const c of combos) {
    const cms = getComboModels(c)
    for (const m of cms) {
      if (!seen.has(m)) {
        seen.add(m)
        const prov = m.includes('/') ? m.split('/')[0] : 'combo'
        list.push({
          value: m,
          label: m.includes('/') ? m.split('/')[1] : m,
          provider: prov,
          vision: hasVision(m),
          reasoning: hasReasoning(m),
        })
      }
    }
  }
  return list
}

export function updateComboStrategy(
  currentStrategies: Record<string, ComboStrategyInfo>,
  comboName: string,
  newStrategy: string
): Record<string, ComboStrategyInfo> {
  const updated = { ...currentStrategies }
  const current = updated[comboName] || {}
  const next = { ...current, fallbackStrategy: newStrategy }

  if (newStrategy === 'fallback' && !next.judgeModel) {
    delete updated[comboName]
  } else {
    updated[comboName] = next
  }
  return updated
}

export function updateJudgeModel(
  currentStrategies: Record<string, ComboStrategyInfo>,
  comboName: string,
  judgeModel: string
): Record<string, ComboStrategyInfo> {
  const updated = { ...currentStrategies }
  const current = updated[comboName] || {}
  updated[comboName] = { ...current, judgeModel }
  return updated
}

export function clearJudgeModel(
  currentStrategies: Record<string, ComboStrategyInfo>,
  comboName: string
): Record<string, ComboStrategyInfo> {
  const updated = { ...currentStrategies }
  if (updated[comboName]) {
    const { judgeModel: _, ...rest } = updated[comboName]
    if (!rest.fallbackStrategy || rest.fallbackStrategy === 'fallback') {
      delete updated[comboName]
    } else {
      updated[comboName] = rest
    }
  }
  return updated
}

export function parseCapacityAdapterSettings(ca: Record<string, unknown> | undefined): CapacityAdapterState {
  const v = ca?.vision as Record<string, unknown> | undefined
  const a = ca?.audioInput as Record<string, unknown> | undefined
  return {
    vision: {
      enabled: v?.enabled !== false,
      roundRobin: !!v?.roundRobin,
      models: Array.isArray(v?.models) ? (v.models as string[]) : ['ag/gemini-3.7-flash-high'],
    },
    audioInput: {
      enabled: a?.enabled !== false,
      roundRobin: !!a?.roundRobin,
      models: Array.isArray(a?.models) ? (a.models as string[]) : [],
    },
  }
}
