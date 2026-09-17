<script lang="ts">
  // Port of decolua/9router src/shared/components/Sidebar.js — Svelte 5 version.
  import {
    Activity,
    Binary,
    BookOpen,
    ChevronDown,
    ChevronRight,
    Coins,
    Globe,
    Image,
    Key,
    Layers,
    Mic,
    Network,
    Radio,
    Server,
    Settings,
    Terminal,
    Video,
    Volume2
  } from 'lucide-svelte'
  import { api } from '../api/client'
  import { TAB_ROUTES, type ActiveTab } from '../lib/router'

  export type { ActiveTab }

  let {
    activeTab = $bindable('connections'),
    navigate = (tab: ActiveTab) => {
      activeTab = tab
    },
    activeConnections = 0,
    totalConnections = 0
  }: {
    activeTab: ActiveTab
    navigate?: (tab: ActiveTab, replace?: boolean) => void
    activeConnections: number
    totalConnections: number
  } = $props()

  let version = $state('')

  $effect(() => {
    api
      .getSystemVersion()
      .then((v) => (version = (v as { currentVersion?: string }).currentVersion || ''))
      .catch(() => {})
  })

  const groups = [
    {
      label: 'Main',
      items: [
        { tab: 'analytics' as ActiveTab, label: 'Overview & Usage', icon: Activity },
        { tab: 'connections' as ActiveTab, label: 'Providers & Endpoints', icon: Server },
        { tab: 'combos' as ActiveTab, label: 'Combo & Routing', icon: Layers },
        { tab: 'keys' as ActiveTab, label: 'CLI & Remote Access', icon: Key },
      ],
    },
    {
      label: 'System',
      items: [
        { tab: 'terminal' as ActiveTab, label: 'Console Logs', icon: Terminal, badge: 'LIVE' },
        { tab: 'settings' as ActiveTab, label: 'Token Saver & Quota', icon: Coins },
      ],
    },
  ] as const

  const mediaItems = [
    { tab: 'media-embedding' as ActiveTab, label: 'Embedding', icon: Binary },
    { tab: 'media-image' as ActiveTab, label: 'Text to Image', icon: Image },
    { tab: 'media-tts' as ActiveTab, label: 'Text To Speech', icon: Volume2 },
    { tab: 'media-stt' as ActiveTab, label: 'Speech To Text', icon: Mic },
    { tab: 'media-video' as ActiveTab, label: 'Video', icon: Video },
    { tab: 'media-web' as ActiveTab, label: 'Web Fetch & Search', icon: Globe },
  ]

  let isMediaOpen = $state(true)
</script>

<aside
  class="flex w-72 flex-col border-r border-border-subtle bg-sidebar backdrop-blur-xl transition-colors duration-300 min-h-full flex-shrink-0 select-none z-30"
