<script lang="ts">
  import { Activity, ArrowLeft, Clock, Search, Server } from 'lucide-svelte'
  import { api, type ProviderConnection } from '../../api/client'
  import { getModelKind, getModelsByProviderId } from '../../lib/models'
  import type { ProviderCatalogItem } from '../../lib/providers'
  import Badge from '../../lib/ui/Badge.svelte'
  import Button from '../../lib/ui/Button.svelte'
  import Card from '../../lib/ui/Card.svelte'
  import { getIconPath } from '../connections/types'
  import MediaConnectionCard from './MediaConnectionCard.svelte'
  import MediaModelCard from './MediaModelCard.svelte'
  import { getMediaProviderStats, type MediaKind } from './mediaTypes'

  interface Props {
    provider: ProviderCatalogItem
    kind: MediaKind
    connections: ProviderConnection[]
    onBack: () => void
    onRefresh: () => void
  }

  let { provider, kind, connections = [], onBack, onRefresh }: Props = $props()

  let searchQuery = $state('')
  let copiedModelId = $state<string | null>(null)
  let testingModelId = $state<string | null>(null)
  let modelTestResults = $state<Record<string, { ok: boolean; error?: string; latency?: number }>>({})
  let latencyTestingConnId = $state<string | null>(null)
  let connLatencies = $state<Record<string, number | null>>({})

  let stats = $derived(getMediaProviderStats(connections, provider.id, provider.noAuth))
  let providerConns = $derived(connections.filter((c) => c.provider === provider.id))
  let icon = $derived(getIconPath(provider.id))

  let rawModels = $derived(getModelsByProviderId(provider.id))
  let mediaModels = $derived(rawModels.filter((m) => getModelKind(m) === kind))
  let filteredModels = $derived(
    mediaModels.filter((m) => {
      if (!searchQuery.trim()) return true
      const q = searchQuery.toLowerCase()
      return m.id.toLowerCase().includes(q) || (m.name && m.name.toLowerCase().includes(q))
    })
  )

  async function handleToggleConn(conn: ProviderConnection) {
    try {
      await api.updateConnection(conn.id, { isActive: conn.isActive === 1 ? 0 : 1 })
      onRefresh()
    } catch (err) {
      alert(`Toggle failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleToggleAll(newActive: boolean) {
    await Promise.allSettled(
      providerConns.map((c) => api.updateConnection(c.id, { isActive: newActive ? 1 : 0 }))
    )
    onRefresh()
  }

  async function handleTestLatency(conn: ProviderConnection) {
    latencyTestingConnId = conn.id
    const start = performance.now()
    try {
      const first = mediaModels[0]
      const target = first ? `${provider.alias || provider.id}/${first.id}` : `${provider.id}/default`
      await api.testModel(target)
      connLatencies[conn.id] = Math.round(performance.now() - start)
    } catch {
      connLatencies[conn.id] = Math.round(performance.now() - start)
    } finally {
      latencyTestingConnId = null
    }
  }

  async function handleTestModel(modelId: string) {
    testingModelId = modelId
    const full = `${provider.alias || provider.id}/${modelId}`
    const start = performance.now()
    try {
      const res = await api.testModel(full)
      modelTestResults[modelId] = { ok: res.ok, error: res.error, latency: Math.round(performance.now() - start) }
    } catch (err) {
      modelTestResults[modelId] = { ok: false, error: err instanceof Error ? err.message : 'Error', latency: Math.round(performance.now() - start) }
    } finally {
      testingModelId = null
    }
  }

  function handleCopyModel(modelId: string) {
    const full = `${provider.alias || provider.id}/${modelId}`
    navigator.clipboard.writeText(full)
    copiedModelId = modelId
    setTimeout(() => {
      if (copiedModelId === modelId) copiedModelId = null
    }, 2000)
  }
</script>

<div class="flex flex-col gap-6 animate-fade-in">
  <div>
    <button
      type="button"
      onclick={onBack}
      class="inline-flex items-center gap-2 text-xs font-medium text-text-muted hover:text-text-main transition-colors cursor-pointer"
    >
      <ArrowLeft class="w-4 h-4" />
      <span>Back to {kind.toUpperCase()} Providers</span>
    </button>
  </div>

  <Card class="p-6">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="w-12 h-12 rounded-xl flex items-center justify-center bg-black/5 dark:bg-white/5 border border-border shrink-0 overflow-hidden">
          <img
            src={icon}
            alt={provider.name}
            class="w-8 h-8 object-contain"
            onerror={(e) => {
              const target = e.currentTarget as HTMLImageElement
              target.style.display = 'none'
            }}
          />
        </div>
        <div>
          <div class="flex items-center gap-2.5 flex-wrap">
            <h1 class="text-xl font-bold text-text-main">{provider.name}</h1>
            <Badge variant="default" size="sm">alias: {provider.alias || provider.id}</Badge>
            {#if stats.status === 'connected'}
              <Badge variant="success" size="sm" dot>{stats.active} Connected</Badge>
            {:else if stats.status === 'ready'}
              <Badge variant="success" size="sm" dot>Ready</Badge>
            {:else if stats.status === 'error'}
              <Badge variant="error" size="sm" dot>{stats.errorCount} Error</Badge>
            {:else}
              <Badge variant="default" size="sm">No connections</Badge>
            {/if}
          </div>
          <p class="text-xs text-text-muted mt-1">
            Category: <span class="capitalize">{provider.category}</span> • {mediaModels.length} {kind} models available
          </p>
        </div>
      </div>

      {#if stats.total > 0}
        <div class="flex items-center gap-3">
          <Button
            size="sm"
            variant="outline"
            class="text-xs"
            onclick={() => handleToggleAll(stats.active === 0)}
          >
            {stats.active === 0 ? 'Enable All Connections' : 'Disable All'}
          </Button>
        </div>
      {/if}
    </div>
  </Card>

  <!-- Active Connections Section -->
  <div class="flex flex-col gap-3">
    <h2 class="text-sm font-semibold uppercase tracking-wider text-text-muted flex items-center gap-2">
      <Server class="w-4 h-4 text-brand-500" />
      Connections ({providerConns.length})
    </h2>

    {#if providerConns.length === 0}
      <Card class="p-6 text-center text-text-muted text-xs border-dashed">
        <p>No connections configured for {provider.name} yet.</p>
        <p class="mt-1 text-text-muted/70">Connections added in Providers & Endpoints will appear here.</p>
      </Card>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        {#each providerConns as conn (conn.id)}
          <MediaConnectionCard
            {conn}
            latency={connLatencies[conn.id]}
            isTesting={latencyTestingConnId === conn.id}
            onTestLatency={() => handleTestLatency(conn)}
            onToggle={() => handleToggleConn(conn)}
          />
        {/each}
      </div>
    {/if}
  </div>

  <!-- Models Section -->
  <div class="flex flex-col gap-3">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <h2 class="text-sm font-semibold uppercase tracking-wider text-text-muted flex items-center gap-2">
        <Activity class="w-4 h-4 text-brand-500" />
        {kind.toUpperCase()} Models ({mediaModels.length})
      </h2>
      <div class="relative w-full sm:w-64">
        <Search class="w-3.5 h-3.5 text-text-muted absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Filter models..."
          class="w-full pl-8 pr-3 py-1 text-xs rounded-lg bg-surface border border-border text-text-main placeholder:text-text-muted focus:outline-none focus:border-brand-500"
        />
      </div>
    </div>

    {#if filteredModels.length === 0}
      <Card class="p-8 text-center text-text-muted text-xs border-dashed">
        <p>No {kind} models found{searchQuery ? ` matching "${searchQuery}"` : ''}.</p>
      </Card>
    {:else}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {#each filteredModels as model (model.id)}
          <MediaModelCard
            {model}
            {kind}
            isCopied={copiedModelId === model.id}
            isTesting={testingModelId === model.id}
            testResult={modelTestResults[model.id]}
            onCopy={() => handleCopyModel(model.id)}
            onTest={() => handleTestModel(model.id)}
          />
        {/each}
      </div>
    {/if}
  </div>
</div>
