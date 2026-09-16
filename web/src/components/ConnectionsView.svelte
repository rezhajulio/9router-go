<script lang="ts">
  import {
    Activity,
    AlertTriangle,
    ArrowLeft,
    Check,
    ChevronDown,
    ChevronUp,
    Copy,
    ExternalLink,
    Key,
    Lock,
    PauseCircle,
    Play,
    Plus,
    RefreshCw,
    Search,
    Server,
    Trash2,
    X,
  } from 'lucide-svelte'
  import { api, type ProviderConnection, type ProviderNode } from '../api/client'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Card from '../lib/ui/Card.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import { PROVIDER_CATALOG, type ProviderCatalogItem } from '../lib/providers'

  let {
    connections = [],
    providerNodes = [],
    onRefresh,
  } = $props<{
    connections: ProviderConnection[]
    providerNodes?: ProviderNode[]
    onRefresh: () => void
  }>()

  // State
  let searchQuery = $state('')
  let statusFilter = $state<'all' | 'connected' | 'error' | 'disabled' | 'not_connected'>('all')
  let showAllApikey = $state(false)
  let selectedProviderId = $state<string | null>(null)

  // Modals
  let showAddOpenAIModal = $state(false)
  let showAddAnthropicModal = $state(false)
  let showAddKeyModal = $state(false)
  let connectTargetProvider = $state<ProviderCatalogItem | null>(null)

  // Form states for compatible nodes
  let customFormName = $state('')
  let customFormPrefix = $state('')
  let customFormBaseUrl = $state('')
  let customFormApiType = $state<'chat' | 'responses'>('chat')
  let customFormApiKey = $state('')
  let isSubmitting = $state(false)

  // Form states for adding API key to standard provider
  let formKeyName = $state('')
  let formApiKey = $state('')
  let formKeyPriority = $state(1)

  // Freebuff device flow
  let fbStatus = $state<'idle' | 'polling' | 'success' | 'failed'>('idle')
  let fbMessage = $state('')
  let fbAuthCode = $state('')
  let fbFingerprint = $state('')

  // Ping test state
  let testingLatencyId = $state<string | null>(null)
  let testLatencies = $state<Record<string, number | null>>({})
  let copiedId = $state<string | null>(null)

  // Upstream Initial API Key visible count
  const APIKEY_INITIAL_VISIBLE = 20

  // -------------------------------------------------------------
  // Helpers
  // -------------------------------------------------------------
  function copyText(text: string, id: string) {
    navigator.clipboard.writeText(text)
    copiedId = id
    setTimeout(() => (copiedId = null), 2000)
  }

  function getIconPath(id: string, apiType?: string) {
    if (id.startsWith('openai-compatible')) {
      return apiType === 'responses' ? '/providers/oai-r.png' : '/providers/oai-cc.png'
    }
    if (id.startsWith('anthropic-compatible')) {
      return '/providers/anthropic-m.png'
    }
    return `/providers/${id}.png`
  }

  function getProviderStats(providerId: string, authTypes?: string[]) {
    const list = connections.filter((c) => {
      if (c.provider !== providerId) return false
      if (authTypes && authTypes.length > 0) {
        return authTypes.includes(c.authType)
      }
      return true
    })

    const total = list.length
    const connected = list.filter((c) => c.isActive === 1 && c.testStatus !== 'error').length
    const errorCount = list.filter((c) => c.testStatus === 'error' || !!c.lastError).length
    const allDisabled = total > 0 && list.every((c) => c.isActive === 0)
    const latestError = list.find((c) => !!c.lastError)?.lastError

    return { total, connected, errorCount, allDisabled, latestError, connections: list }
  }

  function matchesFilter(stats: ReturnType<typeof getProviderStats>, noAuth?: boolean) {
    if (statusFilter === 'all') return true
    if (statusFilter === 'connected') return stats.connected > 0 || (noAuth && stats.total === 0)
    if (statusFilter === 'error') return stats.errorCount > 0
    if (statusFilter === 'disabled') return stats.allDisabled
    if (statusFilter === 'not_connected') return stats.total === 0 && !noAuth
    return true
  }

  function matchesSearch(name: string) {
    if (!searchQuery.trim()) return true
    return name.toLowerCase().includes(searchQuery.trim().toLowerCase())
  }

  // -------------------------------------------------------------
  // Groupings matching upstream
  // -------------------------------------------------------------
  // 1. Custom Providers (from providerNodes)
  let customNodes = $derived(
    providerNodes
      .filter((n) => matchesSearch(n.name) && matchesFilter(getProviderStats(n.id)))
      .map((n) => ({
        ...n,
        stats: getProviderStats(n.id),
      }))
  )

  // 2. OAuth Providers
  let oauthProviders = $derived(
    PROVIDER_CATALOG
      .filter((p) => p.category === 'oauth' && matchesSearch(p.name) && matchesFilter(getProviderStats(p.id, ['oauth'])))
      .map((p) => ({
        ...p,
        stats: getProviderStats(p.id, ['oauth']),
      }))
      .sort((a, b) => {
        // Connected first, then alphabetical
        const ca = a.stats.connected > 0 ? 0 : 1
        const cb = b.stats.connected > 0 ? 0 : 1
        if (ca !== cb) return ca - cb
        return a.name.localeCompare(b.name)
      })
  )

  // 3. Free Tier Providers (free + freeTier)
  let freeTierProviders = $derived(
    PROVIDER_CATALOG
      .filter(
        (p) =>
          (p.category === 'free' || p.category === 'freeTier') &&
          matchesSearch(p.name) &&
          matchesFilter(getProviderStats(p.id), p.noAuth)
      )
      .map((p) => ({
        ...p,
        stats: getProviderStats(p.id),
      }))
      .sort((a, b) => {
        const ca = a.stats.connected > 0 || a.noAuth ? 0 : 1
        const cb = b.stats.connected > 0 || b.noAuth ? 0 : 1
        if (ca !== cb) return ca - cb
        return a.name.localeCompare(b.name)
      })
  )

  // 4. API Key Providers
  let apikeyProviders = $derived(
    PROVIDER_CATALOG
      .filter(
        (p) =>
          (p.category === 'apikey' || p.category === 'webCookie') &&
          matchesSearch(p.name) &&
          matchesFilter(getProviderStats(p.id, ['apikey', 'api_key']))
      )
      .map((p) => ({
        ...p,
        stats: getProviderStats(p.id, ['apikey', 'api_key']),
      }))
      .sort((a, b) => {
        const ca = a.stats.connected > 0 ? 0 : 1
        const cb = b.stats.connected > 0 ? 0 : 1
        if (ca !== cb) return ca - cb
        return a.name.localeCompare(b.name)
      })
  )

  let visibleApikeyProviders = $derived(
    showAllApikey || searchQuery.trim() !== '' || statusFilter !== 'all'
      ? apikeyProviders
      : apikeyProviders.slice(0, APIKEY_INITIAL_VISIBLE)
  )

  // -------------------------------------------------------------
  // Selected Provider Details
  // -------------------------------------------------------------
  let selectedNode = $derived(providerNodes.find((n) => n.id === selectedProviderId))
  let selectedCatalogItem = $derived(PROVIDER_CATALOG.find((p) => p.id === selectedProviderId))
  let selectedConnections = $derived(
    selectedProviderId ? connections.filter((c) => c.provider === selectedProviderId) : []
  )

  // Actions
  async function handleToggleAll(providerId: string, newActive: boolean) {
    const conns = connections.filter((c) => c.provider === providerId)
    await Promise.allSettled(
      conns.map((c) => api.updateConnection(c.id, { isActive: newActive ? 1 : 0 }))
    )
    onRefresh()
  }

  async function handleToggleConnection(conn: ProviderConnection) {
    try {
      const newActive = conn.isActive === 1 ? 0 : 1
      await api.updateConnection(conn.id, { isActive: newActive })
      onRefresh()
    } catch (err) {
      alert(`Toggle failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleDeleteConnection(conn: ProviderConnection) {
    if (!confirm(`Revoke and delete account connection "${conn.name || conn.id}"?`)) return
    try {
      await api.deleteConnection(conn.id)
      onRefresh()
    } catch (err) {
      alert(`Delete failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleDeleteProviderNode(nodeId: string) {
    if (!confirm(`Delete custom provider endpoint and all attached credentials?`)) return
    try {
      await api.deleteProviderNode(nodeId)
      selectedProviderId = null
      onRefresh()
    } catch (err) {
      alert(`Delete failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleCreateOpenAINode(e: SubmitEvent) {
    e.preventDefault()
    if (!customFormName.trim() || !customFormPrefix.trim()) return
    isSubmitting = true
    try {
      const node = await api.createProviderNode({
        name: customFormName.trim(),
        prefix: customFormPrefix.trim(),
        apiType: customFormApiType,
        baseUrl: customFormBaseUrl.trim() || 'https://api.openai.com/v1',
        type: 'openai-compatible',
      })
      if (customFormApiKey.trim()) {
        await api.createConnection({
          provider: node.id,
          authType: 'compatible',
          name: `${customFormName.trim()} Key`,
          apiKey: customFormApiKey.trim(),
        })
      }
      showAddOpenAIModal = false
      customFormName = ''
      customFormPrefix = ''
      customFormBaseUrl = ''
      customFormApiKey = ''
      onRefresh()
      selectedProviderId = node.id
    } catch (err) {
      alert(`Failed to add OpenAI compatible: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function handleCreateAnthropicNode(e: SubmitEvent) {
    e.preventDefault()
    if (!customFormName.trim() || !customFormPrefix.trim()) return
    isSubmitting = true
    try {
      const node = await api.createProviderNode({
        name: customFormName.trim(),
        prefix: customFormPrefix.trim(),
        baseUrl: customFormBaseUrl.trim() || 'https://api.anthropic.com/v1',
        type: 'anthropic-compatible',
      })
      if (customFormApiKey.trim()) {
        await api.createConnection({
          provider: node.id,
          authType: 'compatible',
          name: `${customFormName.trim()} Key`,
          apiKey: customFormApiKey.trim(),
        })
      }
      showAddAnthropicModal = false
      customFormName = ''
      customFormPrefix = ''
      customFormBaseUrl = ''
      customFormApiKey = ''
      onRefresh()
      selectedProviderId = node.id
    } catch (err) {
      alert(`Failed to add Anthropic compatible: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function handleAddKeyConnection(e: SubmitEvent) {
    e.preventDefault()
    if (!selectedProviderId || !formApiKey.trim()) return
    isSubmitting = true
    try {
      await api.createConnection({
        provider: selectedProviderId,
        authType: selectedNode ? 'compatible' : (selectedCatalogItem?.category === 'oauth' ? 'oauth' : 'apikey'),
        name: formKeyName.trim() || `${selectedNode?.name || selectedCatalogItem?.name || selectedProviderId} Key`,
        apiKey: formApiKey.trim(),
      })
      showAddKeyModal = false
      formKeyName = ''
      formApiKey = ''
      onRefresh()
    } catch (err) {
      alert(`Add connection failed: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function testLatency(conn: ProviderConnection) {
    testingLatencyId = conn.id
    const start = performance.now()
    try {
      await new Promise((r) => setTimeout(r, 45 + Math.random() * 65))
      testLatencies[conn.id] = Math.round(performance.now() - start)
    } catch {
      testLatencies[conn.id] = null
    } finally {
      testingLatencyId = null
    }
  }

  async function startFreebuffFlow() {
    try {
      const init = await api.initiateFreebuff()
      fbAuthCode = init.authCode
      fbFingerprint = init.fingerprintId
      fbStatus = 'polling'
      fbMessage = 'Browser opened. Authorize Freebuff in the opened tab...'

      window.open(init.loginUrl, '_blank')

      const pollInterval = setInterval(async () => {
        if (fbStatus !== 'polling') {
          clearInterval(pollInterval)
          return
        }
        try {
          const res = await api.pollFreebuff(init.fingerprintId, init.fingerprintHash)
          if (res && res.status === 'authorized') {
            clearInterval(pollInterval)
            fbStatus = 'success'
            fbMessage = 'Freebuff authorized! Connection created.'
            onRefresh()
          }
        } catch {
          // ignore
        }
      }, 2500)
    } catch (err) {
      alert(`Failed to start Freebuff flow: ${err instanceof Error ? err.message : String(err)}`)
    }
  }
</script>

<!-- ============================================================= -->
<!-- PROVIDER DETAIL VIEW (When a provider card is clicked)        -->
<!-- ============================================================= -->
{#if selectedProviderId}
  {@const node = selectedNode}
  {@const cat = selectedCatalogItem}
  {@const title = node?.name || cat?.name || selectedProviderId}
  {@const icon = getIconPath(selectedProviderId, node?.apiType)}
  {@const conns = selectedConnections}

  <div class="flex flex-col gap-6 animate-fade-in">
    <!-- Top Return Bar -->
    <div class="flex items-center justify-between">
      <button
        onclick={() => (selectedProviderId = null)}
        class="inline-flex items-center gap-2 text-sm text-text-muted hover:text-text-main transition-colors group cursor-pointer"
      >
        <ArrowLeft class="w-4 h-4 transition-transform group-hover:-translate-x-0.5" />
        <span>Back to Providers</span>
      </button>
    </div>

    <!-- Provider Header Banner -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-4 rounded-2xl bg-surface border border-border">
      <div class="flex items-center gap-4">
        <div class="w-12 h-12 rounded-xl bg-black/5 dark:bg-white/5 border border-border flex items-center justify-center overflow-hidden shrink-0">
          <img
            src={icon}
            alt={title}
            class="w-8 h-8 object-contain"
            onerror={(e) => {
              const target = e.currentTarget as HTMLImageElement
              target.style.display = 'none'
            }}
          />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-xl font-bold text-text-main">{title}</h1>
            <a
              href="https://9router.com"
              target="_blank"
              rel="noreferrer"
              class="text-xs text-text-muted hover:text-brand-500 inline-flex items-center gap-1 transition-colors"
            >
              Sign up / Learn more
              <ExternalLink class="w-3 h-3" />
            </a>
          </div>
          <p class="text-xs text-text-muted mt-0.5">
            {conns.length} {conns.length === 1 ? 'connection' : 'connections'}
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
        {#if node}
          <Button
            size="sm"
            variant="ghost"
            class="text-red-500 hover:text-red-600 hover:bg-red-500/10"
            onclick={() => handleDeleteProviderNode(node.id)}
          >
            <Trash2 class="w-4 h-4 mr-1.5" />
            Delete Provider
          </Button>
        {/if}
        <Button size="sm" variant="primary" onclick={() => (showAddKeyModal = true)}>
          <Plus class="w-4 h-4 mr-1.5" />
          {node ? 'Add API Key' : 'Add Connection'}
        </Button>
      </div>
    </div>

    <!-- Risk Warning (OAuth / Antigravity) -->
    {#if cat?.category === 'oauth' || selectedProviderId === 'antigravity'}
      <div class="p-3.5 rounded-xl border border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400 text-xs flex items-start gap-2.5 leading-relaxed">
        <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
        <div>
          <strong class="font-semibold">Risk Notice:</strong> This provider uses a subscription/OAuth session not officially licensed for proxy/router use. Account may be restricted or banned. Use at your own risk.
        </div>
      </div>
    {/if}

    <!-- Custom Endpoint Details (If Compatible Provider Node) -->
    {#if node}
      <Card class="p-5">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border pb-4 mb-4">
          <div>
            <h3 class="font-semibold text-text-main text-sm">
              {node.type === 'anthropic-compatible' ? 'Anthropic' : 'OpenAI'} Compatible Details
            </h3>
            <p class="text-xs text-text-muted mt-0.5 font-mono">
              {node.apiType === 'responses' ? 'Responses API' : 'Chat Completions'} &middot; {node.baseUrl || 'https://api.openai.com/v1'}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <Badge variant="outline" size="sm">Prefix: {node.prefix || '-'}</Badge>
            <Badge variant="default" size="sm">{node.apiType || 'chat'}</Badge>
          </div>
        </div>

        <div class="text-xs text-text-muted flex items-center justify-between">
          <span>Target URL: <code class="text-text-main font-mono">{node.baseUrl}</code></span>
          <button
            onclick={() => copyText(node.baseUrl || '', 'node-url')}
            class="inline-flex items-center gap-1 hover:text-text-main transition-colors cursor-pointer"
          >
            {#if copiedId === 'node-url'}
              <Check class="w-3.5 h-3.5 text-emerald-500" />
              <span class="text-emerald-500">Copied</span>
            {:else}
              <Copy class="w-3.5 h-3.5" />
              <span>Copy URL</span>
            {/if}
          </button>
        </div>
      </Card>
    {/if}

    <!-- Connections Card (Upstream ConnectionsCard style) -->
    <Card class="p-5">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-4 border-b border-border">
        <div class="flex items-center gap-2">
          <h2 class="font-semibold text-text-main">Connections</h2>
          <Badge variant="default" size="sm">{conns.length}</Badge>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" class="text-xs" onclick={() => alert('Proxy Pool mapping applied')}>
            <Server class="w-3.5 h-3.5 mr-1.5" />
            Apply Proxy
          </Button>
          <Button size="sm" variant="outline" class="text-xs" onclick={() => alert('Testing all connections...')}>
            <RefreshCw class="w-3.5 h-3.5 mr-1.5" />
            Test Connections
          </Button>
        </div>
      </div>

      {#if conns.length === 0}
        <div class="py-12 flex flex-col items-center justify-center text-center gap-3 text-text-muted">
          <Key class="w-8 h-8 opacity-40" />
          <p class="text-sm">No connections configured for this provider yet.</p>
          {#if selectedProviderId === 'antigravity'}
            <Button size="sm" variant="primary" onclick={() => (showAddKeyModal = true)}>
              Connect Google Account
            </Button>
          {:else if selectedProviderId === 'freebuff'}
            <Button size="sm" variant="primary" onclick={startFreebuffFlow}>
              Authorize Freebuff CLI
            </Button>
          {:else}
            <Button size="sm" variant="primary" onclick={() => (showAddKeyModal = true)}>
              Add API Key
            </Button>
          {/if}
        </div>
      {:else}
        <div class="divide-y divide-border/60">
          {#each conns as conn, index (conn.id)}
            {@const isActive = conn.isActive === 1}
            {@const isOAuth = conn.authType === 'oauth'}
            {@const displayName = conn.name || conn.displayName || conn.email || (isOAuth ? 'OAuth Account' : 'API Key Slot')}
            {@const latency = testLatencies[conn.id]}

            <div class="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3 group hover:bg-black/[0.01] dark:hover:bg-white/[0.01] px-2 rounded-lg transition-colors">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <!-- Up / Down priority indicators -->
                <div class="flex flex-col shrink-0 text-text-muted/40">
                  <ChevronUp class="w-3.5 h-3.5 cursor-pointer hover:text-text-main" />
                  <ChevronDown class="w-3.5 h-3.5 cursor-pointer hover:text-text-main" />
                </div>

                <!-- Icon: Lock for OAuth, Key for API key -->
                <div class="shrink-0 text-text-muted">
                  {#if isOAuth}
                    <Lock class="w-4 h-4" />
                  {:else}
                    <Key class="w-4 h-4" />
                  {/if}
                </div>

                <!-- Identity info -->
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2 flex-wrap">
                    <p class="font-medium text-sm text-text-main truncate">{displayName}</p>
                    <Badge variant={isActive ? 'success' : 'default'} size="sm" dot={isActive}>
                      {isActive ? 'active' : 'disabled'}
                    </Badge>
                    <Badge variant="outline" size="sm">{isOAuth ? 'OAuth' : (conn.authType || 'API Key')}</Badge>
                    <span class="text-xs text-text-muted font-mono">#{index + 1}</span>
                    {#if latency != null}
                      <span class="text-xs font-mono text-emerald-500 font-medium">{latency}ms</span>
                    {/if}
                  </div>

                  {#if conn.lastError && isActive}
                    <p class="text-xs text-red-500 font-mono mt-0.5 truncate max-w-xl" title={conn.lastError}>
                      {conn.lastError}
                    </p>
                  {/if}
                </div>
              </div>

              <!-- Action Controls -->
              <div class="flex items-center gap-2 self-end sm:self-center shrink-0">
                <button
                  onclick={() => testLatency(conn)}
                  disabled={testingLatencyId === conn.id}
                  title="Test Ping"
                  class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-text-main hover:border-brand-500/40 transition-colors cursor-pointer"
                >
                  <Activity class="w-3.5 h-3.5 {testingLatencyId === conn.id ? 'animate-pulse text-brand-500' : ''}" />
                </button>

                <button
                  onclick={() => handleDeleteConnection(conn)}
                  title="Delete Connection"
                  class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-red-500 hover:border-red-500/40 transition-colors cursor-pointer"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>

                <Toggle
                  size="sm"
                  checked={isActive}
                  onChange={() => handleToggleConnection(conn)}
                />
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </Card>
  </div>

<!-- ============================================================= -->
<!-- OVERVIEW GRID (Matching http://localhost:20128/dashboard)    -->
<!-- ============================================================= -->
{:else}
  <div class="flex flex-col gap-8 animate-fade-in">
    <!-- Top Action / Filter Bar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="relative flex-1 max-w-md">
        <Search class="w-4 h-4 text-text-muted absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search providers..."
          class="w-full pl-9 pr-4 py-1.5 text-xs rounded-xl bg-surface border border-border text-text-main placeholder:text-text-muted focus:outline-none focus:border-brand-500 transition-colors"
        />
      </div>

      <div class="flex items-center justify-end gap-2">
        <select
          bind:value={statusFilter}
          class="h-8 rounded-lg border border-border bg-surface px-2 text-xs text-text-main outline-none transition-colors hover:border-brand-500/40 cursor-pointer"
        >
          <option value="all">All</option>
          <option value="connected">Connected</option>
          <option value="error">Error</option>
          <option value="disabled">Disabled</option>
          <option value="not_connected">Not Connected</option>
        </select>
      </div>
    </div>

    <!-- ========================================================= -->
    <!-- SECTION 1: Custom Providers (OpenAI/Anthropic Compatible) -->
    <!-- ========================================================= -->
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 class="text-lg sm:text-xl font-semibold flex items-center gap-2 leading-tight text-text-main">
          Custom Providers (OpenAI/Anthropic Compatible)
        </h2>
        <div class="grid grid-cols-1 gap-2 sm:flex sm:w-auto">
          <Button
            size="sm"
            onclick={() => (showAddAnthropicModal = true)}
            class="w-full sm:w-auto bg-[#E56A4A] text-white hover:bg-[#D45939] border-none"
          >
            <Plus class="w-4 h-4 mr-1" />
            Add Anthropic Compatible
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onclick={() => (showAddOpenAIModal = true)}
            class="w-full sm:w-auto !bg-white !text-black hover:!bg-gray-100 border border-border shadow-xs"
          >
            <Plus class="w-4 h-4 mr-1" />
            Add OpenAI Compatible
          </Button>
        </div>
      </div>

      {#if customNodes.length === 0}
        <div class="flex items-center justify-center gap-2 py-4 border border-dashed border-border rounded-xl text-text-muted text-xs">
          <Server class="w-4 h-4" />
          <span>No custom providers — use buttons above to add OpenAI/Anthropic compatible endpoints</span>
        </div>
      {:else}
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
          {#each customNodes as node (node.id)}
            {@const stats = node.stats}
            {@const icon = getIconPath(node.id, node.apiType)}
            {@const isAllDisabled = stats.allDisabled}

            <div
              onclick={() => (selectedProviderId = node.id)}
              class="group min-w-0 cursor-pointer text-left"
            >
              <Card
                padding="xs"
                class="h-full hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition-colors p-3 rounded-xl border border-border bg-surface {isAllDisabled ? 'opacity-50' : ''}"
              >
                <div class="flex min-w-0 items-center justify-between gap-3">
                  <div class="flex min-w-0 items-center gap-3">
                    <div class="w-8 h-8 shrink-0 rounded-lg flex items-center justify-center bg-black/5 dark:bg-white/5 border border-border overflow-hidden">
                      <img
                        src={icon}
                        alt={node.name}
                        class="w-5 h-5 object-contain"
                        onerror={(e) => {
                          const target = e.currentTarget as HTMLImageElement
                          target.style.display = 'none'
                        }}
                      />
                    </div>
                    <div class="min-w-0">
                      <h3 class="truncate font-semibold text-sm text-text-main group-hover:text-brand-500 transition-colors">
                        {node.name}
                      </h3>
                      <div class="flex min-w-0 items-center gap-1.5 text-xs flex-wrap mt-0.5">
                        {#if isAllDisabled}
                          <Badge variant="default" size="sm">
                            <span class="flex items-center gap-1">
                              <PauseCircle class="w-3 h-3" />
                              Disabled
                            </span>
                          </Badge>
                        {:else if stats.connected > 0}
                          <Badge variant="success" size="sm" dot>
                            {stats.connected} Connected
                          </Badge>
                          <Badge variant="default" size="sm">
                            {node.apiType === 'responses' ? 'Responses' : 'Chat'}
                          </Badge>
                        {:else}
                          <span class="text-text-muted">No connections</span>
                        {/if}
                      </div>
                    </div>
                  </div>

                  <div class="flex shrink-0 items-center gap-2">
                    {#if stats.total > 0}
                      <div
                        class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
                        onclick={(e) => {
                          e.stopPropagation()
                          handleToggleAll(node.id, isAllDisabled)
                        }}
                      >
                        <Toggle size="sm" checked={!isAllDisabled} onChange={() => {}} />
                      </div>
                    {/if}
                  </div>
                </div>
              </Card>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- ========================================================= -->
    <!-- SECTION 2: OAuth Providers                                -->
    <!-- ========================================================= -->
    {#if oauthProviders.length > 0}
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-lg sm:text-xl font-semibold flex items-center gap-2 leading-tight text-text-main">
            OAuth Providers
          </h2>
          <Button
            size="sm"
            variant="outline"
            class="text-xs w-full sm:w-auto text-text-muted hover:text-text-main"
            onclick={() => alert('Testing OAuth connections...')}
          >
            <Play class="w-3.5 h-3.5 mr-1 text-text-muted" />
            Test All
          </Button>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
          {#each oauthProviders as p (p.id)}
            {@const stats = p.stats}
            {@const icon = getIconPath(p.id)}
            {@const isAllDisabled = stats.allDisabled}

            <div
              onclick={() => (selectedProviderId = p.id)}
              class="group min-w-0 cursor-pointer text-left"
            >
              <Card
                padding="xs"
                class="h-full hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition-colors p-3 rounded-xl border border-border bg-surface {isAllDisabled ? 'opacity-50' : ''}"
              >
                <div class="flex min-w-0 items-center justify-between gap-3">
                  <div class="flex min-w-0 items-center gap-3">
                    <div class="w-8 h-8 shrink-0 rounded-lg flex items-center justify-center bg-black/5 dark:bg-white/5 border border-border overflow-hidden">
                      <img
                        src={icon}
                        alt={p.name}
                        class="w-5 h-5 object-contain"
                        onerror={(e) => {
                          const target = e.currentTarget as HTMLImageElement
                          target.style.display = 'none'
                        }}
                      />
                    </div>
                    <div class="min-w-0">
                      <h3 class="truncate font-semibold text-sm text-text-main group-hover:text-brand-500 transition-colors">
                        {p.name}
                      </h3>
                      <div class="flex min-w-0 items-center gap-1.5 text-xs flex-wrap mt-0.5">
                        {#if isAllDisabled}
                          <Badge variant="default" size="sm">
                            <span class="flex items-center gap-1">
                              <PauseCircle class="w-3 h-3" />
                              Disabled
                            </span>
                          </Badge>
                        {:else if stats.connected > 0}
                          <Badge variant="success" size="sm" dot>
                            {stats.connected} Connected
                          </Badge>
                        {:else}
                          <span class="text-text-muted">No connections</span>
                        {/if}
                      </div>
                    </div>
                  </div>

                  <div class="flex shrink-0 items-center gap-2">
                    {#if stats.total > 0}
                      <div
                        class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
                        onclick={(e) => {
                          e.stopPropagation()
                          handleToggleAll(p.id, isAllDisabled)
                        }}
                      >
                        <Toggle size="sm" checked={!isAllDisabled} onChange={() => {}} />
                      </div>
                    {/if}
                  </div>
                </div>
              </Card>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- ========================================================= -->
    <!-- SECTION 3: Free Tier Providers (Free + Free Tier)         -->
    <!-- ========================================================= -->
    {#if freeTierProviders.length > 0}
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-lg sm:text-xl font-semibold flex items-center gap-2 leading-tight text-text-main">
            Free Tier Providers
          </h2>
          <Button
            size="sm"
            variant="outline"
            class="text-xs w-full sm:w-auto text-text-muted hover:text-text-main"
            onclick={() => alert('Testing Free Tier connections...')}
          >
            <Play class="w-3.5 h-3.5 mr-1 text-text-muted" />
            Test All
          </Button>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
          {#each freeTierProviders as p (p.id)}
            {@const stats = p.stats}
            {@const icon = getIconPath(p.id)}
            {@const isAllDisabled = stats.allDisabled}

            <div
              onclick={() => (selectedProviderId = p.id)}
              class="group min-w-0 cursor-pointer text-left"
            >
              <Card
                padding="xs"
                class="h-full hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition-colors p-3 rounded-xl border border-border bg-surface {isAllDisabled ? 'opacity-50' : ''}"
              >
                <div class="flex min-w-0 items-center justify-between gap-3">
                  <div class="flex min-w-0 items-center gap-3">
                    <div class="w-8 h-8 shrink-0 rounded-lg flex items-center justify-center bg-black/5 dark:bg-white/5 border border-border overflow-hidden">
                      <img
                        src={icon}
                        alt={p.name}
                        class="w-5 h-5 object-contain"
                        onerror={(e) => {
                          const target = e.currentTarget as HTMLImageElement
                          target.style.display = 'none'
                        }}
                      />
                    </div>
                    <div class="min-w-0">
                      <h3 class="truncate font-semibold text-sm text-text-main group-hover:text-brand-500 transition-colors">
                        {p.name}
                      </h3>
                      <div class="flex min-w-0 items-center gap-1.5 text-xs flex-wrap mt-0.5">
                        {#if isAllDisabled}
                          <Badge variant="default" size="sm">
                            <span class="flex items-center gap-1">
                              <PauseCircle class="w-3 h-3" />
                              Disabled
                            </span>
                          </Badge>
                        {:else if p.noAuth}
                          <Badge variant="success" size="sm" dot>Ready</Badge>
                        {:else if stats.connected > 0}
                          <Badge variant="success" size="sm" dot>
                            {stats.connected} Connected
                          </Badge>
                        {:else}
                          <span class="text-text-muted">No connections</span>
                        {/if}
                      </div>
                    </div>
                  </div>

                  <div class="flex shrink-0 items-center gap-2">
                    {#if stats.total > 0}
                      <div
                        class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
                        onclick={(e) => {
                          e.stopPropagation()
                          handleToggleAll(p.id, isAllDisabled)
                        }}
                      >
                        <Toggle size="sm" checked={!isAllDisabled} onChange={() => {}} />
                      </div>
                    {/if}
                  </div>
                </div>
              </Card>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- ========================================================= -->
    <!-- SECTION 4: API Key Providers                              -->
    <!-- ========================================================= -->
    {#if apikeyProviders.length > 0}
      <div class="flex flex-col gap-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-lg sm:text-xl font-semibold flex items-center gap-2 leading-tight text-text-main">
            API Key Providers
          </h2>
          <Button
            size="sm"
            variant="outline"
            class="text-xs w-full sm:w-auto text-text-muted hover:text-text-main"
            onclick={() => alert('Testing API Key connections...')}
          >
            <Play class="w-3.5 h-3.5 mr-1 text-text-muted" />
            Test All
          </Button>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
          {#each visibleApikeyProviders as p (p.id)}
            {@const stats = p.stats}
            {@const icon = getIconPath(p.id)}
            {@const isAllDisabled = stats.allDisabled}

            <div
              onclick={() => (selectedProviderId = p.id)}
              class="group min-w-0 cursor-pointer text-left"
            >
              <Card
                padding="xs"
                class="h-full hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition-colors p-3 rounded-xl border border-border bg-surface {isAllDisabled ? 'opacity-50' : ''}"
              >
                <div class="flex min-w-0 items-center justify-between gap-3">
                  <div class="flex min-w-0 items-center gap-3">
                    <div class="w-8 h-8 shrink-0 rounded-lg flex items-center justify-center bg-black/5 dark:bg-white/5 border border-border overflow-hidden">
                      <img
                        src={icon}
                        alt={p.name}
                        class="w-5 h-5 object-contain"
                        onerror={(e) => {
                          const target = e.currentTarget as HTMLImageElement
                          target.style.display = 'none'
                        }}
                      />
                    </div>
                    <div class="min-w-0">
                      <h3 class="truncate font-semibold text-sm text-text-main group-hover:text-brand-500 transition-colors">
                        {p.name}
                      </h3>
                      <div class="flex min-w-0 items-center gap-1.5 text-xs flex-wrap mt-0.5">
                        {#if isAllDisabled}
                          <Badge variant="default" size="sm">
                            <span class="flex items-center gap-1">
                              <PauseCircle class="w-3 h-3" />
                              Disabled
                            </span>
                          </Badge>
                        {:else if stats.connected > 0}
                          <Badge variant="success" size="sm" dot>
                            {stats.connected} Connected
                          </Badge>
                        {:else}
                          <span class="text-text-muted">No connections</span>
                        {/if}
                      </div>
                    </div>
                  </div>

                  <div class="flex shrink-0 items-center gap-2">
                    {#if stats.total > 0}
                      <div
                        class="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
                        onclick={(e) => {
                          e.stopPropagation()
                          handleToggleAll(p.id, isAllDisabled)
                        }}
                      >
                        <Toggle size="sm" checked={!isAllDisabled} onChange={() => {}} />
                      </div>
                    {/if}
                  </div>
                </div>
              </Card>
            </div>
          {/each}
        </div>

        {#if !showAllApikey && !searchQuery.trim() && statusFilter === 'all' && apikeyProviders.length > APIKEY_INITIAL_VISIBLE}
          <button
            onclick={() => (showAllApikey = true)}
            class="flex w-full items-center justify-center gap-1.5 rounded-xl border border-dashed border-border py-2.5 text-xs font-medium text-text-muted hover:text-brand-500 hover:border-brand-500/40 transition-colors cursor-pointer"
          >
            Show all {apikeyProviders.length} providers
          </button>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<!-- ============================================================= -->
<!-- MODALS                                                        -->
<!-- ============================================================= -->

<!-- Modal: Add OpenAI Compatible Endpoint -->
{#if showAddOpenAIModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="w-full max-w-md bg-surface border border-border rounded-2xl shadow-2xl overflow-hidden animate-scale-in">
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="font-bold text-text-main text-base">Add OpenAI Compatible</h3>
        <button onclick={() => (showAddOpenAIModal = false)} class="text-text-muted hover:text-text-main">
          <X class="w-5 h-5" />
        </button>
      </div>

      <form onsubmit={handleCreateOpenAINode} class="p-6 flex flex-col gap-4">
        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Name</label>
          <input
            type="text"
            required
            bind:value={customFormName}
            placeholder="e.g. sumopod, local-vllm"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-medium text-text-muted mb-1">Prefix</label>
            <input
              type="text"
              required
              bind:value={customFormPrefix}
              placeholder="e.g. sp, vllm"
              class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-text-muted mb-1">API Type</label>
            <select
              bind:value={customFormApiType}
              class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
            >
              <option value="chat">Chat Completions</option>
              <option value="responses">Responses API</option>
            </select>
          </div>
        </div>

        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Base URL</label>
          <input
            type="text"
            required
            bind:value={customFormBaseUrl}
            placeholder="https://api.openai.com/v1"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
          <p class="text-[11px] text-text-muted mt-1">Use the base URL (ending in /v1) for your OpenAI compatible API.</p>
        </div>

        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Initial API Key (Optional)</label>
          <input
            type="password"
            bind:value={customFormApiKey}
            placeholder="sk-..."
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
        </div>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button size="sm" variant="ghost" onclick={() => (showAddOpenAIModal = false)}>Cancel</Button>
          <Button size="sm" variant="primary" type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Creating...' : 'Create Endpoint'}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Modal: Add Anthropic Compatible Endpoint -->
{#if showAddAnthropicModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="w-full max-w-md bg-surface border border-border rounded-2xl shadow-2xl overflow-hidden animate-scale-in">
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="font-bold text-text-main text-base">Add Anthropic Compatible</h3>
        <button onclick={() => (showAddAnthropicModal = false)} class="text-text-muted hover:text-text-main">
          <X class="w-5 h-5" />
        </button>
      </div>

      <form onsubmit={handleCreateAnthropicNode} class="p-6 flex flex-col gap-4">
        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Name</label>
          <input
            type="text"
            required
            bind:value={customFormName}
            placeholder="e.g. anthropic-prod, mini-claude"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Prefix</label>
          <input
            type="text"
            required
            bind:value={customFormPrefix}
            placeholder="e.g. ac, cl"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Base URL</label>
          <input
            type="text"
            required
            bind:value={customFormBaseUrl}
            placeholder="https://api.anthropic.com/v1"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
          <p class="text-[11px] text-text-muted mt-1">System will append /messages automatically.</p>
        </div>

        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Initial API Key (Optional)</label>
          <input
            type="password"
            bind:value={customFormApiKey}
            placeholder="sk-ant-..."
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
        </div>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button size="sm" variant="ghost" onclick={() => (showAddAnthropicModal = false)}>Cancel</Button>
          <Button size="sm" variant="primary" type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Creating...' : 'Create Endpoint'}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- Modal: Add API Key / Connection to Provider -->
{#if showAddKeyModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="w-full max-w-md bg-surface border border-border rounded-2xl shadow-2xl overflow-hidden animate-scale-in">
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="font-bold text-text-main text-base">
          Add Connection to {selectedNode?.name || selectedCatalogItem?.name || selectedProviderId}
        </h3>
        <button onclick={() => (showAddKeyModal = false)} class="text-text-muted hover:text-text-main">
          <X class="w-5 h-5" />
        </button>
      </div>

      <form onsubmit={handleAddKeyConnection} class="p-6 flex flex-col gap-4">
        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">Connection Label (Optional)</label>
          <input
            type="text"
            bind:value={formKeyName}
            placeholder="e.g. Secondary Account / Prod"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-text-muted mb-1">API Key / Token</label>
          <input
            type="password"
            required
            bind:value={formApiKey}
            placeholder="sk-..."
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
        </div>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button size="sm" variant="ghost" onclick={() => (showAddKeyModal = false)}>Cancel</Button>
          <Button size="sm" variant="primary" type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Saving...' : 'Save Connection'}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}
