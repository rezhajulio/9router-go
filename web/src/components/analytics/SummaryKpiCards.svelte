<script lang="ts">
  import Card from '../../lib/ui/Card.svelte'
  import { fmt, fmtCost, type StatsData } from './types'

  let { stats = {} }: { stats?: StatsData } = $props()
</script>

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