>
  <!-- Traffic lights -->
  <div class="flex items-center gap-2 px-6 pt-5 pb-2">
    <div class="w-3 h-3 rounded-full bg-[#FF5F56]"></div>
    <div class="w-3 h-3 rounded-full bg-[#FFBD2E]"></div>
    <div class="w-3 h-3 rounded-full bg-[#27C93F]"></div>
  </div>

  <!-- Logo + version -->
  <div class="px-6 py-4 flex flex-col gap-2">
    <div class="flex items-center gap-3">
      <div
        class="flex items-center justify-center size-9 rounded-[10px] bg-gradient-to-br from-brand-500 to-brand-700 shadow-[var(--shadow-warm)] animate-disco"
      >
        <Network class="text-text-main w-5 h-5" />
      </div>
      <div class="flex flex-col">
        <h1 class="text-lg font-semibold tracking-tight text-text-main">9Router</h1>
        <span class="text-xs text-text-muted">v{version || '…'}</span>
      </div>
    </div>

    <!-- Gateway status -->
    <div
      class="flex items-center justify-between px-3 py-1.5 rounded-[10px] bg-surface border border-border-subtle text-xs font-medium text-text-muted"
    >
      <div class="flex items-center gap-1.5">
        <span class="relative flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
        </span>
        Gateway :20130
      </div>
      <span class="text-success font-semibold">ONLINE</span>
    </div>
  </div>

  <!-- Navigation -->
  <nav class="flex-1 px-4 py-2 space-y-4 overflow-y-auto custom-scrollbar">
    {#each groups as group (group.label)}
      <div class="space-y-0.5">
        <p
          class="px-4 pt-1 text-xs font-semibold text-text-muted/60 uppercase tracking-wider mb-1"
        >
          {group.label}
        </p>
        {#each group.items as item (item.tab)}
          {@const Icon = item.icon}
          {@const isActive = activeTab === item.tab}
          <a
            href={TAB_ROUTES[item.tab]}
            onclick={(e) => {
              if (!e.ctrlKey && !e.metaKey && !e.shiftKey && !e.altKey && e.button === 0) {
                e.preventDefault()
                navigate(item.tab)
              }
            }}
            class="w-full flex items-center justify-between gap-3 px-3 py-2 rounded-[10px] transition-all group cursor-pointer {isActive
              ? 'bg-brand-500/10 text-brand-600 dark:text-brand-400'
              : 'text-text-muted hover:bg-surface-2 hover:text-text-main'}"
          >
            <div class="flex items-center gap-3">
              <Icon class="w-[18px] h-[18px] {isActive ? '' : 'group-hover:text-brand-500 transition-colors'}" />
              <span class="text-[13px] font-medium">{item.label}</span>
            </div>
            {#if 'badge' in item}
              <span
                class="text-[9px] px-1.5 py-0.5 rounded bg-success/15 text-success font-bold uppercase tracking-wider"
              >
                {item.badge}
              </span>
            {/if}
          </a>
        {/each}
      </div>
    {/each}

    <!-- Media Providers (collapsible) -->
    <div class="space-y-0.5 pt-1">
      <button
        type="button"
        onclick={() => (isMediaOpen = !isMediaOpen)}
        class="w-full flex items-center justify-between px-4 pt-1 text-xs font-semibold text-text-muted/60 uppercase tracking-wider mb-1 hover:text-text-muted cursor-pointer transition-colors"
      >
        <span>Media Providers</span>
        {#if isMediaOpen}
          <ChevronDown class="w-3.5 h-3.5" />
        {:else}
          <ChevronRight class="w-3.5 h-3.5" />
        {/if}
      </button>

      {#if isMediaOpen}
        {#each mediaItems as item (item.tab)}
          {@const Icon = item.icon}
          {@const isActive = activeTab === item.tab}
          <a
            href={TAB_ROUTES[item.tab]}
            onclick={(e) => {
              if (!e.ctrlKey && !e.metaKey && !e.shiftKey && !e.altKey && e.button === 0) {
                e.preventDefault()
                navigate(item.tab)
              }
            }}
            class="w-full flex items-center justify-between gap-3 px-3 py-2 rounded-[10px] transition-all group cursor-pointer {isActive
              ? 'bg-brand-500/10 text-brand-600 dark:text-brand-400'
              : 'text-text-muted hover:bg-surface-2 hover:text-text-main'}"
          >
            <div class="flex items-center gap-3">
              <Icon class="w-[18px] h-[18px] {isActive ? '' : 'group-hover:text-brand-500 transition-colors'}" />
              <span class="text-[13px] font-medium">{item.label}</span>
            </div>
          </a>
        {/each}
      {/if}
    </div>
  </nav>
  <!-- Bottom: connection summary + footer -->
  <div class="p-4 border-t border-border-subtle space-y-2">
    <div
      class="p-3 rounded-[10px] bg-surface border border-border-subtle flex items-center justify-between"
    >
      <div>
        <p class="text-xs text-text-muted">Provider Connections</p>
        <p class="text-sm font-semibold text-text-main mt-0.5">
          {activeConnections}
          <span class="text-text-muted font-normal">/ {totalConnections} active</span>
        </p>
      </div>
      <Radio class="w-4 h-4 text-brand-500" />
    </div>

    <div class="flex items-center justify-between px-1 text-[11px] text-text-subtle">
      <span class="flex items-center gap-1 hover:text-text-muted cursor-pointer">
        <BookOpen class="w-3 h-3" />
        Docs
      </span>
      <span class="flex items-center gap-1">
        <Settings class="w-3 h-3" />
        Local Mode
      </span>
    </div>
  </div>
</aside>