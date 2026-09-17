<script lang="ts">
  import {
    AlertTriangle,
    Bot,
    Plus,
    RotateCcw,
    X
  } from 'lucide-svelte'
  import { api, type FreebuffSessionStatusResponse } from '../../api/client'
  import Badge from '../../lib/ui/Badge.svelte'
  import Button from '../../lib/ui/Button.svelte'
  import Card from '../../lib/ui/Card.svelte'
  import type { ModelItem } from './types'
  import AvailableModelItem from './AvailableModelItem.svelte'
  interface Props {
    storageAlias: string
    displayAlias: string
    allAvailableModels: ModelItem[]
    disabledModelIds: string[]
    connectionId?: string
    onAddCustomModel: () => void
    onDisabledModelsChange: (newDisabledIds: string[]) => void
  }

  let {
    storageAlias,
    displayAlias,
    allAvailableModels = [],
    disabledModelIds = [],
    connectionId,
    onAddCustomModel,
    onDisabledModelsChange,
  }: Props = $props()

  let isFreebuff = $derived(
    storageAlias === 'freebuff' || storageAlias === 'fb' || displayAlias === 'fb'
  )

  let freebuffSession = $state<FreebuffSessionStatusResponse | null>(null)
  let isLoadingSession = $state(false)

  async function loadFreebuffSession() {
    if (!isFreebuff) {
      freebuffSession = null
      return
    }
    isLoadingSession = true
    try {
      freebuffSession = await api.getFreebuffSessionStatus(connectionId)
    } catch (err) {
      console.error('Failed to fetch Freebuff session status:', err)
      freebuffSession = null
    } finally {
      isLoadingSession = false
    }
  }

  $effect(() => {
    if (isFreebuff) {
      loadFreebuffSession()
    } else {
      freebuffSession = null
    }
  })

  let sessionExpiresInMin = $derived.by(() => {
    if (!freebuffSession?.expiresAt) return null
    const exp = new Date(freebuffSession.expiresAt).getTime()
    const diffMs = exp - Date.now()
    return Math.max(0, Math.round(diffMs / 60000))
  })

  function checkIsActiveSession(modelId: string): boolean {
    if (!isFreebuff || freebuffSession?.status !== 'active' || !freebuffSession?.currentModel) {
      return false
    }
    const cur = freebuffSession.currentModel.toLowerCase().trim()
    const mid = modelId.toLowerCase().trim()
    return mid === cur || mid.endsWith('/' + cur) || cur.endsWith('/' + mid)
  }

  function checkIsLockedBySession(modelId: string): boolean {
    if (!isFreebuff || freebuffSession?.status !== 'active' || !freebuffSession?.currentModel) {
      return false
    }
    return !checkIsActiveSession(modelId)
  }
  let thinkingMode = $state('auto')
  let testingModelIds = $state<Set<string>>(new Set())
  let modelTestResults = $state<Record<string, 'ok' | 'error'>>({})
  let modelsTestError = $state('')
  let copiedModelId = $state<string | null>(null)

  let displayModels = $derived(
    allAvailableModels.filter((m) => !disabledModelIds.includes(m.id))
  )

  let disabledDisplayModels = $derived(
    allAvailableModels.filter((m) => disabledModelIds.includes(m.id))
  )

  let hasReasoningModels = $derived(
    allAvailableModels.some((m) => m.caps.reasoning)
  )

  async function handleTestModel(modelId: string, fullModel: string) {
    if (testingModelIds.has(modelId)) return
    testingModelIds = new Set([...testingModelIds, modelId])
    modelsTestError = ''
    try {
      const res = await api.testModel(fullModel)
      modelTestResults = { ...modelTestResults, [modelId]: res.ok ? 'ok' : 'error' }
      if (!res.ok) {
        modelsTestError = res.error || 'Model not reachable'
      }
    } catch (err) {
      modelTestResults = { ...modelTestResults, [modelId]: 'error' }
      modelsTestError = err instanceof Error ? err.message : 'Network error'
    } finally {
      const next = new Set(testingModelIds)
      next.delete(modelId)
      testingModelIds = next
    }
  }

  async function handleDisableModel(modelId: string) {
    if (!storageAlias) return
    const next = [...new Set([...disabledModelIds, modelId])]
    onDisabledModelsChange(next)
    try {
      await api.saveDisabledModels(storageAlias, next)
    } catch (err) {
      console.error('Failed to save disabled models:', err)
    }
  }

  async function handleEnableModel(modelId: string) {
    if (!storageAlias) return
    const next = disabledModelIds.filter((id) => id !== modelId)
    onDisabledModelsChange(next)
    try {
      await api.saveDisabledModels(storageAlias, next)
    } catch (err) {
      console.error('Failed to save disabled models:', err)
    }
  }

  async function handleDisableAll() {
    if (!storageAlias) return
    const allIds = allAvailableModels.map((m) => m.id)
    onDisabledModelsChange(allIds)
    try {
      await api.saveDisabledModels(storageAlias, allIds)
    } catch (err) {
      console.error('Failed to disable all models:', err)
    }
  }

  async function handleEnableAll() {
    if (!storageAlias) return
    onDisabledModelsChange([])
    try {
      await api.saveDisabledModels(storageAlias, [])
    } catch (err) {
      console.error('Failed to enable all models:', err)
    }
  }

  function copyModel(fullModel: string, modelId: string) {
    const model = allAvailableModels.find((m) => m.id === modelId)
    const isReasoning = model?.caps.reasoning ?? false
    const textToCopy =
      isReasoning && thinkingMode && thinkingMode !== 'auto'
        ? `${fullModel}(${thinkingMode})`
        : fullModel
    navigator.clipboard.writeText(textToCopy)
    copiedModelId = modelId
    setTimeout(() => {
      if (copiedModelId === modelId) copiedModelId = null
    }, 2000)
  }
