<script lang="ts">
  import {
    Activity,
    ArrowDownRight,
    ArrowUpRight,
    ChevronDown,
    ChevronRight,
    Coins,
    Cpu,
    Database,
    Download,
    ExternalLink,
    Filter,
    Layers,
    Maximize2,
    Minus,
    Network,
    Plus,
    Radio,
    RefreshCw,
    Search,
    SlidersHorizontal,
    Sparkles,
    TrendingUp,
    X,
    Zap
  } from 'lucide-svelte'
  import { api, getAuthHeaders, type ProviderConnection, type ProviderNode } from '../api/client'
  import Card from '../lib/ui/Card.svelte'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'

  interface Props {
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
  }

  let { connections = [], providerNodes = [] }: Props = $props()

  type MainTab = 'overview' | 'details'
  type Period = 'today' | '24h' | '7d' | '30d' | '60d'
  type TableView = 'model' | 'account' | 'apiKey' | 'endpoint'
  type ViewMode = 'costs' | 'tokens'

  let activeTab = $state<MainTab>('overview')
  let period = $state<Period>('today')
  let tableView = $state<TableView>('model')
  let viewMode = $state<ViewMode>('costs')
  let isLoading = $state(true)
  let isFetching = $state(false)

  interface StatsData {
    totalRequests?: number
    totalPromptTokens?: number
    totalCompletionTokens?: number
    totalCachedTokens?: number
    totalCost?: number
    byProvider?: Record<string, any>
    byModel?: Record<string, any>
    byAccount?: Record<string, any>
    byApiKey?: Record<string, any>
    byEndpoint?: Record<string, any>
    activeRequests?: any[]
    recentRequests?: any[]
    errorProvider?: string
    pending?: any
  }

  let stats = $state<StatsData>({})
  let expandedGroups = $state<Record<string, boolean>>({})

  // Request details tab state
  let details = $state<any[]>([])
  let detailsTotal = $state(0)
  let detailsPage = $state(1)
  let detailsLoading = $state(false)
  let selectedDetail = $state<any | null>(null)

  const PERIODS: { value: Period; label: string }[] = [
    { value: 'today', label: 'Today' },
    { value: '24h', label: '24h' },
    { value: '7d', label: '7D' },
    { value: '30d', label: '30D' },
    { value: '60d', label: '60D' },
  ]

  const TABLE_OPTIONS: { value: TableView; label: string }[] = [
    { value: 'model', label: 'Usage by Model' },
    { value: 'account', label: 'Usage by Account' },
    { value: 'apiKey', label: 'Usage by API Key' },
    { value: 'endpoint', label: 'Usage by Endpoint' },
  ]

  function fmt(n?: number): string {
    return (n || 0).toLocaleString()
  }

  function fmtCost(n?: number): string {
    return '$' + (n || 0).toFixed(2)
  }

  function timeAgo(timestamp?: string): string {
    if (!timestamp) return 'just now'
    const diff = Math.floor((Date.now() - new Date(timestamp).getTime()) / 1000)
    if (diff < 60) return `${Math.max(1, diff)}s ago`
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
    return `${Math.floor(diff / 86400)}d ago`
  }

  // Load stats when period changes
  async function loadStats(targetPeriod: Period) {
    isFetching = true
    try {
      const res = await api.getUsageStats(targetPeriod)
      if (res) {
        stats = res
      }
    } catch (err) {
      console.error('Failed to load usage stats:', err)
    } finally {
      isLoading = false
      isFetching = false
    }
  }

  // Load request details for Details tab
  async function loadDetails(page = 1) {
    detailsLoading = true
    try {
      const limit = 20
      const offset = (page - 1) * limit
      const res = await api.getRequestDetails(limit, offset)
      if (res && Array.isArray(res.details)) {
        details = res.details
        detailsTotal = res.total || 0
        detailsPage = page
      }
    } catch (err) {
      console.error('Failed to load request details:', err)
    } finally {
      detailsLoading = false
    }
  }

  $effect(() => {
    loadStats(period)
  })

  $effect(() => {
    if (activeTab === 'details') {
      loadDetails(detailsPage)
    }
  })

  // SSE real-time updates for activeRequests and recentRequests
  $effect(() => {
    const es = new EventSource('/api/usage/stream')
    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        if (data.recentRequests) {
          stats.recentRequests = data.recentRequests
        }
        if (data.activeRequests) {
          stats.activeRequests = data.activeRequests
        }
      } catch {}
    }
    return () => es.close()
  })

  function toggleGroup(key: string) {
    expandedGroups[key] = !expandedGroups[key]
  }

  // Grouped table data processing
  let tableData = $derived(() => {
    if (!stats) return []
    let sourceMap: Record<string, any> = {}
    if (tableView === 'model') sourceMap = stats.byModel || {}
    else if (tableView === 'account') sourceMap = stats.byAccount || {}
    else if (tableView === 'apiKey') sourceMap = stats.byApiKey || {}
    else if (tableView === 'endpoint') sourceMap = stats.byEndpoint || {}

    return Object.entries(sourceMap)
      .map(([key, item]) => {
        const totalTokens = (item.promptTokens || 0) + (item.completionTokens || 0)
        return {
          key,
          ...item,
          totalTokens,
        }
      })
      .sort((a, b) => (b.requests || 0) - (a.requests || 0))
  })

  // Unique active provider names for topology graph
  let topologyProviders = $derived(() => {
    const seen = new Set<string>()
    const list: { id: string; name: string; type: string }[] = []

    // From active provider connections
    for (const c of connections) {
      if (c.isActive !== 0 && !seen.has(c.provider)) {
        seen.add(c.provider)
        list.push({ id: c.provider, name: c.name || c.provider, type: 'connection' })
      }
    }

    // From recent usage if not already included
    if (stats.byProvider) {
      for (const prov of Object.keys(stats.byProvider)) {
        if (!seen.has(prov)) {
          seen.add(prov)
          list.push({ id: prov, name: prov, type: 'active' })
        }
      }
    }

    return list.slice(0, 12)
  })
