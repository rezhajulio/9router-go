<script lang="ts">
  import { onMount } from 'svelte'
  import { Loader2 } from 'lucide-svelte'
  import { api, type APIKey, type Combo, type ProviderConnection, type ProviderNode, type Settings } from './api/client'
  import AnalyticsView from './components/analytics/AnalyticsView.svelte'
  import ApiKeysView from './components/ApiKeysView.svelte'
  import CombosView from './components/combos/CombosView.svelte'
  import ConnectionsView from './components/connections/ConnectionsView.svelte'
  import SettingsView from './components/SettingsView.svelte'
  import Sidebar, { type ActiveTab } from './components/Sidebar.svelte'
  import TerminalView from './components/TerminalView.svelte'
  import TopBar from './components/TopBar.svelte'
  import { getAuthHeaders } from './api/client'

  let activeTab = $state<ActiveTab>('connections')
  let connections = $state<ProviderConnection[]>([])
  let providerNodes = $state<ProviderNode[]>([])
  let combos = $state<Combo[]>([])
  let apiKeys = $state<APIKey[]>([])
  let settings = $state<Settings>({})
  let isLoading = $state(true)
  let isCreateComboOpen = $state(false)

  async function loadData() {
    try {
      const [connsRes, nodesRes, combosRes, keysRes, settingsRes] = await Promise.all([
        api.getConnections().catch(() => []),
        api.getProviderNodes().catch(() => []),
        api.getCombos().catch(() => []),
        api.getApiKeys().catch(() => []),
        api.getSettings().catch(() => ({})),
      ])
      connections = connsRes
      providerNodes = nodesRes
      combos = combosRes
      apiKeys = keysRes
      settings = settingsRes
    } finally {
      isLoading = false
    }
  }

  onMount(() => {
    loadData()

    const interval = setInterval(async () => {
      try {
        const [connsRes, nodesRes] = await Promise.all([
          api.getConnections().catch(() => null),
          api.getProviderNodes().catch(() => null)
        ])
        if (connsRes) connections = connsRes
        if (nodesRes) providerNodes = nodesRes
      } catch {
        // silent refresh error
      }
    }, 3000)

    return () => {
      clearInterval(interval)
    }
  })

  let activeConnectionsCount = $derived(connections.filter((c) => c.isActive === 1).length)

  const pageMeta: Record<ActiveTab, { title: string; description: string }> = {
    analytics: { title: 'Overview & Usage', description: 'Real-time routing and token telemetry' },
    connections: { title: 'Providers & Endpoints', description: 'Manage your AI provider connections' },
    combos: { title: 'Combo & Routing', description: 'Model combos and failover strategies' },
    keys: { title: 'CLI & Remote Access', description: 'API keys for your CLI tools' },
    terminal: { title: 'Console Logs', description: 'Live gateway event stream' },
    settings: { title: 'Token Saver & Quota', description: 'RTK engines and system configuration' },
  }

  function handleOpenNewCombo() {
    activeTab = 'combos'
    isCreateComboOpen = true
  }
</script>

<div class="flex h-screen w-full overflow-hidden bg-bg text-text-main font-body transition-colors duration-300">
  <!-- Left Sidebar (upstream w-72 frosted shell) -->
  <Sidebar bind:activeTab activeConnections={activeConnectionsCount} totalConnections={connections.length} />

  <!-- Main Viewport (TopBar + Scrollable Canvas) -->
  <div class="flex-1 flex flex-col min-w-0 h-full relative isolate">
    <!-- Faint grid background (upstream landing-grid) -->
    <div class="landing-grid absolute inset-0 pointer-events-none -z-10" aria-hidden="true"></div>
    <TopBar
      pageTitle={pageMeta[activeTab].title}
      pageDescription={pageMeta[activeTab].description}
      onNewCombo={handleOpenNewCombo}
    />

    <main class="flex-1 overflow-y-auto custom-scrollbar p-6 lg:p-10">
      <div class="max-w-7xl mx-auto">
        {#if isLoading}
          <div class="flex flex-col items-center justify-center h-[70vh] gap-3 text-text-muted">
            <Loader2 class="w-7 h-7 animate-spin text-brand-500" />
            <span class="font-code text-xs">Connecting to 9Router Localhost Gateway (:20130)...</span>
          </div>
        {:else}
          {#if activeTab === 'connections'}
            <ConnectionsView {connections} {providerNodes} onRefresh={loadData} />
          {:else if activeTab === 'combos'}
            <CombosView {combos} {connections} {providerNodes} onRefresh={loadData} bind:isCreatingOpen={isCreateComboOpen} />
          {:else if activeTab === 'analytics'}
            <AnalyticsView {connections} {providerNodes} />
          {:else if activeTab === 'terminal'}
            <TerminalView />
          {:else if activeTab === 'keys'}
            <ApiKeysView {apiKeys} onRefresh={loadData} />
          {:else if activeTab === 'settings'}
            <SettingsView {settings} onRefresh={loadData} />
          {/if}
        {/if}
      </div>
    </main>
  </div>
</div>