</script>

<Card class="p-5">
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-4 border-b border-border">
    <div class="flex flex-wrap items-center gap-3">
      <div class="flex items-center gap-2">
        <h2 class="font-semibold text-text-main">Available Models</h2>
        <Badge tone="default">{displayModels.length}</Badge>
      </div>

      {#if hasReasoningModels}
        <select
          bind:value={thinkingMode}
          class="h-7 rounded-lg border border-border bg-surface px-2 text-xs text-text-main outline-none transition-colors hover:border-brand-500/40 cursor-pointer"
        >
          <option value="auto">Thinking: Auto</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
      {/if}
    </div>

    <div class="flex flex-wrap items-center gap-2">
      {#if disabledDisplayModels.length > 0}
        <Button size="sm" variant="outline" class="text-xs" onclick={handleEnableAll}>
          <RotateCcw class="w-3.5 h-3.5 mr-1.5" />
          Active All
        </Button>
      {/if}
      {#if displayModels.length > 0}
        <Button size="sm" variant="outline" class="text-xs text-red-500 hover:text-red-600 hover:border-red-500/40" onclick={handleDisableAll}>
          <X class="w-3.5 h-3.5 mr-1.5" />
          Disable All
        </Button>
      {/if}
    </div>
  </div>

  {#if modelsTestError}
    <div class="mt-4 p-3.5 rounded-xl border border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400 text-xs flex items-start gap-2.5 leading-relaxed">
      <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
      <div class="flex-1">
        <p class="font-medium">{modelsTestError}</p>
      </div>
      <button
        type="button"
        onclick={() => (modelsTestError = '')}
        class="text-red-600 dark:text-red-400 hover:opacity-75 cursor-pointer"
      >
        <X class="w-4 h-4" />
      </button>
    </div>
  {/if}
  {#if isFreebuff && freebuffSession?.status === 'active' && freebuffSession?.currentModel}
    <div class="mt-4 p-4 rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-200 text-xs flex items-start gap-3 leading-relaxed shadow-xs">
      <span class="text-base shrink-0 leading-none">🔒</span>
      <div class="flex-1 min-w-0">
        <div class="flex items-center justify-between gap-2 flex-wrap">
          <p class="font-semibold text-sm text-emerald-900 dark:text-emerald-100">
            Active Session: <span class="font-mono bg-emerald-500/20 px-1.5 py-0.5 rounded text-xs">{freebuffSession.currentModel}</span>
            {#if sessionExpiresInMin !== null}
              <span class="font-normal text-xs text-emerald-700 dark:text-emerald-300 ml-1">
                ({sessionExpiresInMin > 0 ? `Expires in ${sessionExpiresInMin} min` : 'Expires soon'})
              </span>
            {/if}
          </p>
          <button
            type="button"
            title="Refresh session status"
            disabled={isLoadingSession}
            onclick={loadFreebuffSession}
            class="p-1 rounded-md text-emerald-700 dark:text-emerald-300 hover:bg-emerald-500/20 transition-colors cursor-pointer"
          >
            <RotateCcw class="w-3.5 h-3.5 {isLoadingSession ? 'animate-spin' : ''}" />
          </button>
        </div>
        <p class="mt-1 text-emerald-700/90 dark:text-emerald-300/90">
          Freebuff limits each account to 1 model per hour. Other models are locked until this session expires.
        </p>
      </div>
    </div>
  {/if}


  <div class="mt-4 flex flex-col gap-4">
    {#if allAvailableModels.length === 0}
      <div class="py-12 flex flex-col items-center justify-center text-center gap-3 text-text-muted">
        <Bot class="w-8 h-8 opacity-40" />
        <p class="text-sm">No models registered for this provider yet.</p>
        <Button size="sm" variant="primary" onclick={onAddCustomModel}>
          <Plus class="w-4 h-4 mr-1.5" />
          Add Model
        </Button>
      </div>
    {:else}
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
        {#each displayModels as model (model.id)}
          {@const fullModel = displayAlias ? `${displayAlias}/${model.id}` : model.id}
          {@const displayModelText = `${fullModel}${thinkingMode !== 'auto' && model.caps.reasoning ? `(${thinkingMode})` : ''}`}

          <AvailableModelItem
            {model}
            {fullModel}
            {displayModelText}
            isTesting={testingModelIds.has(model.id)}
            testResult={modelTestResults[model.id]}
            isCopied={copiedModelId === model.id}
            isActiveSession={checkIsActiveSession(model.id)}
            isLockedBySession={checkIsLockedBySession(model.id)}
            onTest={() => handleTestModel(model.id, fullModel)}
            onCopy={() => copyModel(fullModel, model.id)}
            onDisable={() => handleDisableModel(model.id)}
          />
        {/each}

        <!-- Add Model dashed card -->
        <button
          type="button"
          onclick={onAddCustomModel}
          class="p-3 rounded-xl border border-dashed border-border hover:border-brand-500/50 bg-transparent hover:bg-black/[0.01] dark:hover:bg-white/[0.01] transition-colors flex items-center justify-center gap-2 text-xs font-medium text-text-muted hover:text-text-main cursor-pointer min-h-[58px]"
        >
          <Plus class="w-4 h-4" />
          <span>Add Model</span>
        </button>
      </div>

      {#if disabledDisplayModels.length > 0}
        <div class="pt-3 border-t border-border flex flex-col gap-2">
          <div class="text-xs font-medium text-text-muted">
            Disabled models ({disabledDisplayModels.length})
          </div>
          <div class="flex flex-wrap gap-1.5">
            {#each disabledDisplayModels as m (m.id)}
              <button
                type="button"
                onclick={() => handleEnableModel(m.id)}
                class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-mono bg-surface-2 hover:bg-surface-3 text-text-muted hover:text-text-main border border-border transition-colors cursor-pointer"
                title="Enable model {m.id}"
              >
                <span>+</span>
                <span>{m.id}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}
    {/if}
  </div>
</Card>
