<script lang="ts">
  // Port of decolua/9router src/shared/components/Header.js — Svelte 5 version.
  import { ChevronRight, Plus, Search } from 'lucide-svelte'
  import ThemeToggle from '../lib/ui/ThemeToggle.svelte'
  import Button from '../lib/ui/Button.svelte'

  let {
    pageTitle = 'Dashboard',
    pageDescription = '',
    onNewCombo,
    onSearch
  }: {
    pageTitle?: string
    pageDescription?: string
    onNewCombo?: () => void
    onSearch?: (query: string) => void
  } = $props()

  let searchInput = $state('')

  function handleInput(e: Event) {
    const val = (e.target as HTMLInputElement).value
    searchInput = val
    if (onSearch) onSearch(val)
  }
</script>

<header
  class="h-16 bg-vibrancy backdrop-blur-xl border-b border-border-subtle px-6 flex items-center justify-between gap-4 flex-shrink-0 z-20"
>
  <!-- Left: breadcrumb + title -->
  <div class="flex items-center gap-2 min-w-0">
    <div
      class="flex items-center justify-center size-8 rounded-[10px] bg-gradient-to-br from-brand-500 to-brand-700 shadow-[var(--shadow-warm)] flex-shrink-0 animate-disco"
    >
      <span class="text-text-main font-bold text-sm">9R</span>
    </div>
    <ChevronRight class="w-4 h-4 text-text-subtle flex-shrink-0" />
    <div class="min-w-0">
      <p class="text-sm font-semibold text-text-main truncate leading-tight">{pageTitle}</p>
      {#if pageDescription}
        <p class="text-xs text-text-muted truncate leading-tight">{pageDescription}</p>
      {/if}
    </div>
  </div>

  <!-- Center search -->
  <div class="relative w-full max-w-md hidden md:flex items-center">
    <Search class="absolute left-3 w-4 h-4 text-text-subtle pointer-events-none" />
    <input
      type="text"
      placeholder="Search providers, models, keys..."
      value={searchInput}
      oninput={handleInput}
      class="w-full bg-surface border border-border-subtle rounded-[10px] pl-9 pr-16 py-2 text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:border-brand-500/50 transition"
    />
    <span
      class="absolute right-2.5 px-1.5 py-0.5 rounded-md bg-surface-2 font-code text-[10px] text-text-subtle pointer-events-none border border-border"
    >
      ⌘K
    </span>
  </div>

  <!-- Right actions -->
  <div class="flex items-center gap-3">
    <ThemeToggle />
    <Button size="sm" onclick={() => onNewCombo?.()}>
      <Plus class="w-4 h-4" />
      New Combo
    </Button>
  </div>
</header>