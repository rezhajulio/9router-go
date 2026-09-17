<script lang="ts">
  import { Search, Server } from 'lucide-svelte'
  import { api, type ProviderConnection } from '../../api/client'
  import { getModelKind, getModelsByProviderId } from '../../lib/models'
  import { getProvidersByKind, type ProviderCatalogItem } from '../../lib/providers'
  import Badge from '../../lib/ui/Badge.svelte'
  import MediaProviderCard from './MediaProviderCard.svelte'
  import MediaProviderDetail from './MediaProviderDetail.svelte'
  import {
    getMediaProviderStats,
    MEDIA_KIND_INFO,
    type MediaKind
  } from './mediaTypes'

  interface Props {
    kind: MediaKind
    connections?: ProviderConnection[]
    onRefresh: () => void
  }

  let { kind, connections = [], onRefresh }: Props = $props()

  let selectedProvider = $state<ProviderCatalogItem | null>(null)
  let searchQuery = $state('')
  let statusFilter = $state<'all' | 'connected' | 'error' | 'disabled' | 'not_connected'>('all')

  let info = $derived(MEDIA_KIND_INFO[kind])
  let providers = $derived(getProvidersByKind(kind))

  function getKindModelCount(providerId: string): number {
    return getModelsByProviderId(providerId).filter((m) => getModelKind(m) === kind).length
  }

  let filteredProviders = $derived(
    providers
      .filter((p) => {
        if (searchQuery.trim()) {
          const q = searchQuery.toLowerCase()
          if (!p.name.toLowerCase().includes(q) && !p.id.toLowerCase().includes(q)) {
            return false
          }
        }
        const stats = getMediaProviderStats(connections, p.id, p.noAuth)
        if (statusFilter === 'all') return true
        if (statusFilter === 'connected') return stats.status === 'connected'
        if (statusFilter === 'error') return stats.status === 'error'
        if (statusFilter === 'disabled') return stats.status === 'disabled'
        if (statusFilter === 'not_connected') return stats.status === 'none'
        return true
      })
      .sort((a, b) => {
        const statsA = getMediaProviderStats(connections, a.id, a.noAuth)
        const statsB = getMediaProviderStats(connections, b.id, b.noAuth)
        const scoreA = statsA.active > 0 ? 2 : statsA.status === 'ready' ? 1 : 0
        const scoreB = statsB.active > 0 ? 2 : statsB.status === 'ready' ? 1 : 0
        return scoreB - scoreA || a.name.localeCompare(b.name)
      })
  )

  let connectedCount = $derived(
    providers.filter((p) => {
      const stats = getMediaProviderStats(connections, p.id, p.noAuth)
      return stats.status === 'connected' || stats.status === 'ready'
    }).length
  )

  async function handleToggleAll(providerId: string, newActive: boolean) {
    const list = connections.filter((c) => c.provider === providerId)
    await Promise.allSettled(
      list.map((c) => api.updateConnection(c.id, { isActive: newActive ? 1 : 0 }))
    )
    onRefresh()
  }
</script>

{#if selectedProvider}
  <MediaProviderDetail
    provider={selectedProvider}
    {kind}
    {connections}
    onBack={() => (selectedProvider = null)}
    {onRefresh}
  />
{:else}
  <div class="flex flex-col gap-6 animate-fade-in">
    <!-- Header banner / summary -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-border">
      <div>
        <h1 class="text-xl font-bold text-text-main flex items-center gap-2.5">
          <span>{info.title}</span>
          <Badge tone="default" size="sm">{providers.length} providers</Badge>
        </h1>
        <p class="text-xs text-text-muted mt-1 leading-relaxed max-w-2xl">
          {info.description}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Badge variant={connectedCount > 0 ? 'success' : 'default'} size="md" dot={connectedCount > 0}>
          {connectedCount} / {providers.length} ready
        </Badge>
      </div>
    </div>

    <!-- Search & Filter Bar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="relative flex-1 max-w-md">
        <Search class="w-4 h-4 text-text-muted absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search {info.singular.toLowerCase()} providers..."
          class="w-full pl-9 pr-4 py-1.5 text-xs rounded-xl bg-surface border border-border text-text-main placeholder:text-text-muted focus:outline-none focus:border-brand-500 transition-colors"
        />
      </div>

      <select
        bind:value={statusFilter}
        class="h-8 rounded-lg border border-border bg-surface px-2 text-xs text-text-main outline-none transition-colors hover:border-brand-500/40 cursor-pointer"
      >
        <option value="all">All Providers</option>
        <option value="connected">Connected</option>
        <option value="error">Error</option>
        <option value="disabled">Disabled</option>
        <option value="not_connected">Not Connected</option>
      </select>
    </div>

    <!-- Providers Grid -->
    {#if filteredProviders.length === 0}
      <div class="flex flex-col items-center justify-center py-16 gap-3 text-text-muted text-xs border border-dashed border-border rounded-xl">
        <Server class="w-8 h-8 opacity-40" />
        <span>{info.emptyMessage}</span>
      </div>
    {:else}
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:gap-4 lg:grid-cols-3 xl:grid-cols-4">
        {#each filteredProviders as provider (provider.id)}
          <MediaProviderCard
            {provider}
            stats={getMediaProviderStats(connections, provider.id, provider.noAuth)}
            modelCount={getKindModelCount(provider.id)}
            onSelect={() => (selectedProvider = provider)}
            onToggleAll={(active) => handleToggleAll(provider.id, active)}
          />
        {/each}
      </div>
    {/if}
  </div>
{/if}
