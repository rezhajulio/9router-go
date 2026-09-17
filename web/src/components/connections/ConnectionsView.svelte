<script lang="ts">
  import { api, type ProviderConnection, type ProviderNode } from '../../api/client'
  import { PROVIDER_CATALOG } from '../../lib/providers'
  import { getModelsByProviderId, PROVIDER_ID_TO_ALIAS } from '../../lib/models'
  import {
    buildAvailableModels,
    fetchProviderModelsData,
    type CustomModelData
  } from './types'
  import ProviderDetailBanner from './ProviderDetailBanner.svelte'
  import ConnectionsListCard from './ConnectionsListCard.svelte'
  import AvailableModelsCard from './AvailableModelsCard.svelte'
  import ProvidersOverviewGrid from './ProvidersOverviewGrid.svelte'
  import AddConnectionModal from './AddConnectionModal.svelte'
  import AddCompatibleNodeModal from './AddCompatibleNodeModal.svelte'
  import AddCustomModelModal from './AddCustomModelModal.svelte'

  interface Props {
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
    onRefresh: () => void
    selectedProviderId?: string | null
  }

  let {
    connections = [],
    providerNodes = [],
    onRefresh,
    selectedProviderId = $bindable(null),
  }: Props = $props()

  // Modals
  let showAddOpenAIModal = $state(false)
  let showAddAnthropicModal = $state(false)
  let showAddKeyModal = $state(false)
  let showAddCustomModelModal = $state(false)
  let isSubmitting = $state(false)
  let copiedId = $state<string | null>(null)

  // Provider Detail models data
  let customModels = $state<CustomModelData[]>([])
  let disabledModelIds = $state<string[]>([])

  function copyText(text: string, id: string) {
    navigator.clipboard.writeText(text)
    copiedId = id
    setTimeout(() => (copiedId = null), 2000)
  }

  // Provider Detail resolution
  let selectedNode = $derived(providerNodes.find((n) => n.id === selectedProviderId))
  let selectedCatalogItem = $derived(PROVIDER_CATALOG.find((p) => p.id === selectedProviderId))
  let selectedConnections = $derived(
    selectedProviderId ? connections.filter((c) => c.provider === selectedProviderId) : []
  )

  let currentStorageAlias = $derived(
    selectedNode ? selectedNode.id : selectedCatalogItem?.alias || (selectedProviderId ? PROVIDER_ID_TO_ALIAS[selectedProviderId] : '') || selectedProviderId || ''
  )

  let currentDisplayAlias = $derived(
    selectedNode ? selectedNode.id : selectedCatalogItem?.alias || (selectedProviderId ? PROVIDER_ID_TO_ALIAS[selectedProviderId] : '') || selectedProviderId || ''
  )

  let builtInModels = $derived(selectedProviderId ? getModelsByProviderId(selectedProviderId) : [])
  let providerCustomModels = $derived(
    customModels.filter((m) => m.providerAlias === currentStorageAlias || m.providerAlias === selectedProviderId)
  )

  let allAvailableModels = $derived(buildAvailableModels(builtInModels, providerCustomModels))

  async function loadModels(providerId: string, storageAlias: string) {
    const data = await fetchProviderModelsData(providerId, storageAlias)
    customModels = data.customModels
    disabledModelIds = data.disabledModelIds
  }

  $effect(() => {
    if (selectedProviderId) {
      const storageAlias = selectedNode ? selectedNode.id : selectedCatalogItem?.alias || PROVIDER_ID_TO_ALIAS[selectedProviderId] || selectedProviderId
      loadModels(selectedProviderId, storageAlias)
    }
  })

  async function handleToggleAll(providerId: string, newActive: boolean) {
    const conns = connections.filter((c) => c.provider === providerId)
    await Promise.allSettled(conns.map((c) => api.updateConnection(c.id, { isActive: newActive ? 1 : 0 })))
    onRefresh()
  }

  async function handleToggleConnection(conn: ProviderConnection) {
    try {
      await api.updateConnection(conn.id, { isActive: conn.isActive === 1 ? 0 : 1 })
      onRefresh()
    } catch (err) {
      alert(`Toggle failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleDeleteConnection(conn: ProviderConnection) {
    if (!confirm(`Revoke and delete account connection "${conn.name || conn.id}"?`)) return
    try {
      await api.deleteConnection(conn.id)
      onRefresh()
    } catch (err) {
      alert(`Delete failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleDeleteProviderNode(nodeId: string) {
    if (!confirm(`Delete custom provider endpoint and all attached credentials?`)) return
    try {
      await api.deleteProviderNode(nodeId)
      selectedProviderId = null
      onRefresh()
    } catch (err) {
      alert(`Delete failed: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleCreateNode(data: { name: string; prefix: string; baseUrl: string; apiType?: 'chat' | 'responses'; apiKey?: string; type: string }) {
    isSubmitting = true
    try {
      const node = await api.createProviderNode({
        name: data.name,
        prefix: data.prefix,
        baseUrl: data.baseUrl,
        apiType: data.apiType,
        type: data.type as any,
      })
      if (data.apiKey) {
        await api.createConnection({
          provider: node.id,
          authType: 'compatible',
          name: `${data.name} Key`,
          apiKey: data.apiKey,
        })
      }
      showAddOpenAIModal = false
      showAddAnthropicModal = false
      onRefresh()
      selectedProviderId = node.id
    } catch (err) {
      alert(`Failed to add provider node: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function handleAddKeyConnection(keyName: string, apiKey: string) {
    if (!selectedProviderId) return
    isSubmitting = true
    try {
      await api.createConnection({
        provider: selectedProviderId,
        authType: selectedNode ? 'compatible' : selectedCatalogItem?.category === 'oauth' ? 'oauth' : 'apikey',
        name: keyName || `${selectedNode?.name || selectedCatalogItem?.name || selectedProviderId} Key`,
        apiKey,
      })
      showAddKeyModal = false
      onRefresh()
    } catch (err) {
      alert(`Add connection failed: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }

  async function submitAddCustomModel(modelId: string, modelName: string) {
    if (!currentStorageAlias) return
    isSubmitting = true
    try {
      await api.saveCustomModel(`${currentStorageAlias}|${modelId}|llm`, {
        id: modelId,
        name: modelName || modelId,
        providerAlias: currentStorageAlias,
        type: 'llm',
      })
      showAddCustomModelModal = false
      await loadModels(selectedProviderId!, currentStorageAlias)
    } catch (err) {
      alert(`Failed to add custom model: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSubmitting = false
    }
  }
</script>

{#if selectedProviderId}
  <div class="flex flex-col gap-6 animate-fade-in">
    <ProviderDetailBanner
      {selectedProviderId}
      node={selectedNode}
      catalogItem={selectedCatalogItem}
      connectionCount={selectedConnections.length}
      {copiedId}
      onBack={() => (selectedProviderId = null)}
      onDeleteNode={handleDeleteProviderNode}
      onAddConnection={() => (showAddKeyModal = true)}
      onCopyUrl={(url) => copyText(url, 'node-url')}
    />

    <ConnectionsListCard
      connections={selectedConnections}
      {selectedProviderId}
      onDeleteConnection={handleDeleteConnection}
      onToggleConnection={handleToggleConnection}
      onAddConnection={() => (showAddKeyModal = true)}
      {onRefresh}
    />

    <AvailableModelsCard
      storageAlias={currentStorageAlias}
      displayAlias={currentDisplayAlias}
      {allAvailableModels}
      {disabledModelIds}
      connectionId={selectedConnections[0]?.id}
      onAddCustomModel={() => (showAddCustomModelModal = true)}
      onDisabledModelsChange={(next) => (disabledModelIds = next)}
    />
  </div>
{:else}
  <ProvidersOverviewGrid
    {connections}
    {providerNodes}
    onSelectProvider={(id) => (selectedProviderId = id)}
    onToggleAll={handleToggleAll}
    onAddAnthropic={() => (showAddAnthropicModal = true)}
    onAddOpenAI={() => (showAddOpenAIModal = true)}
  />
{/if}

<!-- Modals -->
<AddCompatibleNodeModal
  isOpen={showAddOpenAIModal}
  type="openai-compatible"
  {isSubmitting}
  onClose={() => (showAddOpenAIModal = false)}
  onSubmit={handleCreateNode}
/>

<AddCompatibleNodeModal
  isOpen={showAddAnthropicModal}
  type="anthropic-compatible"
  {isSubmitting}
  onClose={() => (showAddAnthropicModal = false)}
  onSubmit={handleCreateNode}
/>

<AddConnectionModal
  isOpen={showAddKeyModal}
  title="Add Connection to {selectedNode?.name || selectedCatalogItem?.name || selectedProviderId}"
  {isSubmitting}
  onClose={() => (showAddKeyModal = false)}
  onSubmit={handleAddKeyConnection}
/>

<AddCustomModelModal
  isOpen={showAddCustomModelModal}
  {isSubmitting}
  onClose={() => (showAddCustomModelModal = false)}
  onSubmit={submitAddCustomModel}
/>
