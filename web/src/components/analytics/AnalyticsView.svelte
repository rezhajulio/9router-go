<script lang="ts">
  import { api, type ProviderConnection, type ProviderNode } from '../../api/client'
  import Card from '../../lib/ui/Card.svelte'
  import {
    fmt,
    timeAgo,
    PERIODS,
    type MainTab,
    type Period,
    type StatsData,
    type RequestDetailItem
  } from './types'
  import SummaryKpiCards from './SummaryKpiCards.svelte'
  import UsageBreakdownTable from './UsageBreakdownTable.svelte'
  import RequestDetailsTab from './RequestDetailsTab.svelte'
  import ProviderTopologyCard from './ProviderTopologyCard.svelte'

  interface Props {
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
  }

  let { connections = [] }: Props = $props()

  let activeTab = $state<MainTab>('overview')
  let period = $state<Period>('today')
  let isFetching = $state(false)
  let stats = $state<StatsData>({})

  // Request details tab state
  let details = $state<RequestDetailItem[]>([])
  let detailsTotal = $state(0)
  let detailsPage = $state(1)
  let detailsLoading = $state(false)

  async function loadStats(targetPeriod: Period) {
    isFetching = true
    try {
      const res = await api.getUsageStats(targetPeriod)
      if (res) stats = res
    } catch (err) {
      console.error('Failed to load usage stats:', err)
    } finally {
      isFetching = false
    }
  }

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

  let topologyProviders = $derived(() => {
    const seen = new Set<string>()
    const list: { id: string; name: string; type: string }[] = []

    for (const c of connections) {
      if (c.isActive !== 0 && !seen.has(c.provider)) {
        seen.add(c.provider)
        list.push({ id: c.provider, name: c.name || c.provider, type: 'connection' })
      }
    }

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
    <!-- 5 Overview KPI Cards -->
    <SummaryKpiCards {stats} />

    <!-- Topology + Recent Requests -->
    <div class="grid min-w-0 grid-cols-1 items-stretch gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(300px,1fr)]">
      <ProviderTopologyCard activeNodeCount={topologyProviders().length} />

      <!-- Recent Requests Card -->
      <Card padding="sm" class="flex min-w-0 flex-col overflow-hidden" style="height: 480px">
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
                {#each stats.recentRequests as req}
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

    <!-- Breakdown Table -->
    <UsageBreakdownTable {stats} />
  {:else}
    <RequestDetailsTab
      {details}
      {detailsTotal}
      {detailsPage}
      {detailsLoading}
      onPageChange={loadDetails}
      onRefresh={() => loadDetails(detailsPage)}
    />
  {/if}
</div>
