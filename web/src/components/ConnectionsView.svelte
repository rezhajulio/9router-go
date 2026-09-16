<script lang="ts">
  import {
    Activity,
    AlertTriangle,
    Check,
    ChevronDown,
    ChevronRight,
    Copy,
    Cpu,
    ExternalLink,
    Globe,
    Key,
    Layers,
    Loader2,
    Plus,
    Power,
    RefreshCw,
    Search,
    Shield,
    Sparkles,
    Trash2,
    Zap
  } from 'lucide-svelte'
  import { api, type ProviderConnection } from '../api/client'
  import {
    PROVIDER_CATALOG,
    PROVIDER_CATEGORIES,
    type ProviderCatalogItem
  } from '../lib/providers'
  import Button from '../lib/ui/Button.svelte'
  import Badge from '../lib/ui/Badge.svelte'

  let {
    connections = [],
    onRefresh
  }: {
    connections: ProviderConnection[]
    onRefresh: () => void
  } = $props()

  // Filters
  let search = $state('')
  let statusFilter = $state<'all' | 'connected' | 'not_connected'>('all')
  let activeCategory = $state<string>('all')

  // Expanded provider cards (show account list)
  let expandedProviders = $state<Record<string, boolean>>({})

  // Modals
  let isAddCompatibleOpen = $state(false)
  let compatiblePreset = $state<'openai' | 'anthropic'>('openai')
  let isConnectKeyOpen = $state(false)
  let connectTargetProvider = $state<ProviderCatalogItem | null>(null)
  let isFreebuffModalOpen = $state(false)
  let isAntigravityModalOpen = $state(false)

  // Form states
  let formName = $state('')
  let formApiKey = $state('')
  let formBaseUrl = $state('')
  let formPriority = $state(1)
  let isSubmitting = $state(false)

  // Latency & copy states
  let latencyMap = $state<Record<string, number | 'error' | 'testing'>>({})
  let isPingingAll = $state(false)
  let copiedId = $state<string | null>(null)

  // Freebuff flow
  let fbAuthCode = $state('')
  let fbFingerprint = $state('')
  let fbStatus = $state<'idle' | 'polling' | 'success' | 'error'>('idle')
  let fbMessage = $state('')

  // Map connections by provider id
  let connectionsByProvider = $derived.by(() => {
    const map = new Map<string, ProviderConnection[]>()
    for (const c of connections) {
      const list = map.get(c.provider) || []
      list.push(c)
      map.set(c.provider, list)
    }
    return map
  })

  // Detect custom compatible connections (openai-compatible-*, anthropic-compatible-*, or prefixed)
  let customConnections = $derived(
    connections.filter(
      (c) =>
        c.provider.startsWith('openai-compatible') ||
        c.provider.startsWith('anthropic-compatible') ||
        c.provider.startsWith('custom-')
    )
  )

  function copyText(text: string, id: string) {
    navigator.clipboard.writeText(text)
    copiedId = id
    setTimeout(() => (copiedId = null), 2000)
  }

  function toggleExpand(providerId: string) {
    expandedProviders[providerId] = !expandedProviders[providerId]
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
    if (!confirm(`Revoke and delete connection "${conn.name || conn.id}"?`)) return
    try {
      await api.deleteConnection(conn.id)
      onRefresh()
    } catch (err) {
      alert(`Delete failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleTestPing(connId: string, provider: string) {
    latencyMap[connId] = 'testing'
    const start = performance.now()
    try {
      const res = await fetch('/v1/models', {
        headers: { 'x-provider': provider },
      })
      const rtt = Math.round(performance.now() - start)
      latencyMap[connId] = res.ok ? rtt : 'error'
    } catch {
      latencyMap[connId] = 'error'
    }
  }

  async function handleTestAll() {
    isPingingAll = true
    for (const conn of connections.filter((c) => c.isActive === 1)) {
      handleTestPing(conn.id, conn.provider)
    }
    setTimeout(() => (isPingingAll = false), 2500)
  }

  function openConnectModal(item: ProviderCatalogItem) {
    if (item.id === 'antigravity') {
      isAntigravityModalOpen = true
      return
    }
    if (item.id === 'freebuff') {
      isFreebuffModalOpen = true
      startFreebuffFlow()
      return
    }
    connectTargetProvider = item
    formName = `${item.name} Key`
    formApiKey = ''
    formBaseUrl = ''
    formPriority = 1
    isConnectKeyOpen = true
  }

  function openAddCompatible(preset: 'openai' | 'anthropic') {
    compatiblePreset = preset
    formName = preset === 'anthropic' ? 'Anthropic Compatible' : 'OpenAI Compatible'
    formBaseUrl = preset === 'anthropic' ? 'https://api.anthropic.com/v1' : 'https://api.openai.com/v1'
    formApiKey = ''
    formPriority = 1
    isAddCompatibleOpen = true
  }

  async function handleSaveKeyConnection(e: SubmitEvent) {
    e.preventDefault()
    if (!connectTargetProvider || !formApiKey) return
    isSubmitting = true
    try {
      await api.createConnection({
        provider: connectTargetProvider.id,
        authType: connectTargetProvider.category === 'oauth' ? 'oauth' : 'apikey',
        name: formName || connectTargetProvider.name,
        apiKey: formApiKey,
        data: formBaseUrl ? JSON.stringify({ baseUrl: formBaseUrl }) : undefined,
      })
      isConnectKeyOpen = false
      connectTargetProvider = null
      onRefresh()
    } catch (err) {
      alert(`Connect failed: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function handleSaveCompatible(e: SubmitEvent) {
    e.preventDefault()
    if (!formApiKey) return
    isSubmitting = true
    try {
      await api.createConnection({
        provider: compatiblePreset,
        authType: 'compatible',
        name: formName || `${compatiblePreset}-custom`,
        apiKey: formApiKey,
        data: formBaseUrl ? JSON.stringify({ baseUrl: formBaseUrl }) : undefined,
      })
      isAddCompatibleOpen = false
      onRefresh()
    } catch (err) {
      alert(`Add compatible failed: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function startFreebuffFlow() {
    const chars = '0123456789abcdef'
    let code = ''
    for (let i = 0; i < 32; i++) code += chars[Math.floor(Math.random() * chars.length)]
    fbAuthCode = code
    fbFingerprint = 'cli_' + code.substring(0, 16)
    fbStatus = 'polling'
    fbMessage = 'Browser opened. Authorize token on Freebuff...'

    window.open(`https://freebuff.com/login?auth_code=${fbAuthCode}`, '_blank')

    const pollInterval = setInterval(async () => {
      if (fbStatus !== 'polling') {
        clearInterval(pollInterval)
        return
      }
      try {
        const res = await api.pollFreebuff(fbFingerprint, fbAuthCode)
        if (res && res.status === 'authorized') {
          clearInterval(pollInterval)
          fbStatus = 'success'
          fbMessage = 'Freebuff authorized! Credentials saved.'
          onRefresh()
        }
      } catch {
        // continue
      }
    }, 2500)

    setTimeout(() => {
      if (fbStatus === 'polling') {
        clearInterval(pollInterval)
        fbStatus = 'error'
        fbMessage = 'Pairing timed out. Try again.'
      }
    }, 180000)
  }

  // Filter helper for catalog items
  function matchFilter(item: ProviderCatalogItem): boolean {
    if (activeCategory !== 'all' && item.category !== activeCategory) return false

    const conns = connectionsByProvider.get(item.id) || []
    const isConnected = conns.length > 0
    if (statusFilter === 'connected' && !isConnected) return false
    if (statusFilter === 'not_connected' && isConnected) return false

    if (search.trim()) {
      const q = search.toLowerCase()
      const matchName = item.name.toLowerCase().includes(q)
      const matchId = item.id.toLowerCase().includes(q)
      const matchAlias = item.alias?.toLowerCase().includes(q)
      const matchConn = conns.some(
        (c) =>
          c.name?.toLowerCase().includes(q) ||
          c.email?.toLowerCase().includes(q) ||
          (c.baseUrl && c.baseUrl.toLowerCase().includes(q))
      )
      return matchName || matchId || matchAlias || matchConn
    }

    return true
  }

  // Summary counts
  let totalConnectedProviders = $derived(
    new Set(connections.map((c) => c.provider)).size
  )
  let totalActiveConnections = $derived(
    connections.filter((c) => c.isActive === 1).length
  )

  // Show-all state for large API-key section
  let showAllApiKey = $state(false)
</script>

<div class="space-y-8">
  <!-- Top Action & Status Bar -->
  <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
    <div>
      <div class="flex items-center gap-2 mb-1">
        <span
          class="text-[10px] uppercase tracking-wider text-brand-500 font-code font-bold px-2 py-0.5 rounded bg-brand-500/10 border border-brand-500/25"
        >
          Provider Matrix
        </span>
        <span class="text-xs text-text-muted">•</span>
        <span class="text-xs font-code text-success flex items-center gap-1.5">
          <span class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"></span>
          {totalActiveConnections} active accounts across {totalConnectedProviders} providers
        </span>
      </div>
      <h2 class="text-2xl font-bold tracking-tight text-text-main font-headline">
        Providers & Endpoints
      </h2>
      <p class="text-xs text-text-muted mt-1">
        Manage upstream LLMs, multi-account OAuth pools, and OpenAI/Anthropic compatible proxies.
      </p>
    </div>

    <!-- Actions -->
    <div class="flex flex-wrap items-center gap-2">
      <Button variant="primary" size="sm" onclick={() => openAddCompatible('anthropic')}>
        <Plus class="w-3.5 h-3.5" />
        Add Anthropic Compatible
      </Button>
      <Button variant="secondary" size="sm" onclick={() => openAddCompatible('openai')}>
        <Plus class="w-3.5 h-3.5" />
        Add OpenAI Compatible
      </Button>
      <Button variant="secondary" size="sm" onclick={handleTestAll} disabled={isPingingAll}>
        {#if isPingingAll}
          <Loader2 class="w-3.5 h-3.5 animate-spin text-info" />
        {:else}
          <Activity class="w-3.5 h-3.5 text-info" />
        {/if}
        Test All Active
      </Button>
    </div>
  </div>

  <!-- Filter Rail -->
  <div
    class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 p-3 rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)]"
  >
    <!-- Status Filter -->
    <div class="flex items-center gap-1 overflow-x-auto pb-1 sm:pb-0">
      <button
        type="button"
        onclick={() => (statusFilter = 'all')}
        class="px-3 py-1.5 rounded-[8px] text-xs font-medium transition-colors cursor-pointer {statusFilter ===
        'all'
          ? 'bg-surface-3 text-text-main font-semibold'
          : 'text-text-muted hover:text-text-main hover:bg-surface-2'}"
      >
        All ({PROVIDER_CATALOG.length + customConnections.length})
      </button>

      <button
        type="button"
        onclick={() => (statusFilter = 'connected')}
        class="px-3 py-1.5 rounded-[8px] text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5 {statusFilter ===
        'connected'
          ? 'bg-surface-3 text-text-main font-semibold'
          : 'text-text-muted hover:text-text-main hover:bg-surface-2'}"
      >
        <span>Connected</span>
        <Badge tone="success" class="text-[10px] px-1.5 py-0">{connections.length}</Badge>
      </button>

      <button
        type="button"
        onclick={() => (statusFilter = 'not_connected')}
        class="px-3 py-1.5 rounded-[8px] text-xs font-medium transition-colors cursor-pointer {statusFilter ===
        'not_connected'
          ? 'bg-surface-3 text-text-main font-semibold'
          : 'text-text-muted hover:text-text-main hover:bg-surface-2'}"
      >
        Not Connected
      </button>

      <!-- Category quick tabs -->
      <div class="h-4 w-px bg-border mx-1"></div>

      <button
        type="button"
        onclick={() => (activeCategory = 'all')}
        class="px-2.5 py-1 rounded-[6px] text-xs transition cursor-pointer {activeCategory ===
        'all'
          ? 'bg-brand-500/10 text-brand-600 dark:text-brand-400 font-semibold'
          : 'text-text-subtle hover:text-text-main'}"
      >
        All Groups
      </button>
      {#each PROVIDER_CATEGORIES as cat (cat.id)}
        <button
          type="button"
          onclick={() => (activeCategory = cat.id)}
          class="px-2.5 py-1 rounded-[6px] text-xs whitespace-nowrap transition cursor-pointer {activeCategory ===
          cat.id
            ? 'bg-brand-500/10 text-brand-600 dark:text-brand-400 font-semibold'
            : 'text-text-subtle hover:text-text-main'}"
        >
          {cat.label.split(' ')[0]}
        </button>
      {/each}
    </div>

    <!-- Search Input -->
    <div class="relative min-w-[240px]">
      <Search class="w-3.5 h-3.5 text-text-subtle absolute left-3 top-2.5 pointer-events-none" />
      <input
        type="text"
        bind:value={search}
        placeholder="Filter by name, alias, key, or email..."
        class="w-full bg-surface-2 border border-border-subtle rounded-[10px] pl-9 pr-3 py-1.5 text-xs text-text-main placeholder:text-text-subtle focus:outline-none focus:border-brand-500/50"
      />
    </div>
  </div>

  <!-- SECTION 0: Custom Compatible Endpoints (from user's DB) -->
  {#if activeCategory === 'all' || activeCategory === 'custom'}
    {#if customConnections.length > 0 || statusFilter !== 'connected'}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Cpu class="w-4 h-4 text-brand-500" />
            <h3 class="text-base font-semibold text-text-main">
              Custom Compatible Endpoints
            </h3>
            <Badge tone="default">{customConnections.length}</Badge>
          </div>
          <span class="text-xs text-text-subtle">
            User-registered OpenAI & Anthropic reverse proxies
          </span>
        </div>

        {#if customConnections.length === 0}
          <div
            class="p-6 rounded-[14px] border border-dashed border-border text-center text-xs text-text-muted"
          >
            No custom compatible endpoints registered yet. Click "+ Add OpenAI Compatible" or "+ Add Anthropic Compatible" above.
          </div>
        {:else}
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {#each customConnections as conn (conn.id)}
              {@const isActive = conn.isActive === 1}
              {@const latency = latencyMap[conn.id]}

              <div
                class="p-4 rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)] hover:shadow-[var(--shadow-warm)] transition flex flex-col justify-between gap-3"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="flex items-center gap-2.5 min-w-0">
                    <div
                      class="size-8 rounded-[8px] bg-bg flex items-center justify-center font-bold text-xs font-code text-brand-500 flex-shrink-0"
                    >
                      {conn.provider.startsWith('anthropic') ? 'AN' : 'OA'}
                    </div>
                    <div class="min-w-0">
                      <p class="font-semibold text-xs text-text-main truncate">
                        {conn.name || conn.provider}
                      </p>
                      <p class="font-code text-[11px] text-text-muted truncate">
                        {conn.baseUrl || 'endpoint'}
                      </p>
                    </div>
                  </div>

                  <Badge tone={isActive ? 'success' : 'default'} class="text-[10px]">
                    {isActive ? 'Active' : 'Disabled'}
                  </Badge>
                </div>

                <div class="pt-2 border-t border-border-subtle flex items-center justify-between text-xs font-code">
                  <div class="text-text-muted">
                    {#if latency === 'testing'}
                      <Loader2 class="w-3 h-3 animate-spin text-info" />
                    {:else if typeof latency === 'number'}
                      <span class="text-success font-bold">{latency}ms</span>
                    {:else}
                      <button
                        type="button"
                        onclick={() => handleTestPing(conn.id, conn.provider)}
                        class="text-info hover:underline cursor-pointer"
                      >
                        Ping
                      </button>
                    {/if}
                  </div>

                  <div class="flex items-center gap-1">
                    <button
                      type="button"
                      onclick={() => handleToggleConnection(conn)}
                      class="p-1 rounded text-text-subtle hover:text-text-main cursor-pointer"
                      title={isActive ? 'Disable' : 'Enable'}
                    >
                      <Power class="w-3.5 h-3.5 {isActive ? 'text-success' : ''}" />
                    </button>
                    <button
                      type="button"
                      onclick={() => handleDeleteConnection(conn)}
                      class="p-1 rounded text-text-subtle hover:text-danger cursor-pointer"
                      title="Delete"
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}

  <!-- SECTION 1: OAuth Providers -->
  {#if activeCategory === 'all' || activeCategory === 'oauth'}
    {@const oauthItems = PROVIDER_CATALOG.filter((p) => p.category === 'oauth').filter(matchFilter)}
    {#if oauthItems.length > 0}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Shield class="w-4 h-4 text-info" />
            <h3 class="text-base font-semibold text-text-main">OAuth Providers</h3>
            <Badge tone="default">{oauthItems.length}</Badge>
          </div>
          <span class="text-xs text-text-subtle">
            Multi-account rotation, Google Antigravity, Kiro, Codex, Cursor
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {#each oauthItems as item (item.id)}
            {@const conns = connectionsByProvider.get(item.id) || []}
            {@const activeConns = conns.filter((c) => c.isActive === 1)}
            {@const isExpanded = !!expandedProviders[item.id]}

            <div
              class="rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)] hover:shadow-[var(--shadow-warm)] transition flex flex-col justify-between overflow-hidden"
            >
              <div class="p-4 space-y-3">
                <!-- Header -->
                <div class="flex items-start justify-between gap-2">
                  <div class="flex items-center gap-2.5 min-w-0">
                    <div
                      class="size-8 rounded-[8px] flex items-center justify-center font-bold text-xs font-code flex-shrink-0"
                      style="background-color: {item.color}22; color: {item.color};"
                    >
                      {item.alias?.substring(0, 2).toUpperCase() || 'OA'}
                    </div>
                    <div class="min-w-0">
                      <p class="font-semibold text-xs text-text-main truncate">{item.name}</p>
                      <p class="font-code text-[11px] text-text-muted">
                        /{item.alias}
                      </p>
                    </div>
                  </div>

                  {#if conns.length > 0}
                    <Badge tone="success" class="text-[10px]">
                      {activeConns.length}/{conns.length} Connected
                    </Badge>
                  {:else}
                    <Badge tone="default" class="text-[10px]">Not Connected</Badge>
                  {/if}
                </div>

                <!-- Expanded Accounts List -->
                {#if isExpanded && conns.length > 0}
                  <div class="space-y-1.5 pt-2 border-t border-border-subtle font-code text-[11px]">
                    {#each conns as conn (conn.id)}
                      {@const isActive = conn.isActive === 1}
                      {@const latency = latencyMap[conn.id]}

                      <div
                        class="p-2 rounded-[8px] bg-surface-2 flex items-center justify-between gap-2"
                      >
                        <div class="min-w-0">
                          <p class="text-xs text-text-main font-medium truncate">
                            {conn.name || conn.email || conn.id.slice(0, 8)}
                          </p>
                          <span class="text-[10px] text-text-subtle">Slot P{conn.priority || 1}</span>
                        </div>

                        <div class="flex items-center gap-1.5 flex-shrink-0">
                          {#if latency === 'testing'}
                            <Loader2 class="w-3 h-3 animate-spin text-info" />
                          {:else if typeof latency === 'number'}
                            <span class="text-[10px] text-success font-bold">{latency}ms</span>
                          {:else}
                            <button
                              type="button"
                              onclick={() => handleTestPing(conn.id, conn.provider)}
                              class="text-[10px] text-info hover:underline"
                            >
                              ping
                            </button>
                          {/if}

                          <button
                            type="button"
                            onclick={() => handleToggleConnection(conn)}
                            class="p-1 text-text-subtle hover:text-text-main"
                            title={isActive ? 'Disable' : 'Enable'}
                          >
                            <Power class="w-3 h-3 {isActive ? 'text-success' : ''}" />
                          </button>

                          <button
                            type="button"
                            onclick={() => handleDeleteConnection(conn)}
                            class="p-1 text-text-subtle hover:text-danger"
                            title="Delete"
                          >
                            <Trash2 class="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>

              <!-- Footer action -->
              <div
                class="px-4 py-2 bg-surface-2 border-t border-border-subtle flex items-center justify-between text-xs"
              >
                {#if conns.length > 0}
                  <button
                    type="button"
                    onclick={() => toggleExpand(item.id)}
                    class="text-text-muted hover:text-text-main flex items-center gap-1 font-medium cursor-pointer"
                  >
                    {#if isExpanded}
                      <ChevronDown class="w-3.5 h-3.5" />
                      Hide Accounts
                    {:else}
                      <ChevronRight class="w-3.5 h-3.5" />
                      View {conns.length} {conns.length === 1 ? 'Account' : 'Accounts'}
                    {/if}
                  </button>
                {:else}
                  <span class="text-[11px] text-text-subtle">OAuth Ready</span>
                {/if}

                <Button variant="ghost" size="sm" onclick={() => openConnectModal(item)}>
                  <Plus class="w-3 h-3" />
                  {conns.length > 0 ? 'Add Another' : 'Connect'}
                </Button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  <!-- SECTION 2: Free Providers (No auth needed) -->
  {#if activeCategory === 'all' || activeCategory === 'free'}
    {@const freeItems = PROVIDER_CATALOG.filter((p) => p.category === 'free').filter(matchFilter)}
    {#if freeItems.length > 0}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Sparkles class="w-4 h-4 text-success" />
            <h3 class="text-base font-semibold text-text-main">Free Providers</h3>
            <Badge tone="success">Zero Auth</Badge>
          </div>
          <span class="text-xs text-text-subtle">
            Available out-of-the-box without registration
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {#each freeItems as item (item.id)}
            {@const conns = connectionsByProvider.get(item.id) || []}

            <div
              class="p-4 rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)] flex flex-col justify-between gap-3"
            >
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-2.5">
                  <div
                    class="size-8 rounded-[8px] flex items-center justify-center font-bold text-xs font-code"
                    style="background-color: {item.color}22; color: {item.color};"
                  >
                    {item.alias?.substring(0, 2).toUpperCase() || 'FR'}
                  </div>
                  <div>
                    <p class="font-semibold text-xs text-text-main">{item.name}</p>
                    <p class="font-code text-[11px] text-text-muted">/{item.alias}</p>
                  </div>
                </div>

                <Badge tone="success" class="text-[10px]">Free</Badge>
              </div>

              <div class="pt-2 border-t border-border-subtle flex items-center justify-between text-xs">
                <span class="text-text-subtle text-[11px]">Ready to route</span>
                <Button variant="ghost" size="sm" onclick={() => openConnectModal(item)}>
                  Configure
                </Button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  <!-- SECTION 3: Free Tier Providers (Generous credit / tier) -->
  {#if activeCategory === 'all' || activeCategory === 'freeTier'}
    {@const freeTierItems = PROVIDER_CATALOG.filter((p) => p.category === 'freeTier').filter(matchFilter)}
    {#if freeTierItems.length > 0}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Zap class="w-4 h-4 text-warning" />
            <h3 class="text-base font-semibold text-text-main">Free Tier Providers</h3>
            <Badge tone="warning">{freeTierItems.length}</Badge>
          </div>
          <span class="text-xs text-text-subtle">
            Generous recurring free quota (NVIDIA, Gemini CLI, Cloudflare, etc.)
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {#each freeTierItems as item (item.id)}
            {@const conns = connectionsByProvider.get(item.id) || []}
            {@const isConnected = conns.length > 0}

            <div
              class="p-4 rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)] flex flex-col justify-between gap-3"
            >
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-2.5 min-w-0">
                  <div
                    class="size-8 rounded-[8px] flex items-center justify-center font-bold text-xs font-code flex-shrink-0"
                    style="background-color: {item.color}22; color: {item.color};"
                  >
                    {item.alias?.substring(0, 2).toUpperCase() || 'FT'}
                  </div>
                  <div class="min-w-0">
                    <p class="font-semibold text-xs text-text-main truncate">{item.name}</p>
                    <p class="font-code text-[11px] text-text-muted">/{item.alias}</p>
                  </div>
                </div>

                {#if isConnected}
                  <Badge tone="success" class="text-[10px]">
                    {conns.length} Connected
                  </Badge>
                {:else}
                  <Badge tone="default" class="text-[10px]">Available</Badge>
                {/if}
              </div>

              <div class="pt-2 border-t border-border-subtle flex items-center justify-between text-xs">
                <span class="text-[11px] text-text-subtle font-code">
                  {isConnected ? `${conns.filter((c) => c.isActive === 1).length} active` : 'Free Quota'}
                </span>

                <Button variant="ghost" size="sm" onclick={() => openConnectModal(item)}>
                  {isConnected ? 'Add Key' : 'Connect'}
                </Button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}

  <!-- SECTION 4: API Key Providers -->
  {#if activeCategory === 'all' || activeCategory === 'apikey'}
    {@const apikeyItems = PROVIDER_CATALOG.filter((p) => p.category === 'apikey').filter(matchFilter)}
    {@const visibleItems = showAllApiKey ? apikeyItems : apikeyItems.slice(0, 20)}
    {#if apikeyItems.length > 0}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Key class="w-4 h-4 text-brand-500" />
            <h3 class="text-base font-semibold text-text-main">API Key Providers</h3>
            <Badge tone="default">{apikeyItems.length}</Badge>
          </div>
          <span class="text-xs text-text-subtle">
            Commercial and cloud endpoints (OpenAI, Anthropic, DeepSeek, Groq, etc.)
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {#each visibleItems as item (item.id)}
            {@const conns = connectionsByProvider.get(item.id) || []}
            {@const isConnected = conns.length > 0}
            {@const isExpanded = !!expandedProviders[item.id]}

            <div
              class="rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)] hover:shadow-[var(--shadow-warm)] transition flex flex-col justify-between overflow-hidden"
            >
              <div class="p-4 space-y-3">
                <div class="flex items-start justify-between gap-2">
                  <div class="flex items-center gap-2.5 min-w-0">
                    <div
                      class="size-8 rounded-[8px] flex items-center justify-center font-bold text-xs font-code flex-shrink-0"
                      style="background-color: {item.color}22; color: {item.color};"
                    >
                      {item.alias?.substring(0, 2).toUpperCase() || 'AK'}
                    </div>
                    <div class="min-w-0">
                      <p class="font-semibold text-xs text-text-main truncate">{item.name}</p>
                      <p class="font-code text-[11px] text-text-muted">/{item.alias}</p>
                    </div>
                  </div>

                  {#if isConnected}
                    <Badge tone="success" class="text-[10px]">
                      {conns.length} Connected
                    </Badge>
                  {:else}
                    <Badge tone="default" class="text-[10px]">Ready</Badge>
                  {/if}
                </div>

                <!-- Expanded account rows -->
                {#if isExpanded && conns.length > 0}
                  <div class="space-y-1.5 pt-2 border-t border-border-subtle font-code text-[11px]">
                    {#each conns as conn (conn.id)}
                      {@const isActive = conn.isActive === 1}
                      {@const latency = latencyMap[conn.id]}

                      <div
                        class="p-2 rounded-[8px] bg-surface-2 flex items-center justify-between gap-2"
                      >
                        <span class="truncate font-medium text-text-main text-xs">
                          {conn.name || 'Key'} (P{conn.priority || 1})
                        </span>

                        <div class="flex items-center gap-1.5 flex-shrink-0">
                          {#if latency === 'testing'}
                            <Loader2 class="w-3 h-3 animate-spin text-info" />
                          {:else if typeof latency === 'number'}
                            <span class="text-[10px] text-success font-bold">{latency}ms</span>
                          {:else}
                            <button
                              type="button"
                              onclick={() => handleTestPing(conn.id, conn.provider)}
                              class="text-[10px] text-info hover:underline"
                            >
                              ping
                            </button>
                          {/if}

                          <button
                            type="button"
                            onclick={() => handleToggleConnection(conn)}
                            class="p-1 text-text-subtle hover:text-text-main"
                            title={isActive ? 'Disable' : 'Enable'}
                          >
                            <Power class="w-3 h-3 {isActive ? 'text-success' : ''}" />
                          </button>

                          <button
                            type="button"
                            onclick={() => handleDeleteConnection(conn)}
                            class="p-1 text-text-subtle hover:text-danger"
                            title="Delete"
                          >
                            <Trash2 class="w-3 h-3" />
                          </button>
                        </div>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>

              <div
                class="px-4 py-2 bg-surface-2 border-t border-border-subtle flex items-center justify-between text-xs"
              >
                {#if conns.length > 0}
                  <button
                    type="button"
                    onclick={() => toggleExpand(item.id)}
                    class="text-text-muted hover:text-text-main flex items-center gap-1 cursor-pointer font-medium"
                  >
                    {#if isExpanded}
                      <ChevronDown class="w-3.5 h-3.5" />
                      Hide Keys
                    {:else}
                      <ChevronRight class="w-3.5 h-3.5" />
                      View ({conns.length})
                    {/if}
                  </button>
                {:else}
                  <span class="text-[11px] text-text-subtle font-code">Not configured</span>
                {/if}

                <Button variant="ghost" size="sm" onclick={() => openConnectModal(item)}>
                  <Plus class="w-3 h-3" />
                  {isConnected ? 'Add Key' : 'Add Key'}
                </Button>
              </div>
            </div>
          {/each}
        </div>

        {#if apikeyItems.length > 20}
          <div class="text-center pt-2">
            <Button
              variant="outline"
              size="sm"
              onclick={() => (showAllApiKey = !showAllApiKey)}
            >
              {showAllApiKey ? 'Show Less' : `Show All ${apikeyItems.length} Providers`}
            </Button>
          </div>
        {/if}
      </div>
    {/if}
  {/if}

  <!-- SECTION 5: Web Cookie Providers -->
  {#if activeCategory === 'all' || activeCategory === 'webCookie'}
    {@const cookieItems = PROVIDER_CATALOG.filter((p) => p.category === 'webCookie').filter(matchFilter)}
    {#if cookieItems.length > 0}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Globe class="w-4 h-4 text-text-muted" />
            <h3 class="text-base font-semibold text-text-main">Web Cookie Providers</h3>
            <Badge tone="default">{cookieItems.length}</Badge>
          </div>
          <span class="text-xs text-text-subtle">
            Browser cookie session authentication (iFlow, etc.)
          </span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {#each cookieItems as item (item.id)}
            {@const conns = connectionsByProvider.get(item.id) || []}

            <div
              class="p-4 rounded-[14px] bg-surface border border-border-subtle shadow-[var(--shadow-soft)] flex flex-col justify-between gap-3"
            >
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-2.5">
                  <div
                    class="size-8 rounded-[8px] flex items-center justify-center font-bold text-xs font-code"
                    style="background-color: {item.color}22; color: {item.color};"
                  >
                    {item.alias?.substring(0, 2).toUpperCase() || 'CK'}
                  </div>
                  <div>
                    <p class="font-semibold text-xs text-text-main">{item.name}</p>
                    <p class="font-code text-[11px] text-text-muted">/{item.alias}</p>
                  </div>
                </div>

                <Badge tone={conns.length > 0 ? 'success' : 'default'} class="text-[10px]">
                  {conns.length > 0 ? `${conns.length} Connected` : 'Cookie'}
                </Badge>
              </div>

              <div class="pt-2 border-t border-border-subtle flex items-center justify-between text-xs">
                <span class="text-[11px] text-text-subtle">Session auth</span>
                <Button variant="ghost" size="sm" onclick={() => openConnectModal(item)}>
                  Configure
                </Button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<!-- MODAL: Connect API Key -->
{#if isConnectKeyOpen && connectTargetProvider}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
  >
    <div
      class="w-full max-w-md p-6 rounded-[14px] bg-surface border border-border shadow-[var(--shadow-elev)] space-y-4"
    >
      <div class="flex items-center justify-between pb-3 border-b border-border-subtle">
        <div class="flex items-center gap-2.5">
          <div
            class="size-8 rounded-[8px] flex items-center justify-center font-bold text-xs font-code"
            style="background-color: {connectTargetProvider.color}22; color: {connectTargetProvider.color};"
          >
            {connectTargetProvider.alias?.substring(0, 2).toUpperCase()}
          </div>
          <div>
            <h4 class="font-semibold text-sm text-text-main">
              Connect {connectTargetProvider.name}
            </h4>
            <p class="text-xs text-text-muted font-code">/{connectTargetProvider.alias}</p>
          </div>
        </div>
        <button
          type="button"
          onclick={() => (isConnectKeyOpen = false)}
          class="text-text-muted hover:text-text-main text-xs"
        >
          ✕
        </button>
      </div>

      <form onsubmit={handleSaveKeyConnection} class="space-y-3 text-xs font-body">
        <div>
          <label for="key-name-input" class="block font-semibold text-text-muted mb-1">Connection Label</label>
          <input
            id="key-name-input"
            type="text"
            placeholder="e.g. Production Key"
            bind:value={formName}
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 text-text-main focus:outline-none focus:border-brand-500/50"
          />
        </div>

        <div>
          <label for="key-secret-input" class="block font-semibold text-text-muted mb-1">API Key Secret *</label>
          <input
            id="key-secret-input"
            type="password"
            placeholder="sk-..."
            bind:value={formApiKey}
            required
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 font-code text-text-main focus:outline-none focus:border-brand-500/50"
          />
        </div>

        <div>
          <label for="key-url-input" class="block font-semibold text-text-muted mb-1">Custom Base URL (Optional)</label>
          <input
            id="key-url-input"
            type="text"
            placeholder="Leave blank for official provider endpoint"
            bind:value={formBaseUrl}
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 font-code text-text-main focus:outline-none focus:border-brand-500/50"
          />
        </div>

        <div>
          <label for="key-prio-select" class="block font-semibold text-text-muted mb-1">Priority (Routing Order)</label>
          <select
            id="key-prio-select"
            bind:value={formPriority}
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 font-code text-text-main focus:outline-none"
          >
            <option value={1}>P1 - Primary Active</option>
            <option value={2}>P2 - Secondary</option>
            <option value={3}>P3 - Backup Cascade</option>
          </select>
        </div>

        <div class="flex items-center justify-end gap-2 pt-3 border-t border-border-subtle">
          <Button variant="ghost" size="sm" onclick={() => (isConnectKeyOpen = false)}>
            Cancel
          </Button>
          <Button variant="primary" size="sm" disabled={isSubmitting}>
            {#if isSubmitting}
              <Loader2 class="w-3.5 h-3.5 animate-spin" />
            {:else}
              <Check class="w-3.5 h-3.5" />
            {/if}
            Save Connection
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- MODAL: Add Compatible Custom Endpoint -->
{#if isAddCompatibleOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
  >
    <div
      class="w-full max-w-md p-6 rounded-[14px] bg-surface border border-border shadow-[var(--shadow-elev)] space-y-4"
    >
      <div class="flex items-center justify-between pb-3 border-b border-border-subtle">
        <h4 class="font-semibold text-sm text-text-main">
          Add {compatiblePreset === 'anthropic' ? 'Anthropic' : 'OpenAI'} Compatible Endpoint
        </h4>
        <button
          type="button"
          onclick={() => (isAddCompatibleOpen = false)}
          class="text-text-muted hover:text-text-main text-xs"
        >
          ✕
        </button>
      </div>

      <form onsubmit={handleSaveCompatible} class="space-y-3 text-xs font-body">
        <div>
          <label for="comp-name" class="block font-semibold text-text-muted mb-1">Display Name</label>
          <input
            id="comp-name"
            type="text"
            placeholder="e.g. My Custom Proxy"
            bind:value={formName}
            required
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 text-text-main focus:outline-none focus:border-brand-500/50"
          />
        </div>

        <div>
          <label for="comp-url" class="block font-semibold text-text-muted mb-1">Base API URL *</label>
          <input
            id="comp-url"
            type="text"
            placeholder="https://my-proxy.com/v1"
            bind:value={formBaseUrl}
            required
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 font-code text-text-main focus:outline-none focus:border-brand-500/50"
          />
        </div>

        <div>
          <label for="comp-key" class="block font-semibold text-text-muted mb-1">API Key *</label>
          <input
            id="comp-key"
            type="password"
            placeholder="sk-..."
            bind:value={formApiKey}
            required
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 font-code text-text-main focus:outline-none focus:border-brand-500/50"
          />
        </div>

        <div>
          <label for="comp-prio" class="block font-semibold text-text-muted mb-1">Priority</label>
          <select
            id="comp-prio"
            bind:value={formPriority}
            class="w-full bg-surface-2 border border-border-subtle rounded-[8px] px-3 py-2 font-code text-text-main focus:outline-none"
          >
            <option value={1}>P1 - Primary</option>
            <option value={2}>P2 - Secondary</option>
            <option value={3}>P3 - Backup</option>
          </select>
        </div>

        <div class="flex items-center justify-end gap-2 pt-3 border-t border-border-subtle">
          <Button variant="ghost" size="sm" onclick={() => (isAddCompatibleOpen = false)}>
            Cancel
          </Button>
          <Button variant="primary" size="sm" disabled={isSubmitting}>
            {#if isSubmitting}
              <Loader2 class="w-3.5 h-3.5 animate-spin" />
            {:else}
              <Check class="w-3.5 h-3.5" />
            {/if}
            Save Endpoint
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}

<!-- MODAL: Antigravity Google OAuth -->
{#if isAntigravityModalOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
  >
    <div
      class="w-full max-w-md p-6 rounded-[14px] bg-surface border border-border shadow-[var(--shadow-elev)] space-y-4"
    >
      <div class="flex items-center justify-between pb-3 border-b border-border-subtle">
        <h4 class="font-semibold text-sm text-text-main">Google Antigravity OAuth</h4>
        <button
          type="button"
          onclick={() => (isAntigravityModalOpen = false)}
          class="text-text-muted hover:text-text-main text-xs"
        >
          ✕
        </button>
      </div>

      <div class="space-y-3 text-xs text-text-muted">
        <p>
          Authenticate your Google account to access Gemini 1.5/2.0 Pro and Flash models under Antigravity multi-account failover.
        </p>

        <a
          href="/api/oauth/antigravity/login"
          class="flex items-center justify-center gap-2 w-full py-2.5 rounded-[10px] bg-brand-500 hover:bg-brand-600 text-white font-bold transition shadow-[var(--shadow-warm)]"
        >
          <Globe class="w-4 h-4" />
          <span>Launch Google OAuth Login</span>
        </a>

        <div class="flex justify-end pt-2">
          <Button variant="ghost" size="sm" onclick={() => (isAntigravityModalOpen = false)}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- MODAL: Freebuff Device Flow -->
{#if isFreebuffModalOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
  >
    <div
      class="w-full max-w-md p-6 rounded-[14px] bg-surface border border-border shadow-[var(--shadow-elev)] space-y-4"
    >
      <div class="flex items-center justify-between pb-3 border-b border-border-subtle">
        <h4 class="font-semibold text-sm text-text-main">Freebuff Device Flow</h4>
        <button
          type="button"
          onclick={() => (isFreebuffModalOpen = false)}
          class="text-text-muted hover:text-text-main text-xs"
        >
          ✕
        </button>
      </div>

      <div class="space-y-3 text-xs font-body">
        <p class="text-text-muted">
          Authorize token via Freebuff. A new tab has been opened.
        </p>

        <div class="p-3 rounded-[8px] bg-surface-2 border border-border font-code">
          <span class="text-[10px] text-text-subtle uppercase block">Pairing Code</span>
          <span class="text-info font-bold tracking-wider select-all">
            {fbAuthCode || 'Generating...'}
          </span>
        </div>

        <div
          class="p-3 rounded-[8px] flex items-center gap-2 {fbStatus === 'success'
            ? 'bg-success/10 text-success'
            : fbStatus === 'error'
              ? 'bg-danger/10 text-danger'
              : 'bg-surface-2 text-text-muted'}"
        >
          {#if fbStatus === 'polling'}
            <Loader2 class="w-3.5 h-3.5 animate-spin text-info" />
          {:else if fbStatus === 'success'}
            <Check class="w-3.5 h-3.5 text-success" />
          {/if}
          <span>{fbMessage}</span>
        </div>

        <div class="flex justify-end pt-2">
          <Button variant="secondary" size="sm" onclick={() => (isFreebuffModalOpen = false)}>
            Done
          </Button>
        </div>
      </div>
    </div>
  </div>
{/if}
