<script lang="ts">
  import { api, type ProviderConnection, type ProviderNode, type Settings } from '../../api/client'
  import { PROVIDER_CATALOG } from '../../lib/providers'
  import { getModelsByProviderId, PROVIDER_ID_TO_ALIAS } from '../../lib/models'
  import { buildAvailableModels, fetchProviderModelsData, isChatModel, type CustomModelData } from './types'
  import ProviderDetailBanner from './ProviderDetailBanner.svelte'
  import ConnectionsListCard from './ConnectionsListCard.svelte'
  import AvailableModelsCard from './AvailableModelsCard.svelte'
  import ProvidersOverviewGrid from './ProvidersOverviewGrid.svelte'
  import AddConnectionModal from './AddConnectionModal.svelte'
  import AddCompatibleNodeModal from './AddCompatibleNodeModal.svelte'
  import AddCustomModelModal from './AddCustomModelModal.svelte'

  interface Props { connections?: ProviderConnection[]; providerNodes?: ProviderNode[]; onRefresh: () => void; selectedProviderId?: string | null }

  let { connections = [], providerNodes = [], onRefresh, selectedProviderId = $bindable(null) }: Props = $props()

  // Modals
  let showAddOpenAIModal = $state(false), showAddAnthropicModal = $state(false), showAddKeyModal = $state(false), showAddCustomModelModal = $state(false), isSubmitting = $state(false), copiedId = $state<string | null>(null)
  let customModels = $state<CustomModelData[]>([])
  let disabledModelIds = $state<string[]>([])
  let settings = $state<Settings | null>(null)
  let strictModelAssignment = $state(false)
  let localAssignedModels = $state<Record<string, string | null>>({})
  function copyText(text: string, id: string) {
    navigator.clipboard.writeText(text)
    copiedId = id
    setTimeout(() => (copiedId = null), 2000)
  }

  // Provider Detail resolution
  let selectedNode = $derived(providerNodes.find((n) => n.id === selectedProviderId))
  let selectedCatalogItem = $derived(PROVIDER_CATALOG.find((p) => p.id === selectedProviderId))
  let selectedConnections = $derived(
    selectedProviderId
      ? connections
          .filter((c) => c.provider === selectedProviderId)
          .map((c) => {
            let dataObj: Record<string, unknown> = {}
            if (typeof c.data === 'string' && c.data) {
              try { dataObj = JSON.parse(c.data) } catch {}
            }
            const pData = c.providerSpecificData as { assignedModel?: string } | undefined
            const assigned = c.id in localAssignedModels ? localAssignedModels[c.id] : (c.assignedModel ?? pData?.assignedModel ?? (dataObj.assignedModel as string) ?? (dataObj.freebuffModel as string) ?? null)
            return {
              ...c,
              assignedModel: assigned,
              providerSpecificData: { ...(c.providerSpecificData || {}), ...((dataObj.providerSpecificData as Record<string, unknown>) || {}), assignedModel: assigned },
            }
          })
      : []
  )

  let currentStorageAlias = $derived(
    selectedNode?.id || selectedCatalogItem?.alias || (selectedProviderId ? PROVIDER_ID_TO_ALIAS[selectedProviderId] : '') || selectedProviderId || ''
  )
  let currentDisplayAlias = $derived(currentStorageAlias)
  let builtInModels = $derived(
    selectedProviderId ? getModelsByProviderId(selectedProviderId).filter(isChatModel) : []
  )
  let providerCustomModels = $derived(
    customModels.filter((m) => (m.providerAlias === currentStorageAlias || m.providerAlias === selectedProviderId) && isChatModel(m))
  )
  let allAvailableModels = $derived(buildAvailableModels(builtInModels, providerCustomModels))
  async function loadModels(providerId: string, storageAlias: string) {
    const data = await fetchProviderModelsData(providerId, storageAlias)
    customModels = data.customModels
    disabledModelIds = data.disabledModelIds
  }

  async function loadSettings(providerId?: string | null) {
    try {
      const s = await api.getSettings()
      settings = s
      const pid = providerId ?? selectedProviderId
      if (pid) strictModelAssignment = s?.providerStrategies?.[pid]?.strictModelAssignment ?? false
    } catch (e) {
      console.error('Failed to load settings:', e)
    }
  }

  $effect(() => {
    if (selectedProviderId) {
      const storageAlias = selectedNode?.id || selectedCatalogItem?.alias || PROVIDER_ID_TO_ALIAS[selectedProviderId] || selectedProviderId
      loadModels(selectedProviderId, storageAlias)
      if (settings) {
        strictModelAssignment = settings.providerStrategies?.[selectedProviderId]?.strictModelAssignment ?? false
      } else {
        loadSettings(selectedProviderId)
      }
    }
  })

  async function handleToggleStrictModelAssignment(enabled: boolean) {
    if (!selectedProviderId) return
    strictModelAssignment = enabled
    const current = settings?.providerStrategies || {}
    const updated = {
      ...current,
      [selectedProviderId]: { ...(current[selectedProviderId] || {}), strictModelAssignment: enabled },
    }
    if (settings) settings.providerStrategies = updated
    try {
      await api.updateSettings({ providerStrategies: updated })
    } catch (err) {
      alert(`Failed to update strict model assignment: ${err instanceof Error ? err.message : String(err)}`)
      strictModelAssignment = !enabled
      if (settings) settings.providerStrategies = current
    }
  }

  async function handleAssignModel(conn: ProviderConnection, modelId: string) {
    const next = modelId || null
    localAssignedModels[conn.id] = next
    try {
      await api.updateConnection(conn.id, { assignedModel: next, providerSpecificData: { assignedModel: next } })
      onRefresh()
    } catch (err) {
      alert(`Failed to assign model: ${err instanceof Error ? err.message : String(err)}`)
      delete localAssignedModels[conn.id]
    }
  }

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
        type: data.type as unknown as string,
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
      {strictModelAssignment}
      availableModels={allAvailableModels}
      onToggleStrictModelAssignment={handleToggleStrictModelAssignment}
      onAssignModel={handleAssignModel}
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
<AddCompatibleNodeModal isOpen={showAddOpenAIModal} type="openai-compatible" {isSubmitting} onClose={() => (showAddOpenAIModal = false)} onSubmit={handleCreateNode} />
<AddCompatibleNodeModal isOpen={showAddAnthropicModal} type="anthropic-compatible" {isSubmitting} onClose={() => (showAddAnthropicModal = false)} onSubmit={handleCreateNode} />
<AddConnectionModal isOpen={showAddKeyModal} title="Add Connection to {selectedNode?.name || selectedCatalogItem?.name || selectedProviderId}" {isSubmitting} onClose={() => (showAddKeyModal = false)} onSubmit={handleAddKeyConnection} />
<AddCustomModelModal isOpen={showAddCustomModelModal} {isSubmitting} onClose={() => (showAddCustomModelModal = false)} onSubmit={submitAddCustomModel} />
