// Typed API client for 9router-go Native Dashboard

export interface ProviderConnection {
  id: string
  provider: string
  authType: string
  name: string | null
  email: string | null
  priority: number | null
  isActive: number // 1 or 0
  data: string // JSON string
  createdAt: string
  updatedAt: string
}

export interface Combo {
  id: string
  name: string
  kind: string | null
  models: string // JSON array string
  strategy: string
  createdAt: string
  updatedAt: string
}

export interface APIKey {
  id: string
  key: string
  name: string | null
  machineId: string | null
  isActive: number
  createdAt: string
}

export interface Settings {
  requireApiKey?: boolean
  rtkEnabled?: boolean
  cavemanEnabled?: boolean
  cavemanLevel?: string
  ponytailEnabled?: boolean
  ponytailLevel?: string
  headroomUrl?: string
  headroomKompress?: boolean
  autoUpdate?: boolean
  providerStrategies?: Record<string, { proxyPoolId?: string; rotateStrategy?: string }>
  [key: string]: unknown
}

export interface ProviderNode {
  id: string
  type: string
  name: string
  prefix?: string
  apiType?: string
  baseUrl?: string
  createdAt?: string
  updatedAt?: string
}

export interface FreebuffInitiateResponse {
  loginUrl: string
  authCode: string
  fingerprintId: string
  fingerprintHash: string
  expiresAt: string
}

export interface FreebuffPollResponse {
  status: 'authorized' | 'pending' | 'expired'
  connectionId?: string
  user?: {
    id?: string
    name?: string
    email?: string
  }
}

// Helper to get auth header if stored in localStorage
export function getAuthHeaders(): Record<string, string> {
  const token = localStorage.getItem('9router_key') || 'sk-8b71f86e0a1f2fb5-nhz496-cfa1c800'
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = {
    ...getAuthHeaders(),
    ...(options.headers as Record<string, string> || {}),
  }
  const res = await fetch(path, { ...options, headers })
  if (!res.ok) {
    let errText = ''
    try {
      const errJson = await res.json()
      errText = errJson.error?.message || errJson.error || errJson.message || JSON.stringify(errJson)
    } catch {
      errText = await res.text()
    }
    throw new Error(errText || `Request failed with status ${res.status}`)
  }
  return res.json()
}

// Connections API
export const api = {
  // Connections
  getConnections: () => request<ProviderConnection[]>('/api/connections'),
  createConnection: (payload: { id?: string; provider: string; authType: string; name?: string; apiKey?: string; data?: string }) =>
    request<{ success: boolean; id: string }>('/api/connections', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  updateConnection: (id: string, payload: Partial<ProviderConnection>) =>
    request<{ success: boolean }>(`/api/connections/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),
  deleteConnection: (id: string) =>
    request<{ success: boolean }>(`/api/connections/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),

  // Provider Nodes (Custom Endpoints)
  getProviderNodes: async () => {
    const res = await request<{ nodes: ProviderNode[] }>('/api/provider-nodes')
    return res.nodes || []
  },
  createProviderNode: async (payload: { name: string; prefix: string; apiType?: string; baseUrl?: string; type?: string }) => {
    const res = await request<{ node: ProviderNode }>('/api/provider-nodes', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
    return res.node
  },
  deleteProviderNode: (id: string) =>
    request<{ success: boolean }>(`/api/provider-nodes/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),

  // Combos
  getCombos: () => request<Combo[]>('/api/combos'),
  createCombo: (payload: { id?: string; name: string; kind?: string; models: string; strategy?: string }) =>
    request<{ success: boolean; id: string }>('/api/combos', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  updateCombo: (id: string, payload: Partial<Combo>) =>
    request<{ success: boolean }>(`/api/combos/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    }),
  deleteCombo: (id: string) =>
    request<{ success: boolean }>(`/api/combos/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),

  // API Keys
  getApiKeys: () => request<APIKey[]>('/api/keys'),
  createApiKey: (payload: { name?: string; machineId?: string; key?: string }) =>
    request<{ success: boolean; id: string; key: string }>('/api/keys', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  deleteApiKey: (id: string) =>
    request<{ success: boolean }>(`/api/keys/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
  toggleApiKey: (id: string) =>
    request<{ success: boolean; isActive: boolean }>(`/api/keys/${encodeURIComponent(id)}/toggle`, {
      method: 'PUT',
    }),

  // Models
  getCustomModels: () => request<Record<string, string>>('/api/models/custom'),
  getDisabledModels: () => request<Record<string, string>>('/api/models/disabled'),
  saveCustomModel: (key: string, value: unknown) =>
    request<{ success: boolean }>('/api/models/custom', {
      method: 'POST',
      body: JSON.stringify({ key, value }),
    }),
  deleteCustomModel: (key: string) =>
    request<{ success: boolean }>(`/api/models/custom/${encodeURIComponent(key)}`, {
      method: 'DELETE',
    }),
  saveDisabledModels: (provider: string, modelIds: string[]) =>
    request<{ success: boolean }>(`/api/models/disabled/${encodeURIComponent(provider)}`, {
      method: 'PUT',
      body: JSON.stringify(modelIds),
    }),

  // Settings
  getSettings: () => request<Settings>('/api/settings'),
  updateSettings: (settings: Partial<Settings>) =>
    request<{ success: boolean }>('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
  // OAuth Flows
  initiateFreebuff: () => request<FreebuffInitiateResponse>('/api/oauth/freebuff/initiate', { method: 'POST' }),
  pollFreebuff: (fingerprintId: string, fingerprintHash: string) =>
    request<FreebuffPollResponse>('/api/oauth/freebuff/poll', {
      method: 'POST',
      body: JSON.stringify({ fingerprintId, fingerprintHash }),
    }),
  getAntigravityAuthorizeUrl: () => request<{ url: string; redirectUrl: string; state: string }>('/api/oauth/antigravity/authorize'),

  // Usage & Telemetry
  getUsageStats: () => request<any>('/api/usage/stats'),
  getSystemVersion: () => request<{ version: string }>('/api/version'),
  resetHealth: (provider: string, model?: string) =>
    request<{ status: string }>(`/admin/health/reset?provider=${encodeURIComponent(provider)}${model ? `&model=${encodeURIComponent(model)}` : ''}`, {
      method: 'POST',
    }),
}
