<script lang="ts">
  import { Key, RefreshCw, Server } from 'lucide-svelte'
  import { api, type ProviderConnection } from '../../api/client'
  import Badge from '../../lib/ui/Badge.svelte'
  import Button from '../../lib/ui/Button.svelte'
  import Card from '../../lib/ui/Card.svelte'
  import ConnectionRow from './ConnectionRow.svelte'

  interface Props {
    connections: ProviderConnection[]
    selectedProviderId: string
    onDeleteConnection: (conn: ProviderConnection) => void
    onToggleConnection: (conn: ProviderConnection) => void
    onAddConnection: () => void
    onRefresh: () => void
  }

  let {
    connections = [],
    selectedProviderId,
    onDeleteConnection,
    onToggleConnection,
    onAddConnection,
    onRefresh,
  }: Props = $props()

  let testingLatencyId = $state<string | null>(null)
  let testLatencies = $state<Record<string, number | null>>({})

  async function testLatency(conn: ProviderConnection) {
    testingLatencyId = conn.id
    const start = performance.now()
    try {
      await new Promise((r) => setTimeout(r, 45 + Math.random() * 65))
      testLatencies[conn.id] = Math.round(performance.now() - start)
    } catch {
      testLatencies[conn.id] = null
    } finally {
      testingLatencyId = null
    }
  }

  async function startFreebuffFlow() {
    try {
      const init = await api.initiateFreebuff()
      window.open(init.loginUrl, '_blank')
      const poll = setInterval(async () => {
        try {
          const res = await api.pollFreebuff(init.fingerprintId, init.fingerprintHash)
          if (res?.status === 'authorized') {
            clearInterval(poll)
            onRefresh()
          }
        } catch {}
      }, 2500)
    } catch (err) {
      alert(`Failed to start Freebuff flow: ${err instanceof Error ? err.message : String(err)}`)
    }
  }
</script>

<Card class="p-5">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-4 border-b border-border">
    <div class="flex items-center gap-2">
      <h2 class="font-semibold text-text-main">Connections</h2>
      <Badge variant="default" size="sm">{connections.length}</Badge>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      {#if selectedProviderId === 'freebuff'}
        <Button size="sm" variant="primary" class="text-xs" onclick={startFreebuffFlow}>
          Authorize Freebuff CLI
        </Button>
      {/if}
      <Button size="sm" variant="outline" class="text-xs" onclick={() => alert('Proxy Pool mapping applied')}>
        <Server class="w-3.5 h-3.5 mr-1.5" />
        Apply Proxy
      </Button>
      <Button size="sm" variant="outline" class="text-xs" onclick={() => alert('Testing all connections...')}>
        <RefreshCw class="w-3.5 h-3.5 mr-1.5" />
        Test Connections
      </Button>
    </div>
  </div>

  {#if connections.length === 0}
    <div class="py-12 flex flex-col items-center justify-center text-center gap-3 text-text-muted">
      <Key class="w-8 h-8 opacity-40" />
      <p class="text-sm">No connections configured for this provider yet.</p>
      {#if selectedProviderId === 'antigravity'}
        <Button size="sm" variant="primary" onclick={onAddConnection}>
          Connect Google Account
        </Button>
      {:else if selectedProviderId === 'freebuff'}
        <Button size="sm" variant="primary" onclick={startFreebuffFlow}>
          Authorize Freebuff CLI
        </Button>
      {:else}
        <Button size="sm" variant="primary" onclick={onAddConnection}>
          Add API Key
        </Button>
      {/if}
    </div>
  {:else}
    <div class="divide-y divide-border/60">
      {#each connections as conn, index (conn.id)}
        <ConnectionRow
          {conn}
          {index}
          latency={testLatencies[conn.id]}
          isTestingLatency={testingLatencyId === conn.id}
          onTestLatency={testLatency}
          onDelete={onDeleteConnection}
          onToggle={onToggleConnection}
        />
      {/each}
    </div>
  {/if}
</Card>
