<script lang="ts">
  import {
    Activity,
    AlertTriangle,
    Check,
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

  let {
    connections = [],
    onRefresh
  }: {
    connections: ProviderConnection[]
    onRefresh: () => void
  } = $props()

  let search = $state('')
  let filterType = $state<'all' | 'active' | 'oauth' | 'custom' | 'free'>('all')

  // Modals
  let isAddOpen = $state(false)
  let addPreset = $state<'openai' | 'anthropic'>('openai')
  let isFreebuffModalOpen = $state(false)
  let isAntigravityModalOpen = $state(false)

  // Form Fields
  let formProvider = $state('openai')
  let formName = $state('')
  let formApiKey = $state('')
  let formBaseUrl = $state('')
  let formPriority = $state(1)
  let isSubmitting = $state(false)

  // Latency ping map
  let latencyMap = $state<Record<string, number | 'error' | 'testing'>>({})
  let isPingingAll = $state(false)
  let copiedId = $state<string | null>(null)

  // Freebuff Device Flow State
  let fbAuthCode = $state('')
  let fbFingerprint = $state('')
  let fbStatus = $state<'idle' | 'polling' | 'success' | 'error'>('idle')
  let fbMessage = $state('')

  function copyText(text: string, id: string) {
    navigator.clipboard.writeText(text)
    copiedId = id
    setTimeout(() => (copiedId = null), 2000)
  }

  let activeCount = $derived(connections.filter((c) => c.isActive === 1).length)
  let oauthCount = $derived(connections.filter((c) => c.type === 'oauth').length)
  let customCount = $derived(connections.filter((c) => c.type !== 'oauth').length)
  let freeCount = $derived(connections.filter((c) => c.provider === 'freebuff' || c.name.toLowerCase().includes('free')).length)

  // Split connections
  let antigravityAccounts = $derived(connections.filter((c) => c.provider === 'antigravity'))
  let freebuffAccounts = $derived(connections.filter((c) => c.provider === 'freebuff'))
  let clineAccounts = $derived(connections.filter((c) => c.provider === 'cline' || c.provider === 'clinepass'))

  let customProxies = $derived(
    connections.filter((c) => {
      if (c.provider === 'antigravity' || c.provider === 'freebuff') return false
      if (search.trim()) {
        const q = search.toLowerCase()
        return (
          c.name.toLowerCase().includes(q) ||
          c.provider.toLowerCase().includes(q) ||
          (c.baseUrl && c.baseUrl.toLowerCase().includes(q))
        )
      }
      if (filterType === 'active') return c.isActive === 1
      if (filterType === 'oauth') return c.type === 'oauth'
      if (filterType === 'custom') return c.type !== 'oauth'
      if (filterType === 'free') return c.name.toLowerCase().includes('free')
      return true
    })
  )

  async function handleToggle(conn: ProviderConnection) {
    try {
      await api.toggleConnection(conn.id)
      onRefresh()
    } catch (err) {
      alert(`Failed to toggle: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleDelete(conn: ProviderConnection) {
    if (!confirm(`Revoke and delete connection "${conn.name || conn.id}"?`)) return
    try {
      await api.deleteConnection(conn.id)
      onRefresh()
    } catch (err) {
      alert(`Failed to delete: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleTestPing(conn: ProviderConnection) {
    latencyMap[conn.id] = 'testing'
    const start = performance.now()
    try {
      const res = await fetch('/v1/models', {
        headers: { 'x-provider': conn.provider },
      })
      const rtt = Math.round(performance.now() - start)
      if (res.ok) {
        latencyMap[conn.id] = rtt
      } else {
        latencyMap[conn.id] = 'error'
      }
    } catch {
      latencyMap[conn.id] = 'error'
    }
  }

  async function handleTestAll() {
    isPingingAll = true
    for (const conn of connections.filter((c) => c.isActive === 1)) {
      handleTestPing(conn)
    }
    setTimeout(() => {
      isPingingAll = false
    }, 2000)
  }

  async function handleAddSubmit(e: SubmitEvent) {
    e.preventDefault()
    if (!formProvider || !formApiKey) return
    isSubmitting = true
    try {
      await api.createConnection({
        provider: formProvider,
        name: formName || `${formProvider}-custom`,
        apiKey: formApiKey,
        baseUrl: formBaseUrl || undefined,
        priority: formPriority,
      })
      isAddOpen = false
      formName = ''
      formApiKey = ''
      formBaseUrl = ''
      onRefresh()
    } catch (err) {
      alert(`Failed to create connection: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  function openAddModal(preset: 'openai' | 'anthropic') {
    addPreset = preset
    formProvider = preset
    formName = preset === 'anthropic' ? 'Anthropic Compatible (Prod)' : 'OpenAI Compatible (Custom)'
    formBaseUrl = preset === 'anthropic' ? 'https://api.anthropic.com/v1' : 'https://api.openai.com/v1'
    isAddOpen = true
  }

  async function startFreebuffFlow() {
    const chars = '0123456789abcdef'
    let code = ''
    for (let i = 0; i < 32; i++) code += chars[Math.floor(Math.random() * chars.length)]
    fbAuthCode = code
    fbFingerprint = 'cli_' + code.substring(0, 16)
    fbStatus = 'polling'
    fbMessage = 'Browser window opened. Log in on Freebuff to authorize token...'

    window.open(`https://freebuff.com/login?auth_code=${fbAuthCode}`, '_blank')

    const pollInterval = setInterval(async () => {
      if (fbStatus !== 'polling') {
        clearInterval(pollInterval)
        return
      }
      try {
        const res = await api.pollFreebuffToken(fbAuthCode, fbFingerprint)
        if (res && res.status === 'success') {
          clearInterval(pollInterval)
          fbStatus = 'success'
          fbMessage = 'Freebuff authorized successfully! Session credentials stored.'
          onRefresh()
        }
      } catch {
        // continue polling
      }
    }, 2500)

    setTimeout(() => {
      if (fbStatus === 'polling') {
        clearInterval(pollInterval)
        fbStatus = 'error'
        fbMessage = 'Device authorization timed out. Please try again.'
      }
    }, 180000)
  }

  const virtualGeminiModels = [
    { id: 'ag/gemini-2.5-flash-high', name: 'Gemini 2.5 Flash [High]' },
    { id: 'ag/gemini-2.5-flash-medium', name: 'Gemini 2.5 Flash [Medium]' },
    { id: 'ag/gemini-2.5-flash-low', name: 'Gemini 2.5 Flash [Low]' },
    { id: 'ag/gemini-2.5-flash', name: 'Gemini 2.5 Flash Standard' },
  ]
</script>

<div class="space-y-6">
  <!-- Top Action & Filter Bar (From Stitch Screenshot) -->
  <div class="space-y-4">
    <div class="flex flex-col lg:flex-row lg:items-end justify-between gap-4">
      <div class="space-y-1.5">
        <div class="flex items-center gap-2">
          <span class="font-code text-[10px] uppercase tracking-wider text-brand-500 px-2 py-0.5 rounded bg-brand-500/10 border border-brand-500/25 font-bold">
            Upstream Matrix
          </span>
          <span class="font-code text-[11px] text-info flex items-center gap-1.5 font-medium">
            <span class="w-1.5 h-1.5 rounded-full bg-info animate-pulse"></span>
            {activeCount} Nodes Active
          </span>
        </div>
        <h1 class="font-headline text-2xl sm:text-3xl font-bold text-text-main tracking-tight">
          Providers & API Connections
        </h1>
        <p class="font-body text-xs sm:text-sm text-text-muted max-w-2xl leading-relaxed">
          Manage your AI upstream providers, multi-account OAuth pools, and OpenAI/Anthropic compatible proxies with automated failover and sub-millisecond health checks.
        </p>
      </div>

      <!-- Action Cluster -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onclick={() => openAddModal('anthropic')}
          class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-body text-xs font-bold shadow-md shadow-brand-500/25 transition cursor-pointer"
        >
          <Plus class="w-4 h-4" />
          <span>+ Add Anthropic Compatible</span>
        </button>

        <button
          type="button"
          onclick={() => openAddModal('openai')}
          class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-surface-2 hover:bg-surface-3 text-text-main font-body text-xs font-semibold border border-border transition cursor-pointer"
        >
          <Plus class="w-4 h-4 text-info" />
          <span>Add OpenAI Compatible</span>
        </button>

        <button
          type="button"
          onclick={() => (isAntigravityModalOpen = true)}
          class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-surface-2 hover:bg-surface-3 text-text-main font-body text-xs font-semibold border border-border transition cursor-pointer"
        >
          <Key class="w-4 h-4 text-success" />
          <span>Connect OAuth</span>
        </button>

        <button
          type="button"
          onclick={handleTestAll}
          disabled={isPingingAll}
          class="flex items-center gap-1.5 px-3 py-2 rounded-lg bg-surface hover:bg-surface-3 text-text-muted hover:text-text-main font-body text-xs border border-border transition cursor-pointer"
        >
          {#if isPingingAll}
            <Loader2 class="w-4 h-4 animate-spin text-info" />
          {:else}
            <Activity class="w-4 h-4 text-info" />
          {/if}
          <span>Test All Connections</span>
        </button>
      </div>
    </div>

    <!-- Filter & Search Rail (Stitch Screenshot) -->
    <div class="flex flex-col sm:flex-row items-center justify-between gap-3 p-2 rounded-xl bg-surface border border-border">
      <div class="flex items-center gap-1 overflow-x-auto w-full sm:w-auto pb-1 sm:pb-0">
        <button
          type="button"
          onclick={() => (filterType = 'all')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer {filterType === 'all'
            ? 'bg-surface-3 text-text-main shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-surface-3'}"
        >
          <span>All Providers</span>
          <span class="font-code text-[10px] px-1.5 py-0.2 rounded bg-surface-2 text-info">
            {connections.length}
          </span>
        </button>

        <button
          type="button"
          onclick={() => (filterType = 'active')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer {filterType === 'active'
            ? 'bg-surface-3 text-text-main shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-surface-3'}"
        >
          <span>Active</span>
          <span class="font-code text-[10px] px-1.5 py-0.2 rounded bg-success/15 text-success">
            {activeCount}
          </span>
        </button>

        <button
          type="button"
          onclick={() => (filterType = 'oauth')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer {filterType === 'oauth'
            ? 'bg-surface-3 text-text-main shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-surface-3'}"
        >
          <span>OAuth Multiplex</span>
          <span class="font-code text-[10px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">
            {oauthCount}
          </span>
        </button>

        <button
          type="button"
          onclick={() => (filterType = 'custom')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer {filterType === 'custom'
            ? 'bg-surface-3 text-text-main shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-surface-3'}"
        >
          <span>Custom Proxies</span>
          <span class="font-code text-[10px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">
            {customCount}
          </span>
        </button>

        <button
          type="button"
          onclick={() => (filterType = 'free')}
          class="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer {filterType === 'free'
            ? 'bg-surface-3 text-text-main shadow-sm'
            : 'text-text-muted hover:text-text-main hover:bg-surface-3'}"
        >
          <span>Free Tier</span>
          <span class="font-code text-[10px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">
            {freeCount}
          </span>
        </button>
      </div>

      <!-- Search Input with ⌘F -->
      <div class="relative w-full sm:w-80 flex items-center">
        <Search class="absolute left-3 w-4 h-4 text-text-subtle pointer-events-none" />
        <input
          type="text"
          bind:value={search}
          placeholder="Filter by endpoint, key hash, or model..."
          class="w-full bg-surface-2 border border-border rounded-lg pl-9 pr-12 py-1.5 font-body text-xs text-text-main placeholder:text-text-subtle focus:outline-none focus:border-brand-500 transition"
        />
        <span class="absolute right-2 px-1.5 py-0.5 rounded bg-surface-2 font-code text-[10px] text-text-muted pointer-events-none border border-border">
          ⌘F
        </span>
      </div>
    </div>
  </div>

  <!-- SECTION 1: Active OAuth & Multi-Account Pools (Antigravity Expanded Matrix) -->
  <div class="space-y-3">
    <div class="flex items-center justify-between px-1">
      <div class="flex items-center gap-2">
        <Shield class="w-4 h-4 text-info" />
        <h2 class="font-headline text-base font-bold text-text-main">OAuth High-Availability Pools</h2>
      </div>
      <span class="font-code text-[11px] text-text-subtle">Virtual Session Balancing · Auto-Rotating</span>
    </div>

    <!-- Antigravity Master Node Container (Stitch Design) -->
    <div class="rounded-xl bg-surface border border-border overflow-hidden shadow-xl">
      <!-- Antigravity Header Rail -->
      <div class="p-5 bg-surface-2 border-b border-border flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div class="flex items-center gap-4">
          <!-- Antigravity Logo -->
          <div class="w-12 h-12 rounded-xl bg-bg border border-border flex items-center justify-center text-info shadow-inner">
            <svg class="w-7 h-7" fill="currentColor" viewBox="0 0 32 32">
              <path d="M16 2L3 28h26L16 2zm0 6.2l9.1 18H6.9L16 8.2z"></path>
              <circle cx="16" cy="18" r="3"></circle>
            </svg>
          </div>

          <div class="space-y-1">
            <div class="flex items-center gap-2">
              <h3 class="font-headline text-base font-bold text-text-main">Antigravity Multi-Pool</h3>
              <span class="font-code text-[10px] px-2 py-0.5 rounded-full bg-success/15 text-success flex items-center gap-1 font-semibold border border-success/25">
                <span class="w-1.5 h-1.5 rounded-full bg-success animate-ping"></span>
                {antigravityAccounts.filter((a) => a.isActive === 1).length} Connected
              </span>
              <span class="font-code text-[10px] px-2 py-0.5 rounded bg-bg text-info border border-border">
                Round-Robin Active
              </span>
            </div>
            <div class="flex items-center gap-3 text-xs text-text-muted">
              <span>Mean RTT: <strong class="font-code text-success">42ms</strong></span>
              <span>•</span>
              <span>Success Rate: <strong class="font-code text-success">99.82%</strong></span>
              <span>•</span>
              <button
                type="button"
                onclick={() => (isAntigravityModalOpen = true)}
                class="text-info hover:underline flex items-center gap-1 cursor-pointer"
              >
                <span>Manage OAuth credentials</span>
                <ExternalLink class="w-3 h-3" />
              </button>
            </div>
          </div>
        </div>

        <!-- Pool Orchestration Controls -->
        <div class="flex items-center gap-2">
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-bg border border-border">
            <span class="font-body text-xs text-text-muted">Balancing</span>
            <div class="w-2 h-2 rounded-full bg-info animate-pulse"></div>
          </div>

          <button
            type="button"
            onclick={() => handleTestAll()}
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-bg hover:bg-surface-2 text-text-main text-xs font-semibold border border-border transition cursor-pointer"
          >
            <Activity class="w-3.5 h-3.5 text-success" />
            <span>Apply Proxy</span>
          </button>

          <button
            type="button"
            onclick={() => handleTestAll()}
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-bg hover:bg-surface-2 text-text-main text-xs font-semibold border border-border transition cursor-pointer"
          >
            <Zap class="w-3.5 h-3.5 text-brand-500" />
            <span>Test Sequential</span>
          </button>
        </div>
      </div>

      <!-- Alert Banner (Stitch Design) -->
      <div class="p-3 bg-surface border-b border-border flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs">
        <div class="flex items-center gap-2 text-brand-400">
          <AlertTriangle class="w-4 h-4 flex-shrink-0" />
          <span class="font-bold">High Availability Autonomous Routing Active:</span>
          <span class="text-text-muted hidden md:inline">
            Proxy pool operating with proactive token refresh. Auto-failover configured on HTTP 429 & limit. Cooldown set to 90s.
          </span>
        </div>
        <span class="font-code text-[10px] text-brand-400 bg-brand-500/10 px-2 py-0.5 rounded border border-brand-500/20 self-start sm:self-auto">
          Cluster Policy v2.4
        </span>
      </div>

      <!-- Account Credentials Table -->
      <div class="overflow-x-auto">
        <table class="w-full text-left font-body text-xs">
          <thead>
            <tr class="border-b border-border text-text-subtle font-code uppercase text-[10px] tracking-wider bg-surface-2">
              <th class="py-2.5 px-4">Account Credential Identity</th>
              <th class="py-2.5 px-4">Health Status</th>
              <th class="py-2.5 px-4">Routing Slot</th>
              <th class="py-2.5 px-4">Priority</th>
              <th class="py-2.5 px-4 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/50 font-code">
            {#each antigravityAccounts as conn, idx (conn.id)}
              {@const isActive = conn.isActive === 1}
              {@const latency = latencyMap[conn.id]}

              <tr class="hover:bg-surface-2/40 transition">
                <td class="py-3 px-4 text-text-main font-medium flex items-center gap-2.5">
                  <div class="w-2 h-2 rounded-sm bg-brand-500"></div>
                  <span>{conn.name || `account-${idx + 1}@gmail.com`}</span>
                </td>
                <td class="py-3 px-4">
                  <span class="inline-flex items-center gap-1.5 text-[11px] {isActive ? 'text-success' : 'text-text-subtle'}">
                    <span class="w-1.5 h-1.5 rounded-full {isActive ? 'bg-success' : 'bg-[#636c7e]'}"></span>
                    <span>{isActive ? 'Active' : 'Disabled'}</span>
                  </span>
                </td>
                <td class="py-3 px-4 text-info">
                  Slot #{idx + 1}
                </td>
                <td class="py-3 px-4">
                  <span class="px-2 py-0.5 rounded bg-bg border border-border text-[10px]">
                    P{conn.priority || 1}
                  </span>
                </td>
                <td class="py-3 px-4 text-right">
                  <div class="flex items-center justify-end gap-1.5">
                    {#if latency === 'testing'}
                      <Loader2 class="w-3.5 h-3.5 animate-spin text-info" />
                    {:else if typeof latency === 'number'}
                      <span class="text-[10px] text-success font-bold">{latency}ms</span>
                    {:else}
                      <button
                        type="button"
                        onclick={() => handleTestPing(conn)}
                        class="p-1 text-text-subtle hover:text-info transition cursor-pointer"
                        title="Ping"
                      >
                        <Activity class="w-3.5 h-3.5" />
                      </button>
                    {/if}

                    <button
                      type="button"
                      onclick={() => handleToggle(conn)}
                      class="p-1 text-text-subtle hover:text-text-main transition cursor-pointer"
                      title={isActive ? 'Disable' : 'Enable'}
                    >
                      <Power class="w-3.5 h-3.5 {isActive ? 'text-success' : 'text-text-subtle'}" />
                    </button>

                    <button
                      type="button"
                      onclick={() => handleDelete(conn)}
                      class="p-1 text-text-subtle hover:text-hover:text-danger transition cursor-pointer"
                      title="Delete"
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Enroll link & Virtualized Models Bar -->
      <div class="p-4 bg-surface-2 border-t border-border space-y-3">
        <div class="flex items-center justify-between text-xs">
          <button
            type="button"
            onclick={() => (isAntigravityModalOpen = true)}
            class="text-info hover:underline font-semibold flex items-center gap-1 cursor-pointer"
          >
            <Plus class="w-3.5 h-3.5" />
            <span>+ Enroll Additional OAuth Account</span>
          </button>
          <span class="font-code text-[10px] text-text-subtle">
            Cluster load evenly distributed (16.6% slice / slot)
          </span>
        </div>

        <!-- Virtualized Model IDs Chips -->
        <div class="pt-2 border-t border-border/60 space-y-2">
          <div class="flex items-center justify-between text-[10px] font-code text-text-subtle">
            <span class="uppercase tracking-wider">Virtualized Antigravity Model IDs</span>
            <span>CLICK ID TO COPY ROUTE TARGET</span>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2">
            {#each virtualGeminiModels as model (model.id)}
              <div
                role="button"
                tabindex="0"
                onclick={() => copyText(model.id, model.id)}
                onkeydown={(e) => e.key === 'Enter' && copyText(model.id, model.id)}
                class="p-2.5 rounded-lg bg-surface border border-border hover:border-brand-500/50 flex items-center justify-between cursor-pointer transition group"
              >
                <div>
                  <div class="font-code text-xs font-bold text-text-main group-hover:text-brand-500">
                    {model.id}
                  </div>
                  <div class="text-[10px] text-text-muted font-body">{model.name}</div>
                </div>
                {#if copiedId === model.id}
                  <Check class="w-3.5 h-3.5 text-success" />
                {:else}
                  <Copy class="w-3.5 h-3.5 text-text-subtle group-hover:text-text-main" />
                {/if}
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- SECTION 2: Custom Compatible Proxies (OpenAI / Anthropic Spec) -->
  <div class="space-y-3">
    <div class="flex items-center justify-between px-1">
      <div class="flex items-center gap-2">
        <Cpu class="w-4 h-4 text-brand-500" />
        <h2 class="font-headline text-base font-bold text-text-main">
          Custom Compatible Proxies (OpenAI / Anthropic Spec)
        </h2>
      </div>
      <span class="font-code text-[11px] text-text-subtle">{customProxies.length} Configured Endpoints</span>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
      {#each customProxies as conn (conn.id)}
        {@const isActive = conn.isActive === 1}
        {@const latency = latencyMap[conn.id]}

        <div
          class="p-4 rounded-xl bg-surface border border-border hover:border-brand-500/40 transition flex flex-col justify-between gap-3 shadow-md"
        >
          <!-- Page header -->
          <div class="flex items-start justify-between gap-2">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-lg bg-bg border border-border flex items-center justify-center font-bold text-xs font-code text-info">
                {conn.provider.substring(0, 2).toUpperCase()}
              </div>
              <div>
                <div class="font-code text-xs font-bold text-text-main truncate max-w-[140px]">
                  {conn.name || conn.provider}
                </div>
                <div class="font-code text-[10px] text-text-subtle truncate max-w-[140px]">
                  {conn.baseUrl || 'api.default.com'}
                </div>
              </div>
            </div>

            <span class="font-code text-[9px] px-1.5 py-0.2 rounded {isActive
              ? 'bg-success/10 text-success border border-success/20'
              : 'bg-surface-2 text-text-subtle'}">
              {isActive ? 'Connected' : 'Disabled'}
            </span>
          </div>

          <!-- Metrics -->
          <div class="space-y-1 font-code text-[11px] pt-1 border-t border-border/60">
            <div class="flex items-center justify-between text-text-muted">
              <span>Latency:</span>
              {#if latency === 'testing'}
                <Loader2 class="w-3 h-3 animate-spin text-info" />
              {:else if typeof latency === 'number'}
                <span class="text-success font-bold">{latency}ms</span>
              {:else}
                <button
                  type="button"
                  onclick={() => handleTestPing(conn)}
                  class="text-info hover:underline cursor-pointer"
                >
                  Ping
                </button>
              {/if}
            </div>
            <div class="flex items-center justify-between text-text-muted">
              <span>Priority:</span>
              <span class="text-text-main font-semibold">P{conn.priority || 1}</span>
            </div>
          </div>

          <!-- Bottom bar -->
          <div class="flex items-center justify-between pt-1 border-t border-border/60">
            <span class="font-code text-[9px] px-1.5 py-0.5 rounded bg-bg text-info border border-border uppercase">
              {conn.provider.includes('anthropic') ? 'Anthropic API' : 'OpenAI API'}
            </span>

            <div class="flex items-center gap-1">
              <button
                type="button"
                onclick={() => handleToggle(conn)}
                class="p-1 rounded text-text-subtle hover:text-text-main cursor-pointer"
                title={isActive ? 'Disable' : 'Enable'}
              >
                <Power class="w-3 h-3 {isActive ? 'text-success' : 'text-text-subtle'}" />
              </button>
              <button
                type="button"
                onclick={() => handleDelete(conn)}
                class="p-1 rounded text-text-subtle hover:text-hover:text-danger cursor-pointer"
                title="Delete"
              >
                <Trash2 class="w-3 h-3" />
              </button>
            </div>
          </div>
        </div>
      {/each}

      <!-- Add Compatible Proxy Card -->
      <button
        type="button"
        onclick={() => openAddModal('openai')}
        class="p-6 rounded-xl border border-dashed border-border hover:border-brand-500 bg-surface/40 hover:bg-surface flex flex-col items-center justify-center gap-2 transition cursor-pointer text-center group"
      >
        <div class="w-8 h-8 rounded-full bg-brand-500/10 text-brand-500 flex items-center justify-center group-hover:scale-110 transition">
          <Plus class="w-4 h-4" />
        </div>
        <span class="font-headline text-xs font-bold text-text-main">Add Compatible Proxy</span>
        <span class="font-body text-[10px] text-text-subtle">OpenAI / Anthropic REST endpoints</span>
      </button>
    </div>
  </div>

  <!-- SECTION 3: Free Tier & Community Ecosystem -->
  <div class="space-y-3">
    <div class="flex items-center justify-between px-1">
      <div class="flex items-center gap-2">
        <Sparkles class="w-4 h-4 text-success" />
        <h2 class="font-headline text-base font-bold text-text-main">Free Tier & Community Ecosystem</h2>
      </div>
      <button
        type="button"
        onclick={handleTestAll}
        class="font-code text-[11px] text-info hover:underline flex items-center gap-1 cursor-pointer"
      >
        <RefreshCw class="w-3 h-3" />
        <span>Batch Ping Free Endpoints</span>
      </button>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
      <!-- Freebuff -->
      <div class="p-3.5 rounded-xl bg-surface border border-border space-y-2">
        <div class="flex items-center justify-between">
          <span class="w-2 h-2 rounded-full bg-success"></span>
          <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-success/15 text-success">Active Session</span>
        </div>
        <div>
          <div class="font-headline text-xs font-bold text-text-main">Freebuff Free Cluster</div>
          <div class="font-body text-[10px] text-text-muted">Daily Freebucks · Cost Mode Free</div>
        </div>
        <div class="flex items-center justify-between pt-1 border-t border-border/60 font-code text-[10px]">
          <span class="text-text-subtle">Quota: Unlimited</span>
          <button
            type="button"
            onclick={() => {
              isFreebuffModalOpen = true
              startFreebuffFlow()
            }}
            class="text-info hover:underline"
          >
            Pair CLI
          </button>
        </div>
      </div>

      <!-- Gemini CLI -->
      <div class="p-3.5 rounded-xl bg-surface border border-border space-y-2">
        <div class="flex items-center justify-between">
          <span class="w-2 h-2 rounded-full bg-info"></span>
          <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">Direct</span>
        </div>
        <div>
          <div class="font-headline text-xs font-bold text-text-main">Gemini CLI</div>
          <div class="font-body text-[10px] text-text-muted">OAuth Direct Pipeline</div>
        </div>
        <div class="flex items-center justify-between pt-1 border-t border-border/60 font-code text-[10px]">
          <span class="text-text-subtle">RTT: ~34ms</span>
          <span class="text-success">Ready</span>
        </div>
      </div>

      <!-- OpenRouter -->
      <div class="p-3.5 rounded-xl bg-surface border border-border space-y-2">
        <div class="flex items-center justify-between">
          <span class="w-2 h-2 rounded-full bg-brand-500"></span>
          <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">Aggregator</span>
        </div>
        <div>
          <div class="font-headline text-xs font-bold text-text-main">OpenRouter Free</div>
          <div class="font-body text-[10px] text-text-muted">Aggregated Free Models</div>
        </div>
        <div class="flex items-center justify-between pt-1 border-t border-border/60 font-code text-[10px]">
          <span class="text-text-subtle">Credit: Free</span>
          <span class="text-info">Idle</span>
        </div>
      </div>

      <!-- NVIDIA NIM -->
      <div class="p-3.5 rounded-xl bg-surface border border-border space-y-2">
        <div class="flex items-center justify-between">
          <span class="w-2 h-2 rounded-full bg-success"></span>
          <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">Community</span>
        </div>
        <div>
          <div class="font-headline text-xs font-bold text-text-main">NVIDIA NIM</div>
          <div class="font-body text-[10px] text-text-muted">Llama-3-8B Free Pool</div>
        </div>
        <div class="flex items-center justify-between pt-1 border-t border-border/60 font-code text-[10px]">
          <span class="text-text-subtle">Quota: 1k/day</span>
          <span class="text-success">Active</span>
        </div>
      </div>

      <!-- Vertex AI -->
      <div class="p-3.5 rounded-xl bg-surface border border-border space-y-2">
        <div class="flex items-center justify-between">
          <span class="w-2 h-2 rounded-full bg-[#636c7e]"></span>
          <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-surface-2 text-text-muted">GCP</span>
        </div>
        <div>
          <div class="font-headline text-xs font-bold text-text-main">Vertex AI</div>
          <div class="font-body text-[10px] text-text-muted">Cloud ADC Pipeline</div>
        </div>
        <div class="flex items-center justify-between pt-1 border-t border-border/60 font-code text-[10px]">
          <span class="text-text-subtle">ADC Account</span>
          <span class="text-text-muted">Setup</span>
        </div>
      </div>
    </div>
  </div>

  <!-- MODAL: Add Custom Provider (Mac-Style Window from Stitch) -->
  {#if isAddOpen}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4">
      <div class="w-full max-w-lg p-6 rounded-2xl bg-surface-2 border border-border shadow-2xl flex flex-col gap-4">
        <!-- Mac-style Window Top Controls & Title -->
        <div class="flex items-center justify-between pb-2 border-b border-border">
          <div class="flex items-center gap-2">
            <button
              type="button"
              aria-label="Close dialog"
              onclick={() => (isAddOpen = false)}
              class="w-3 h-3 rounded-full bg-[#ff5f56] cursor-pointer"
            ></button>
            <div class="w-3 h-3 rounded-full bg-[#ffbd2e]"></div>
            <div class="w-3 h-3 rounded-full bg-[#27c93f]"></div>
            <span class="ml-2 font-headline text-sm font-bold text-text-main">
              Add {addPreset === 'anthropic' ? 'Anthropic' : 'OpenAI'} Compatible Provider
            </span>
          </div>
        </div>

        <form onsubmit={handleAddSubmit} class="space-y-3 font-body text-xs">
          <div>
            <label for="display-name-input" class="block font-semibold text-text-muted mb-1">Display Name</label>
            <input
              id="display-name-input"
              type="text"
              placeholder="e.g. Anthropic Production Key"
              bind:value={formName}
              required
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-body text-xs text-text-main focus:outline-none focus:border-brand-500"
            />
          </div>

          <div>
            <label for="modal-provider-select" class="block font-semibold text-text-muted mb-1">Provider Protocol</label>
            <select
              id="modal-provider-select"
              bind:value={formProvider}
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-body text-xs text-text-main focus:outline-none focus:border-brand-500"
            >
              <option value="openai">OpenAI Compatible (/v1/chat/completions)</option>
              <option value="anthropic">Anthropic Compatible (/v1/messages)</option>
              <option value="deepseek">DeepSeek Official</option>
              <option value="groq">Groq High-Speed Cloud</option>
              <option value="openrouter">OpenRouter Multi-Model Proxy</option>
              <option value="ollama">Ollama Local Instance</option>
            </select>
          </div>

          <div>
            <label for="modal-base-url-input" class="block font-semibold text-text-muted mb-1">Base API URL</label>
            <input
              id="modal-base-url-input"
              type="text"
              placeholder="https://api.anthropic.com/v1"
              bind:value={formBaseUrl}
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
            />
          </div>

          <div>
            <label for="modal-api-key-input" class="block font-semibold text-text-muted mb-1">API Key Secret *</label>
            <input
              id="modal-api-key-input"
              type="password"
              placeholder="sk-..."
              bind:value={formApiKey}
              required
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
            />
          </div>

          <div>
            <label for="modal-priority-select" class="block font-semibold text-text-muted mb-1">Failover Priority</label>
            <select
              id="modal-priority-select"
              bind:value={formPriority}
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
            >
              <option value={1}>P1 - Primary Active</option>
              <option value={2}>P2 - Secondary</option>
              <option value={3}>P3 - Backup Cascade</option>
            </select>
          </div>

          <div class="flex items-center justify-end gap-2 pt-3 border-t border-border">
            <button
              type="button"
              onclick={() => (isAddOpen = false)}
              class="px-4 py-2 rounded-lg text-text-muted hover:text-text-main cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              class="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-bold shadow-md shadow-brand-500/25 cursor-pointer"
            >
              {#if isSubmitting}
                <Loader2 class="w-3.5 h-3.5 animate-spin" />
              {:else}
                <Check class="w-3.5 h-3.5" />
              {/if}
              <span>Create Provider</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- MODAL: Freebuff Device Flow (Stitch Style) -->
  {#if isFreebuffModalOpen}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4">
      <div class="w-full max-w-md p-6 rounded-2xl bg-surface-2 border border-border shadow-2xl space-y-4">
        <div class="flex items-center justify-between pb-2 border-b border-border">
          <div class="flex items-center gap-2">
            <button
              type="button"
              aria-label="Close dialog"
              onclick={() => (isFreebuffModalOpen = false)}
              class="w-3 h-3 rounded-full bg-[#ff5f56] cursor-pointer"
            ></button>
            <div class="w-3 h-3 rounded-full bg-[#ffbd2e]"></div>
            <div class="w-3 h-3 rounded-full bg-[#27c93f]"></div>
            <span class="ml-2 font-headline text-sm font-bold text-text-main">
              Freebuff CLI Device Pairing
            </span>
          </div>
        </div>

        <div class="space-y-3 font-body text-xs">
          <p class="text-text-muted leading-relaxed">
            Pair with freebuff.com using device flow. A new browser tab has been launched. Log in and allow authorization.
          </p>

          <div class="p-3.5 rounded-xl bg-bg border border-border space-y-1.5 font-code text-xs">
            <div class="text-text-subtle text-[10px] uppercase">Device Pairing Code</div>
            <div class="text-info font-bold select-all tracking-wider">{fbAuthCode || 'Generating...'}</div>
          </div>

          <div class="flex items-center gap-2 p-3 rounded-xl bg-surface text-xs {fbStatus === 'success'
            ? 'text-success bg-success/10 border border-success/20'
            : fbStatus === 'error'
              ? 'text-hover:text-danger bg-danger/10 border border-danger/25'
              : 'text-text-muted'}">
            {#if fbStatus === 'polling'}
              <Loader2 class="w-4 h-4 animate-spin text-info flex-shrink-0" />
            {:else if fbStatus === 'success'}
              <Check class="w-4 h-4 text-success flex-shrink-0" />
            {/if}
            <span class="text-[11px]">{fbMessage}</span>
          </div>

          <div class="flex justify-end pt-2">
            <button
              type="button"
              onclick={() => (isFreebuffModalOpen = false)}
              class="px-4 py-2 rounded-lg bg-surface-3 hover:bg-surface-3 text-text-main font-semibold cursor-pointer"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- MODAL: Antigravity Google OAuth (Stitch Style) -->
  {#if isAntigravityModalOpen}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4">
      <div class="w-full max-w-md p-6 rounded-2xl bg-surface-2 border border-border shadow-2xl space-y-4">
        <div class="flex items-center justify-between pb-2 border-b border-border">
          <div class="flex items-center gap-2">
            <button
              type="button"
              aria-label="Close dialog"
              onclick={() => (isAntigravityModalOpen = false)}
              class="w-3 h-3 rounded-full bg-[#ff5f56] cursor-pointer"
            ></button>
            <div class="w-3 h-3 rounded-full bg-[#ffbd2e]"></div>
            <div class="w-3 h-3 rounded-full bg-[#27c93f]"></div>
            <span class="ml-2 font-headline text-sm font-bold text-text-main">
              Google Antigravity OAuth
            </span>
          </div>
        </div>

        <div class="space-y-3 font-body text-xs">
          <p class="text-text-muted leading-relaxed">
            Authenticate a Google AI account to access Gemini 1.5/2.0 Pro and Flash models under Antigravity multi-account failover.
          </p>

          <a
            href="/api/oauth/antigravity/login"
            class="flex items-center justify-center gap-2 w-full py-2.5 rounded-xl bg-brand-500 hover:bg-brand-600 text-white font-bold shadow-md shadow-brand-500/25 transition cursor-pointer"
          >
            <Globe class="w-4 h-4" />
            <span>Launch Google OAuth Login</span>
          </a>

          <div class="flex justify-end pt-2">
            <button
              type="button"
              onclick={() => (isAntigravityModalOpen = false)}
              class="px-4 py-1.5 rounded-lg text-text-subtle hover:text-text-main cursor-pointer"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>