</script>

<div class="flex min-w-0 flex-col gap-6 px-1 sm:px-0">
  <!-- Tabs + Period Selector Row -->
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <!-- Left: Overview | Details Tabs -->
    <div class="inline-flex rounded-xl bg-surface border border-border p-1 shadow-sm">
      <button
        type="button"
        onclick={() => (activeTab = 'overview')}
        class="rounded-lg px-4 py-1.5 text-xs sm:text-sm font-medium transition-colors cursor-pointer {activeTab === 'overview'
          ? 'bg-brand-500 text-white font-semibold shadow-sm'
          : 'text-text-muted hover:text-text-main'}"
      >
        Overview
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'details')}
        class="rounded-lg px-4 py-1.5 text-xs sm:text-sm font-medium transition-colors cursor-pointer {activeTab === 'details'
          ? 'bg-brand-500 text-white font-semibold shadow-sm'
          : 'text-text-muted hover:text-text-main'}"
      >
        Details
      </button>
    </div>

    <!-- Right: Period Selector (when in Overview) -->
    {#if activeTab === 'overview'}
      <div class="flex items-center gap-1.5 self-start sm:self-auto">
        <div class="inline-flex rounded-xl bg-surface border border-border p-1 shadow-sm">
          {#each PERIODS as p}
            <button
              type="button"
              onclick={() => (period = p.value)}
              disabled={isFetching}
              class="rounded-lg px-3 py-1 text-xs sm:text-sm font-medium transition-colors cursor-pointer {period === p.value
                ? 'bg-brand-500 text-white font-semibold shadow-sm'
                : 'text-text-muted hover:text-text-main'}"
            >
              {p.label}
            </button>
          {/each}
        </div>
        {#if isFetching}
          <span class="w-2 h-2 rounded-full bg-brand-500 animate-ping"></span>
        {/if}
      </div>
    {/if}
  </div>

  {#if activeTab === 'overview'}
    <!-- 5 Overview KPI Cards (Exact Upstream Match) -->
    <div class="grid min-w-0 grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 sm:gap-4">
      <!-- 1. Total Requests -->
      <Card padding="md" class="flex min-w-0 flex-col gap-1.5 justify-between">
        <span class="text-text-muted text-[11px] uppercase font-bold tracking-wider">Total Requests</span>
        <span class="truncate font-headline text-2xl sm:text-3xl font-bold text-text-main">
          {fmt(stats.totalRequests)}
        </span>
      </Card>

      <!-- 2. Total Input Tokens (Terracotta Brand) -->
      <Card padding="md" class="flex min-w-0 flex-col gap-1.5 justify-between">
        <span class="text-text-muted text-[11px] uppercase font-bold tracking-wider">Total Input Tokens</span>
        <span class="truncate font-headline text-2xl sm:text-3xl font-bold text-brand-500">
          {fmt(stats.totalPromptTokens)}
        </span>
      </Card>

      <!-- 3. Cached Tokens (Info Blue) -->
      <Card padding="md" class="flex min-w-0 flex-col gap-1.5 justify-between">
        <span class="text-text-muted text-[11px] uppercase font-bold tracking-wider">Cached Tokens</span>
        <span class="truncate font-headline text-2xl sm:text-3xl font-bold text-info">
          {fmt(stats.totalCachedTokens)}
        </span>
      </Card>

      <!-- 4. Output Tokens (Success Green) -->
      <Card padding="md" class="flex min-w-0 flex-col gap-1.5 justify-between">
        <span class="text-text-muted text-[11px] uppercase font-bold tracking-wider">Output Tokens</span>
        <span class="truncate font-headline text-2xl sm:text-3xl font-bold text-success">
          {fmt(stats.totalCompletionTokens)}
        </span>
      </Card>

      <!-- 5. Est. Cost (Warning Yellow) -->
      <Card padding="md" class="flex min-w-0 flex-col gap-1 justify-between">
        <span class="text-text-muted text-[11px] uppercase font-bold tracking-wider">Est. Cost</span>
        <span class="truncate font-headline text-2xl sm:text-3xl font-bold text-warning">
          ~{fmtCost(stats.totalCost)}
        </span>
        <span class="text-[10px] text-text-muted truncate">Estimated, not actual billing</span>
      </Card>
    </div>

    <!-- Provider Topology (2 cols) + Recent Requests (1 col) -->
    <div class="grid min-w-0 grid-cols-1 items-stretch gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(300px,1fr)]">
      <!-- Interactive Provider Topology Card -->
      <Card padding="md" class="flex min-w-0 flex-col justify-between overflow-hidden relative" style="min-height: 480px">
        <!-- Topology Header -->
        <div class="flex items-center justify-between pb-3 border-b border-border/80 z-10">
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-success"></span>
            <span class="font-headline text-xs font-bold text-text-main uppercase tracking-wider">Provider Topology</span>
            <span class="font-code text-[11px] text-text-muted">• {topologyProviders().length} Nodes Active</span>
          </div>
          <span class="font-code text-[10px] text-info bg-surface-2 px-2 py-0.5 rounded border border-border">
            Interactive Mesh
          </span>
        </div>

        <!-- Visual Node Graph Canvas -->
        <div class="relative w-full flex-1 flex items-center justify-center my-4 overflow-hidden select-none">
          <!-- Hub center node -->
          <div class="z-20 p-3 sm:p-4 rounded-xl bg-surface border-2 border-brand-500 shadow-xl shadow-brand-500/20 text-center flex flex-col items-center">
            <div class="flex items-center gap-1.5">
              <span class="w-2 h-2 rounded-full bg-brand-500 animate-ping"></span>
              <span class="font-headline text-xs sm:text-sm font-bold text-text-main">9Router</span>
            </div>
            <span class="font-code text-[10px] text-text-muted pt-0.5">:20130 Gateway</span>
          </div>

          <!-- Connecting Curved Lines SVG -->
          <svg class="absolute inset-0 w-full h-full pointer-events-none stroke-border/70 dark:stroke-white/10" viewBox="0 0 600 360">
            <line x1="300" y1="180" x2="160" y2="70" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="440" y2="70" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="110" y2="180" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="490" y2="180" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="160" y2="290" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="440" y2="290" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="300" y2="50" stroke-width="1.5" stroke-dasharray="4 2" />
            <line x1="300" y1="180" x2="300" y2="310" stroke-width="1.5" stroke-dasharray="4 2" />
          </svg>

          <!-- Floating Provider Nodes -->
          <div class="absolute top-4 left-1/2 -translate-x-1/2 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
            <span>NVIDIA NIM</span>
          </div>

          <div class="absolute top-10 left-8 sm:left-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
            <span>OpenCode Free</span>
          </div>

          <div class="absolute top-10 right-8 sm:right-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
            <span>Antigravity</span>
          </div>

          <div class="absolute top-1/2 -translate-y-1/2 left-2 sm:left-6 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-warning"></span>
            <span>MiMo Code</span>
          </div>

          <div class="absolute top-1/2 -translate-y-1/2 right-2 sm:right-6 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
            <span>OpenRouter</span>
          </div>

          <div class="absolute bottom-10 left-8 sm:left-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-brand-500"></span>
            <span>Grok CLI</span>
          </div>

          <div class="absolute bottom-10 right-8 sm:right-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
            <span>ClinePass</span>
          </div>

          <div class="absolute bottom-4 left-1/2 -translate-x-1/2 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
            <span>inference.dahl.global</span>
          </div>
        </div>

        <!-- Topology Canvas Controls -->
        <div class="flex items-center justify-between pt-2 border-t border-border/80 text-[10px] font-code text-text-muted z-10">
          <div class="flex items-center gap-3">
            <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-success"></span> Active</span>
            <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-info"></span> Standby</span>
            <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-brand-500"></span> Fallback</span>
          </div>
          <div class="flex items-center gap-1">
            <button type="button" class="p-1 rounded hover:bg-surface-2 border border-border"><Plus class="w-3 h-3" /></button>
            <button type="button" class="p-1 rounded hover:bg-surface-2 border border-border"><Minus class="w-3 h-3" /></button>
            <button type="button" class="p-1 rounded hover:bg-surface-2 border border-border"><Maximize2 class="w-3 h-3" /></button>
          </div>
        </div>
      </Card>

      <!-- Recent Requests Card (Height 480px) -->
      <Card padding="sm" class="flex min-w-0 flex-col overflow-hidden" style="height: 480px">
        <!-- Header -->
        <div class="px-3 py-2 border-b border-border shrink-0 flex items-center justify-between">
          <span class="text-xs font-semibold text-text-muted uppercase tracking-wide">Recent Requests</span>
          <span class="font-code text-[10px] px-1.5 py-0.5 rounded bg-success/15 text-success font-bold">
            LIVE
          </span>
        </div>

        {#if !stats.recentRequests || stats.recentRequests.length === 0}
          <div class="flex-1 flex items-center justify-center text-text-muted text-xs">
            No requests recorded yet.
          </div>
        {:else}
          <div class="flex-1 overflow-y-auto">
            <table class="w-full min-w-[280px] border-collapse text-xs">
              <thead class="sticky top-0 bg-surface z-10 border-b border-border">
                <tr>
                  <th class="py-1.5 pl-3 text-left font-semibold text-text-muted w-2"></th>
                  <th class="py-1.5 text-left font-semibold text-text-muted">Model</th>
                  <th class="py-1.5 text-right font-semibold text-text-muted whitespace-nowrap">In / Out</th>
                  <th class="py-1.5 pr-3 text-right font-semibold text-text-muted">When</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border/50 font-code text-[11px]">
                {#each stats.recentRequests as req, i}
                  <tr class="hover:bg-surface-2 transition-colors">
                    <td class="py-2 pl-3">
                      <span class="block w-1.5 h-1.5 rounded-full {req.status === 'ok' || req.status === 'success' ? 'bg-success' : 'bg-error'}"></span>
                    </td>
                    <td class="py-2 pr-2 font-mono truncate max-w-[130px]" title={req.model}>
                      <div class="truncate text-text-main font-semibold">{req.model}</div>
                      <div class="text-[9px] text-text-muted truncate">{req.provider}</div>
                    </td>
                    <td class="py-2 text-right whitespace-nowrap">
                      <span class="text-brand-500">{fmt(req.promptTokens)}↑</span>
                      <span class="text-success">{fmt(req.completionTokens)}↓</span>
                    </td>
                    <td class="py-2 pr-3 text-right text-text-muted whitespace-nowrap text-[10px]">
                      {timeAgo(req.timestamp)}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </Card>
    </div>

    <!-- Breakdown Table Controls: Dropdown + Costs/Tokens Toggle -->
    <div class="flex flex-col gap-3 pt-2">
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <!-- View selector dropdown -->
        <select
          bind:value={tableView}
          class="w-full sm:w-auto rounded-lg border border-border bg-surface px-3 py-1.5 text-sm font-semibold text-text-main focus:outline-none focus:ring-2 focus:ring-brand-500/50 cursor-pointer"
        >
          {#each TABLE_OPTIONS as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>

        <!-- Toggle: Costs | Tokens -->
        <div class="inline-flex rounded-xl bg-surface border border-border p-1 shadow-sm self-start sm:self-auto">
          <button
            type="button"
            onclick={() => (viewMode = 'costs')}
            class="rounded-lg px-3 py-1 text-xs font-semibold transition-colors cursor-pointer {viewMode === 'costs'
              ? 'bg-brand-500 text-white shadow-sm'
              : 'text-text-muted hover:text-text-main'}"
          >
            Costs
          </button>
          <button
            type="button"
            onclick={() => (viewMode = 'tokens')}
            class="rounded-lg px-3 py-1 text-xs font-semibold transition-colors cursor-pointer {viewMode === 'tokens'
              ? 'bg-brand-500 text-white shadow-sm'
              : 'text-text-muted hover:text-text-main'}"
          >
            Tokens
          </button>
        </div>
      </div>

      <!-- Breakdown Table Card -->
      <Card padding="none" class="overflow-hidden border border-border">
        {#if tableData().length === 0}
          <div class="p-8 text-center text-text-muted text-sm font-body">
            No usage recorded for this period.
          </div>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-left border-collapse text-xs font-body">
              <thead class="bg-surface-2 border-b border-border text-text-muted uppercase text-[10px] font-semibold tracking-wider">
                <tr>
                  <th class="py-3 px-4">
                    {tableView === 'model' ? 'Model' : tableView === 'account' ? 'Account' : tableView === 'apiKey' ? 'Key Name' : 'Endpoint'}
                  </th>
                  <th class="py-3 px-4">Provider</th>
                  <th class="py-3 px-4 text-right">Requests</th>
                  {#if viewMode === 'costs'}
                    <th class="py-3 px-4 text-right">Total Cost</th>
                  {:else}
                    <th class="py-3 px-4 text-right">In Tokens</th>
                    <th class="py-3 px-4 text-right">Out Tokens</th>
                    <th class="py-3 px-4 text-right">Cached Tokens</th>
                  {/if}
                  <th class="py-3 px-4 text-right">Last Used</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border/60">
                {#each tableData() as row}
                  <tr class="hover:bg-surface-2/60 transition-colors">
                    <td class="py-3 px-4 font-mono font-medium text-text-main text-xs">
                      {row.rawModel || row.accountName || row.keyName || row.endpoint || row.key}
                    </td>
                    <td class="py-3 px-4">
                      <Badge variant="neutral" size="sm">{row.provider || 'unknown'}</Badge>
                    </td>
                    <td class="py-3 px-4 text-right font-mono font-semibold text-text-main">
                      {fmt(row.requests)}
                    </td>
                    {#if viewMode === 'costs'}
                      <td class="py-3 px-4 text-right font-mono font-bold text-warning">
                        {fmtCost(row.cost)}
                      </td>
                    {:else}
                      <td class="py-3 px-4 text-right font-mono text-brand-500">
                        {fmt(row.promptTokens)}
                      </td>
                      <td class="py-3 px-4 text-right font-mono text-success">
                        {fmt(row.completionTokens)}
                      </td>
                      <td class="py-3 px-4 text-right font-mono text-info">
                        {fmt(row.cachedTokens)}
                      </td>
                    {/if}
                    <td class="py-3 px-4 text-right text-text-muted whitespace-nowrap text-[11px]">
                      {timeAgo(row.lastUsed)}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </Card>
    </div>

  {:else}
    <!-- Details Tab (Request Logger & Inspector) -->
    <Card padding="none" class="overflow-hidden border border-border">
      <div class="px-5 py-3 border-b border-border flex items-center justify-between bg-surface-2">
        <div>
          <h2 class="font-headline text-sm font-bold text-text-main">Recent Request Details</h2>
          <p class="font-body text-xs text-text-muted">Total recorded: {detailsTotal.toLocaleString()} requests</p>
        </div>
        <Button variant="secondary" size="sm" onclick={() => loadDetails(detailsPage)} disabled={detailsLoading}>
          <RefreshCw class="w-3.5 h-3.5 mr-1 {detailsLoading ? 'animate-spin' : ''}" />
          Refresh
        </Button>
      </div>

      {#if detailsLoading}
        <div class="p-12 text-center text-text-muted text-sm flex items-center justify-center gap-2">
          <span class="w-4 h-4 border-2 border-brand-500 border-t-transparent rounded-full animate-spin"></span>
          <span>Loading request history...</span>
        </div>
      {:else if details.length === 0}
        <div class="p-12 text-center text-text-muted text-sm">
          No request logs found in the database.
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse text-xs font-body">
            <thead class="bg-surface-2 border-b border-border text-text-muted uppercase text-[10px] font-semibold tracking-wider">
              <tr>
                <th class="py-3 px-4 w-2"></th>
                <th class="py-3 px-4">Time</th>
                <th class="py-3 px-4">Provider</th>
                <th class="py-3 px-4">Model</th>
                <th class="py-3 px-4 text-right">TTFT</th>
                <th class="py-3 px-4 text-right">Total Latency</th>
                <th class="py-3 px-4 text-right">Prompt</th>
                <th class="py-3 px-4 text-right">Completion</th>
                <th class="py-3 px-4 text-right">Cached</th>
                <th class="py-3 px-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border/60 font-code text-[11px]">
              {#each details as item}
                <tr class="hover:bg-surface-2 transition-colors cursor-pointer" onclick={() => (selectedDetail = item)}>
                  <td class="py-3 px-4">
                    <span class="block w-2 h-2 rounded-full {item.status === 'success' || item.status === 'ok' ? 'bg-success' : 'bg-error'}"></span>
                  </td>
                  <td class="py-3 px-4 text-text-muted whitespace-nowrap text-[11px]">
                    {timeAgo(item.timestamp)}
                  </td>
                  <td class="py-3 px-4">
                    <Badge variant="neutral" size="sm">{item.provider}</Badge>
                  </td>
                  <td class="py-3 px-4 font-bold text-text-main max-w-[140px] truncate">
                    {item.model}
                  </td>
                  <td class="py-3 px-4 text-right text-text-muted">
                    {item.latency?.ttft ? `${item.latency.ttft}ms` : '—'}
                  </td>
                  <td class="py-3 px-4 text-right text-text-main font-medium">
                    {item.latency?.total ? `${item.latency.total}ms` : '—'}
                  </td>
                  <td class="py-3 px-4 text-right text-brand-500">
                    {fmt(item.tokens?.prompt_tokens)}
                  </td>
                  <td class="py-3 px-4 text-right text-success">
                    {fmt(item.tokens?.completion_tokens)}
                  </td>
                  <td class="py-3 px-4 text-right text-info">
                    {fmt(item.tokens?.cached_tokens)}
                  </td>
                  <td class="py-3 px-4 text-right">
                    <button
                      type="button"
                      onclick={(e) => {
                        e.stopPropagation()
                        selectedDetail = item
                      }}
                      class="text-xs text-brand-500 hover:underline font-semibold"
                    >
                      View
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Pagination Controls -->
        <div class="px-4 py-3 border-t border-border flex items-center justify-between text-xs text-text-muted bg-surface-2">
          <span>Showing page {detailsPage} of {Math.ceil(detailsTotal / 20) || 1}</span>
          <div class="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={detailsPage <= 1}
              onclick={() => loadDetails(detailsPage - 1)}
            >
              Previous
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={detailsPage * 20 >= detailsTotal}
              onclick={() => loadDetails(detailsPage + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      {/if}
    </Card>
  {/if}
</div>

<!-- Slide-over Request Inspector Modal -->
{#if selectedDetail}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
    <div class="w-full max-w-2xl max-h-[85vh] rounded-2xl bg-surface border border-border shadow-2xl flex flex-col overflow-hidden">
      <!-- Modal Header -->
      <div class="px-6 py-4 border-b border-border flex items-center justify-between bg-surface-2">
        <div class="flex items-center gap-2">
          <span class="w-2.5 h-2.5 rounded-full {selectedDetail.status === 'success' ? 'bg-success' : 'bg-error'}"></span>
          <h3 class="font-headline text-base font-bold text-text-main">{selectedDetail.model}</h3>
          <Badge variant="neutral" size="sm">{selectedDetail.provider}</Badge>
        </div>
        <button
          type="button"
          onclick={() => (selectedDetail = null)}
          class="p-1 rounded-lg text-text-muted hover:text-text-main hover:bg-surface-3 transition-colors cursor-pointer"
        >
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Modal Body -->
      <div class="flex-1 overflow-y-auto p-6 space-y-4 font-body text-xs">
        <!-- Metadata Row -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div class="p-3 rounded-lg bg-surface-2 border border-border">
            <div class="text-text-muted text-[10px] uppercase font-bold">Latency</div>
            <div class="font-code text-sm font-bold text-text-main mt-1">
              {selectedDetail.latency?.total || 0}ms
            </div>
          </div>
          <div class="p-3 rounded-lg bg-surface-2 border border-border">
            <div class="text-text-muted text-[10px] uppercase font-bold">TTFT</div>
            <div class="font-code text-sm font-bold text-text-main mt-1">
              {selectedDetail.latency?.ttft || 0}ms
            </div>
          </div>
          <div class="p-3 rounded-lg bg-surface-2 border border-border">
            <div class="text-text-muted text-[10px] uppercase font-bold">Input Tokens</div>
            <div class="font-code text-sm font-bold text-brand-500 mt-1">
              {fmt(selectedDetail.tokens?.prompt_tokens)}
            </div>
          </div>
          <div class="p-3 rounded-lg bg-surface-2 border border-border">
            <div class="text-text-muted text-[10px] uppercase font-bold">Output Tokens</div>
            <div class="font-code text-sm font-bold text-success mt-1">
              {fmt(selectedDetail.tokens?.completion_tokens)}
            </div>
          </div>
        </div>

        <!-- Raw JSON details inspector -->
        <div class="space-y-1.5">
          <span class="font-semibold text-text-main uppercase text-[10px] tracking-wider">Payload Payload</span>
          <pre class="p-4 rounded-xl bg-bg border border-border font-code text-[11px] text-text-main overflow-x-auto max-h-80 leading-relaxed">
{JSON.stringify(selectedDetail, null, 2)}
          </pre>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="px-6 py-3 border-t border-border bg-surface-2 flex justify-end">
        <Button variant="secondary" size="sm" onclick={() => (selectedDetail = null)}>
          Close
        </Button>
      </div>
    </div>
  </div>
{/if}
