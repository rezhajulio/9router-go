<script lang="ts">
  import {
    AlertTriangle,
    ArrowLeft,
    Check,
    Copy,
    ExternalLink,
    Plus,
    Trash2
  } from 'lucide-svelte'
  import type { ProviderNode } from '../../api/client'
  import type { ProviderCatalogItem } from '../../lib/providers'
  import Badge from '../../lib/ui/Badge.svelte'
  import Button from '../../lib/ui/Button.svelte'
  import Card from '../../lib/ui/Card.svelte'
  import { getIconPath } from './types'

  interface Props {
    selectedProviderId: string
    node?: ProviderNode
    catalogItem?: ProviderCatalogItem
    connectionCount: number
    copiedId?: string | null
    onBack: () => void
    onDeleteNode: (nodeId: string) => void
    onAddConnection: () => void
    onCopyUrl: (url: string) => void
  }

  let {
    selectedProviderId,
    node,
    catalogItem,
    connectionCount,
    copiedId = null,
    onBack,
    onDeleteNode,
    onAddConnection,
    onCopyUrl,
  }: Props = $props()

  let title = $derived(node?.name || catalogItem?.name || selectedProviderId)
  let icon = $derived(getIconPath(selectedProviderId, node?.apiType))
</script>

<div class="flex flex-col gap-6">
  <!-- Top Return Bar -->
  <div class="flex items-center justify-between">
    <button
      type="button"
      onclick={onBack}
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
          {connectionCount} {connectionCount === 1 ? 'connection' : 'connections'}
        </p>
      </div>
    </div>

    <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
      {#if node}
        <Button
          size="sm"
          variant="ghost"
          class="text-red-500 hover:text-red-600 hover:bg-red-500/10"
          onclick={() => onDeleteNode(node.id)}
        >
          <Trash2 class="w-4 h-4 mr-1.5" />
          Delete Provider
        </Button>
      {/if}
      <Button size="sm" variant="primary" onclick={onAddConnection}>
        <Plus class="w-4 h-4 mr-1.5" />
        {node ? 'Add API Key' : 'Add Connection'}
      </Button>
    </div>
  </div>

  <!-- Risk Warning -->
  {#if catalogItem?.category === 'oauth' || selectedProviderId === 'antigravity'}
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
          type="button"
          onclick={() => onCopyUrl(node.baseUrl || '')}
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
</div>
