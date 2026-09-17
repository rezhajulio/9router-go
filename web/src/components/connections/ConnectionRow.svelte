<script lang="ts">
  import {
    Activity,
    ChevronDown,
    ChevronUp,
    Key,
    Lock,
    Trash2
  } from 'lucide-svelte'
  import type { ProviderConnection } from '../../api/client'
  import Badge from '../../lib/ui/Badge.svelte'
  import Toggle from '../../lib/ui/Toggle.svelte'

  interface Props {
    conn: ProviderConnection
    index: number
    latency?: number | null
    isTestingLatency?: boolean
    strictModelAssignment?: boolean
    availableModels?: Array<{ id: string; name?: string }>
    onTestLatency: (conn: ProviderConnection) => void
    onDelete: (conn: ProviderConnection) => void
    onToggle: (conn: ProviderConnection) => void
    onAssignModel?: (conn: ProviderConnection, modelId: string) => void
  }

  let {
    conn,
    index,
    latency = null,
    isTestingLatency = false,
    strictModelAssignment = false,
    availableModels = [],
    onTestLatency,
    onDelete,
    onToggle,
    onAssignModel,
  }: Props = $props()

  let isActive = $derived(conn.isActive === 1)
  let isOAuth = $derived(conn.authType === 'oauth')
  let displayName = $derived(
    conn.name ||
      conn.displayName ||
      conn.email ||
      (conn as unknown as { displayName?: string; email?: string }).displayName ||
      (isOAuth ? 'OAuth Account' : 'API Key Slot')
  )
  let assignedModel = $derived(
    conn.assignedModel ||
      (conn.providerSpecificData as { assignedModel?: string; freebuffModel?: string } | undefined)?.assignedModel ||
      (conn.providerSpecificData as { assignedModel?: string; freebuffModel?: string } | undefined)?.freebuffModel ||
      (conn as unknown as { freebuffModel?: string }).freebuffModel ||
      ''
  )
</script>
<div class="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3 group hover:bg-black/[0.01] dark:hover:bg-white/[0.01] px-2 rounded-lg transition-colors">
  <div class="flex items-center gap-3 min-w-0 flex-1">
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
        {#if conn.testStatus === 'error' || !!conn.lastError}
          <Badge variant="error" size="sm" dot>error</Badge>
        {:else if isActive}
          <Badge variant="success" size="sm" dot>active</Badge>
        {:else}
          <Badge variant="default" size="sm">disabled</Badge>
        {/if}
        <Badge variant="outline" size="sm">{isOAuth ? 'OAuth' : (conn.authType || 'API Key')}</Badge>
        <span class="text-xs text-text-muted font-mono">#{index + 1}</span>
        {#if latency != null}
          <span class="text-xs font-mono text-emerald-500 font-medium">{latency}ms</span>
        {/if}
      </div>
      <div class="mt-1.5 flex items-center gap-2">
        <select
          value={assignedModel}
          disabled={!strictModelAssignment}
          onchange={(e) => onAssignModel?.(conn, (e.target as HTMLSelectElement).value)}
          class="text-xs rounded-md border border-border bg-surface px-2 py-1 text-text-main focus:outline-none focus:ring-1 focus:ring-brand-500 disabled:opacity-50 disabled:cursor-not-allowed max-w-[220px] truncate"
          title={strictModelAssignment ? 'Assigned Model' : 'Strict Model Assignment disabled'}
        >
          <option value="">Unassigned</option>
          {#each availableModels as model (model.id)}
            <option value={model.id}>{model.name || model.id}</option>
          {/each}
        </select>
        {#if assignedModel && !strictModelAssignment}
          <span class="text-[11px] text-amber-500/80">(strict mode disabled)</span>
        {/if}
      </div>

      {#if conn.lastError}
        <p class="text-xs text-red-500 font-mono mt-0.5 truncate max-w-xl" title={conn.lastError}>
          {conn.lastError}
        </p>
      {/if}
    </div>
  </div>

  <!-- Action Controls -->
  <div class="flex items-center gap-2 self-end sm:self-center shrink-0">
    <button
      type="button"
      onclick={() => onTestLatency(conn)}
      disabled={isTestingLatency}
      title="Test Ping"
      class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-text-main hover:border-brand-500/40 transition-colors cursor-pointer"
    >
      <Activity class="w-3.5 h-3.5 {isTestingLatency ? 'animate-pulse text-brand-500' : ''}" />
    </button>

    <button
      type="button"
      onclick={() => onDelete(conn)}
      title="Delete Connection"
      class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-red-500 hover:border-red-500/40 transition-colors cursor-pointer"
    >
      <Trash2 class="w-3.5 h-3.5" />
    </button>

    <Toggle
      size="sm"
      checked={isActive}
      onChange={() => onToggle(conn)}
    />
  </div>
</div>
