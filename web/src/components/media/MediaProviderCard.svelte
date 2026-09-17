<script lang="ts">
  import { Layers, PauseCircle } from 'lucide-svelte'
  import type { ProviderCatalogItem } from '../../lib/providers'
  import Badge from '../../lib/ui/Badge.svelte'
  import Card from '../../lib/ui/Card.svelte'
  import Toggle from '../../lib/ui/Toggle.svelte'
  import { getIconPath } from '../connections/types'
  import type { MediaProviderStats } from './mediaTypes'

  interface Props {
    provider: ProviderCatalogItem
    stats: MediaProviderStats
    modelCount: number
    onSelect: () => void
    onToggleAll?: (active: boolean) => void
  }

  let { provider, stats, modelCount, onSelect, onToggleAll }: Props = $props()

  let icon = $derived(getIconPath(provider.id))
  let isAllDisabled = $derived(stats.status === 'disabled')
</script>

<div
  onclick={onSelect}
  onkeydown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onSelect()
    }
  }}
  role="button"
  tabindex="0"
  class="group min-w-0 cursor-pointer text-left focus:outline-none"
>
  <Card
    padding="xs"
    class="h-full hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition-all p-3.5 rounded-xl border {isAllDisabled
      ? 'border-border bg-surface opacity-60'
      : stats.status === 'error'
        ? 'border-red-500/40 bg-red-500/[0.02]'
        : stats.status === 'connected' || stats.status === 'ready'
          ? 'border-emerald-500/35 bg-emerald-500/[0.02]'
          : 'border-border bg-surface hover:border-brand-500/30'}"
  >
    <div class="flex min-w-0 items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-3">
        <div class="w-9 h-9 shrink-0 rounded-lg flex items-center justify-center bg-black/5 dark:bg-white/5 border border-border overflow-hidden">
          <img
            src={icon}
            alt={provider.name}
            class="w-5 h-5 object-contain"
            onerror={(e) => {
              const target = e.currentTarget as HTMLImageElement
              target.style.display = 'none'
            }}
          />
        </div>
        <div class="min-w-0">
          <h3 class="truncate font-semibold text-sm text-text-main group-hover:text-brand-500 transition-colors">
            {provider.name}
          </h3>
          <div class="flex min-w-0 items-center gap-1.5 text-xs flex-wrap mt-0.5">
            {#if isAllDisabled}
              <Badge variant="default" size="sm">
                <span class="flex items-center gap-1">
                  <PauseCircle class="w-3 h-3" />
                  Disabled
                </span>
              </Badge>
            {:else if stats.status === 'error'}
              <Badge variant="error" size="sm" dot>
                {stats.errorCount} Error
              </Badge>
            {:else if stats.status === 'connected'}
              <Badge variant="success" size="sm" dot>
                {stats.active} Connected
              </Badge>
            {:else if stats.status === 'ready'}
              <Badge variant="success" size="sm" dot>Ready</Badge>
            {:else}
              <span class="text-text-muted">No connections</span>
            {/if}

            {#if modelCount > 0}
              <span class="text-[11px] text-text-muted/80 flex items-center gap-0.5">
                • {modelCount} models
              </span>
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
              onToggleAll?.(isAllDisabled)
            }}
            onkeydown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.stopPropagation()
                onToggleAll?.(isAllDisabled)
              }
            }}
            role="button"
            tabindex="0"
          >
            <Toggle size="sm" checked={!isAllDisabled} onChange={() => {}} />
          </div>
        {/if}
      </div>
    </div>
  </Card>
</div>
