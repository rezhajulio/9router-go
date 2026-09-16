<script lang="ts">
  import {
    Activity,
    ArrowDown,
    ArrowUp,
    Check,
    Copy,
    Eye,
    Gavel,
    GripVertical,
    Headphones,
    Layers,
    Loader2,
    Pencil,
    Plus,
    RefreshCw,
    Search,
    Sparkles,
    Trash2,
    X,
    Zap
  } from 'lucide-svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import { api, type Combo, type ProviderConnection, type ProviderNode } from '../api/client'
  let {
    combos = [],
    connections = [],
    providerNodes = [],
    onRefresh,
    isCreatingOpen = $bindable(false)
  }: {
    combos: Combo[]
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
    onRefresh: () => void
    isCreatingOpen?: boolean
  } = $props()

  // Combo Strategy and Capacity Adapter state
  let comboStrategies = $state<Record<string, { fallbackStrategy?: string; judgeModel?: string }>>({})
  let capacityAdapter = $state<{
    vision: { enabled: boolean; roundRobin: boolean; models: string[] }
    audioInput: { enabled: boolean; roundRobin: boolean; models: string[] }
  }>({
    vision: { enabled: true, roundRobin: false, models: ['ag/gemini-3.7-flash-high'] },
    audioInput: { enabled: true, roundRobin: false, models: [] }
  })

  let isSavingAdapter = $state(false)
  let copiedId = $state<string | null>(null)

  // Edit / Create Modal state
  let editingCombo = $state<Combo | null>(null)
  let modalName = $state('')
  let modalModels = $state<string[]>([])
  let modalNameError = $state('')
  let isSavingCombo = $state(false)

  // Model Picker Modal state
  let showModelPicker = $state(false)
  let modelPickerTarget = $state<'combo' | 'vision' | 'audio' | 'judge'>('combo')
  let modelPickerSearch = $state('')
  let customModelInput = $state('')

  // Confirm Delete Modal state
  let deletingCombo = $state<Combo | null>(null)

  const VALID_NAME_REGEX = /^[a-zA-Z0-9_.\-]+$/

  // Normalize models in combo
  function getComboModels(c: Combo): string[] {
    if (Array.isArray(c.models)) return c.models
    if (typeof c.models === 'string') {
      try {
        const parsed = JSON.parse(c.models)
        if (Array.isArray(parsed)) return parsed
        return [c.models]
      } catch {
        return c.models ? [c.models] : []
      }
    }
    return []
  }

  // Load settings for capacityAdapter & comboStrategies
  async function loadSettings() {
    try {
      const s = await api.getSettings()
      if (s?.comboStrategies && typeof s.comboStrategies === 'object') {
        comboStrategies = s.comboStrategies
      }
      if (s?.capacityAdapter && typeof s.capacityAdapter === 'object') {
        const ca = s.capacityAdapter
        capacityAdapter = {
          vision: {
            enabled: ca.vision?.enabled !== false,
            roundRobin: !!ca.vision?.roundRobin,
            models: Array.isArray(ca.vision?.models) ? ca.vision.models : ['ag/gemini-3.7-flash-high']
          },
          audioInput: {
            enabled: ca.audioInput?.enabled !== false,
            roundRobin: !!ca.audioInput?.roundRobin,
            models: Array.isArray(ca.audioInput?.models) ? ca.audioInput.models : []
          }
        }
      }
    } catch (e) {
      console.error('Failed to load settings:', e)
    }
  }

  $effect(() => {
    loadSettings()
  })

  // Watch isCreatingOpen prop
  $effect(() => {
    if (isCreatingOpen && !editingCombo) {
      modalName = ''
      modalModels = []
      modalNameError = ''
    }
  })

  function hasVision(model: string): boolean {
    const m = model.toLowerCase()
    return (
      m.includes('vision') ||
      m.includes('gemini') ||
      m.includes('claude-3') ||
      m.includes('claude-sonnet') ||
      m.includes('claude-opus') ||
      m.includes('gpt-4o') ||
      m.includes('spark') ||
      m.includes('vl') ||
      m.includes('flash')
    )
  }

  function hasReasoning(model: string): boolean {
    const m = model.toLowerCase()
    return (
      m.includes('reason') ||
      m.includes('think') ||
      m.includes('r1') ||
      m.includes('deepseek') ||
      m.includes('spark') ||
      m.includes('high') ||
      m.includes('o1') ||
      m.includes('o3') ||
      m.includes('pro-agent')
    )
  }

  function copyName(name: string, id: string) {
    navigator.clipboard.writeText(name)
    copiedId = id
    setTimeout(() => {
      if (copiedId === id) copiedId = null
    }, 2000)
  }

  // Update per-combo strategy
  async function handleSetStrategy(combo: Combo, newStrategy: string) {
    const updated = { ...comboStrategies }
    const current = updated[combo.name] || {}
    const next = { ...current, fallbackStrategy: newStrategy }

    if (newStrategy === 'fallback' && !next.judgeModel) {
      delete updated[combo.name]
    } else {
      updated[combo.name] = next
    }
    comboStrategies = updated

    try {
      await api.patchSettings({ comboStrategies: updated })
      await api.updateCombo(combo.id, { strategy: newStrategy })
      onRefresh()
    } catch (e) {
      console.error('Failed to update combo strategy:', e)
    }
  }

  async function handleSetJudge(comboName: string, judgeModel: string) {
    const updated = { ...comboStrategies }
    const current = updated[comboName] || {}
    updated[comboName] = { ...current, judgeModel }
    comboStrategies = updated

    try {
      await api.patchSettings({ comboStrategies: updated })
    } catch (e) {
      console.error('Failed to update judge model:', e)
    }
  }

  async function clearJudge(comboName: string) {
    const updated = { ...comboStrategies }
    if (updated[comboName]) {
      const { judgeModel, ...rest } = updated[comboName]
      if (!rest.fallbackStrategy || rest.fallbackStrategy === 'fallback') {
        delete updated[comboName]
      } else {
        updated[comboName] = rest
      }
      comboStrategies = updated
      try {
        await api.patchSettings({ comboStrategies: updated })
      } catch (e) {
        console.error('Failed to clear judge:', e)
      }
    }
  }

  // Capacity Adapter updates
  async function saveCapacityAdapter(next: typeof capacityAdapter) {
    capacityAdapter = next
    try {
      isSavingAdapter = true
      await api.patchSettings({ capacityAdapter: next })
    } catch (e) {
      console.error('Failed to update capacity adapter:', e)
    } finally {
      isSavingAdapter = false
    }
  }

  function toggleAdapter(type: 'vision' | 'audioInput', enabled: boolean) {
    saveCapacityAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], enabled }
    })
  }

  function toggleAdapterRoundRobin(type: 'vision' | 'audioInput', roundRobin: boolean) {
    saveCapacityAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], roundRobin }
    })
  }

  function moveAdapterModel(type: 'vision' | 'audioInput', index: number, delta: number) {
    const arr = [...capacityAdapter[type].models]
    const target = index + delta
    if (target < 0 || target >= arr.length) return
    const temp = arr[index]
    arr[index] = arr[target]
    arr[target] = temp
    saveCapacityAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], models: arr }
    })
  }

  function removeAdapterModel(type: 'vision' | 'audioInput', index: number) {
    const arr = capacityAdapter[type].models.filter((_, i) => i !== index)
    saveCapacityAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], models: arr }
    })
  }

  // Modal Open Handlers
  function openCreateModal() {
    editingCombo = null
    modalName = ''
    modalModels = []
    modalNameError = ''
    isCreatingOpen = true
  }

  function openEditModal(combo: Combo) {
    editingCombo = combo
    modalName = combo.name
    modalModels = [...getComboModels(combo)]
    modalNameError = ''
    isCreatingOpen = true
  }

  function closeModal() {
    isCreatingOpen = false
    editingCombo = null
    modalName = ''
    modalModels = []
    modalNameError = ''
  }

  function validateModalName(name: string): boolean {
    if (!name.trim()) {
      modalNameError = 'Name is required'
      return false
    }
    if (!VALID_NAME_REGEX.test(name.trim())) {
      modalNameError = 'Only letters, numbers, -, _ and . allowed'
      return false
    }
    modalNameError = ''
    return true
  }

  async function handleSaveCombo() {
    if (!validateModalName(modalName)) return
    isSavingCombo = true
    try {
      if (editingCombo) {
        await api.updateCombo(editingCombo.id, {
          name: modalName.trim(),
          models: modalModels
        })
      } else {
        await api.createCombo({
          name: modalName.trim(),
          models: modalModels,
          strategy: 'fallback'
        })
      }
      closeModal()
      onRefresh()
    } catch (e: any) {
      modalNameError = e?.message || 'Failed to save combo'
    } finally {
      isSavingCombo = false
    }
  }

  function confirmDeleteCombo(combo: Combo) {
    deletingCombo = combo
  }

  async function handleDeleteCombo() {
    if (!deletingCombo) return
    try {
      await api.deleteCombo(deletingCombo.id)
      deletingCombo = null
      onRefresh()
    } catch (e) {
      console.error('Failed to delete combo:', e)
    }
  }

  // Model Picker Modal logic
  function openModelPicker(target: 'combo' | 'vision' | 'audio' | 'judge') {
    modelPickerTarget = target
    modelPickerSearch = ''
    customModelInput = ''
    showModelPicker = true
  }

  function selectModel(modelValue: string) {
    if (modelPickerTarget === 'combo') {
      if (!modalModels.includes(modelValue)) {
        modalModels = [...modalModels, modelValue]
      }
    } else if (modelPickerTarget === 'vision') {
      if (!capacityAdapter.vision.models.includes(modelValue)) {
        saveCapacityAdapter({
          ...capacityAdapter,
          vision: {
            ...capacityAdapter.vision,
            models: [...capacityAdapter.vision.models, modelValue]
          }
        })
      }
    } else if (modelPickerTarget === 'audio') {
      if (!capacityAdapter.audioInput.models.includes(modelValue)) {
        saveCapacityAdapter({
          ...capacityAdapter,
          audioInput: {
            ...capacityAdapter.audioInput,
            models: [...capacityAdapter.audioInput.models, modelValue]
          }
        })
      }
    } else if (modelPickerTarget === 'judge' && editingCombo) {
      handleSetJudge(editingCombo.name, modelValue)
    }
    showModelPicker = false
  }

  // Available models aggregated from connections + providerNodes
  let availableModels = $derived(() => {
    const list: Array<{ value: string; label: string; provider: string; vision: boolean; reasoning: boolean }> = []
    const seen = new Set<string>()

    // Gather from providerNodes
    for (const node of providerNodes) {
      const p = node.id
      if (node.models) {
        for (const m of node.models) {
          const val = `${p}/${m}`
          if (!seen.has(val)) {
            seen.add(val)
            list.push({
              value: val,
              label: m,
              provider: node.name || p,
              vision: hasVision(m),
              reasoning: hasReasoning(m)
            })
          }
        }
      }
    }

    // Also include common models from existing combos
    for (const c of combos) {
      const cms = getComboModels(c)
      for (const m of cms) {
        if (!seen.has(m)) {
          seen.add(m)
          const prov = m.includes('/') ? m.split('/')[0] : 'combo'
          list.push({
            value: m,
            label: m.includes('/') ? m.split('/')[1] : m,
            provider: prov,
            vision: hasVision(m),
            reasoning: hasReasoning(m)
          })
        }
      }
    }

    return list
  })

  let filteredPickerModels = $derived(() => {
    const all = availableModels()
    return all.filter((item) => {
      if (modelPickerTarget === 'vision' && !item.vision) {
        // Soft filter: if vision target, prioritize vision, but allow if matches search
        if (!modelPickerSearch.trim()) return item.vision
      }
      if (modelPickerSearch.trim()) {
        const q = modelPickerSearch.toLowerCase()
        return item.value.toLowerCase().includes(q) || item.provider.toLowerCase().includes(q)
      }
      return true
    })
  })
</script>

<div class="flex min-w-0 flex-col gap-6 px-1 sm:px-0">
  <!-- Header / Explainer (matches upstream Next.js exactly) -->
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0">
      <p class="text-sm text-text-muted mt-1">
        Group models under one name, then pick a strategy per combo:
      </p>
      <ul class="text-sm text-text-muted mt-2 flex flex-col gap-1">
        <li>
          <span class="font-medium text-text-main">Fallback</span> — tries models in order (next on failure)
        </li>
        <li>
          <span class="font-medium text-text-main">Round Robin</span> — rotates models across requests to spread load
        </li>
        <li>
          <span class="font-medium text-text-main">Fusion</span> — queries all models in parallel, then a judge
          synthesizes one answer. Best quality, but costs the most: every request bills all panel models + the judge
          (N+1 calls)
        </li>
      </ul>
    </div>
    <button
      type="button"
      onclick={openCreateModal}
      class="inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-medium shadow-sm transition-colors cursor-pointer w-full sm:w-auto shrink-0"
    >
      <Plus class="w-4 h-4" />
      <span>Create Combo</span>
    </button>
  </div>

  <!-- Combos List -->
  {#if combos.length === 0}
    <div class="rounded-xl border border-border bg-surface p-12 text-center">
      <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-brand-500/10 text-brand-500 mb-4">
        <Layers class="w-8 h-8" />
      </div>
      <p class="text-text-main font-medium mb-1">No combos yet</p>
      <p class="text-sm text-text-muted mb-4">Create model combos with fallback support</p>
      <button
        type="button"
        onclick={openCreateModal}
        class="inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-medium transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>Create Combo</span>
      </button>
    </div>
  {:else}
    <div class="flex flex-col gap-3">
      {#each combos as combo (combo.id)}
        {@const modelsList = getComboModels(combo)}
        {@const strategyInfo = comboStrategies[combo.name] || {}}
        {@const currentStrategy = strategyInfo.fallbackStrategy || combo.strategy || 'fallback'}
        {@const judgeModel = strategyInfo.judgeModel || ''}
        {@const isFusion = currentStrategy === 'fusion'}

        <div
          class="group rounded-xl border border-border bg-surface hover:border-brand-500/30 p-3.5 transition-all shadow-xs"
        >
          <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <!-- Left: Icon, Name, Model Badges & Fusion Judge -->
            <div class="flex min-w-0 flex-1 items-start gap-3 sm:items-center">
              <div class="size-8 rounded-lg bg-brand-500/10 flex items-center justify-center shrink-0">
                <Layers class="w-4 h-4 text-brand-500" />
              </div>
              <div class="min-w-0 flex-1">
                <code class="block truncate font-mono text-sm font-medium text-text-main">{combo.name}</code>
                <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1">
                  {#if modelsList.length === 0}
                    <span class="text-xs text-text-muted italic">No models</span>
                  {:else}
                    {#each modelsList.slice(0, 3) as model}
                      <code
                        class="inline-flex items-center gap-1 rounded bg-black/5 dark:bg-white/5 px-1.5 py-0.5 font-mono text-xs text-text-muted"
                      >
                        <span>{model}</span>
                        {#if hasVision(model)}
                          <Eye class="w-3 h-3 text-blue-500 shrink-0" title="Vision — Supports image input" />
                        {/if}
                        {#if hasReasoning(model)}
                          <Sparkles class="w-3 h-3 text-amber-500 shrink-0" title="Reasoning — Supports reasoning / thinking" />
                        {/if}
                      </code>
                    {/each}
                    {#if modelsList.length > 3}
                      <span class="text-[10px] text-text-muted">+{modelsList.length - 3} more</span>
                    {/if}
                  {/if}
                </div>

                <!-- Fusion: judge picker (Auto = first model) -->
                {#if isFusion}
                  <div class="mt-2 flex min-w-0 flex-wrap items-center gap-1.5">
                    <span class="text-[11px] font-medium text-text-muted">Judge</span>
                    <button
                      type="button"
                      onclick={() => {
                        editingCombo = combo
                        openModelPicker('judge')
                      }}
                      class="inline-flex max-w-full items-center gap-1 rounded border border-dashed border-brand-500/40 px-1.5 py-0.5 font-mono text-[11px] text-brand-500 hover:border-brand-500 hover:bg-brand-500/5 transition-colors cursor-pointer"
                      title="Pick the model that fuses panel answers"
                    >
                      <Gavel class="w-3 h-3" />
                      <span class="truncate">{judgeModel || `Auto — ${modelsList[0] || 'first model'}`}</span>
                    </button>
                    {#if judgeModel}
                      <button
                        type="button"
                        onclick={() => clearJudge(combo.name)}
                        class="p-0.5 rounded text-text-muted hover:text-red-500 hover:bg-red-500/10 transition-colors cursor-pointer"
                        title="Reset judge to Auto"
                      >
                        <X class="w-3 h-3" />
                      </button>
                    {/if}
                  </div>
                {/if}
              </div>
            </div>

            <!-- Actions: Strategy selector + Copy/Edit/Delete -->
            <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center sm:gap-3 sm:shrink-0">
              <!-- Strategy dropdown -->
              <div class="w-full sm:w-[190px]">
                <select
                  value={currentStrategy}
                  onchange={(e) => handleSetStrategy(combo, e.currentTarget.value)}
                  class="w-full bg-surface-2 border border-border rounded-lg px-2.5 py-1.5 text-xs text-text-main focus:outline-none focus:border-brand-500 cursor-pointer font-body"
                >
                  <option value="fallback">Fallback — try in order</option>
                  <option value="round-robin">Round Robin — rotate</option>
                  <option value="fusion">Fusion — panel + judge</option>
                </select>
              </div>

              <!-- Icon buttons with labels matching upstream -->
              <div class="grid grid-cols-3 gap-1 sm:flex sm:items-center">
                <button
                  type="button"
                  onclick={() => copyName(combo.name, combo.id)}
                  class="flex flex-col items-center justify-center rounded px-2.5 py-1 text-text-muted transition-colors hover:bg-black/5 dark:hover:bg-white/5 hover:text-brand-500 cursor-pointer"
                  title="Copy combo name"
                >
                  {#if copiedId === combo.id}
                    <Check class="w-4 h-4 text-success" />
                    <span class="text-[10px] leading-tight text-success font-medium">Copied</span>
                  {:else}
                    <Copy class="w-4 h-4" />
                    <span class="text-[10px] leading-tight">Copy</span>
                  {/if}
                </button>

                <button
                  type="button"
                  onclick={() => openEditModal(combo)}
                  class="flex flex-col items-center justify-center rounded px-2.5 py-1 text-text-muted transition-colors hover:bg-black/5 dark:hover:bg-white/5 hover:text-brand-500 cursor-pointer"
                  title="Edit"
                >
                  <Pencil class="w-4 h-4" />
                  <span class="text-[10px] leading-tight">Edit</span>
                </button>

                <button
                  type="button"
                  onclick={() => confirmDeleteCombo(combo)}
                  class="flex flex-col items-center justify-center rounded px-2.5 py-1 text-red-500 transition-colors hover:bg-red-500/10 cursor-pointer"
                  title="Delete"
                >
                  <Trash2 class="w-4 h-4" />
                  <span class="text-[10px] leading-tight">Delete</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Vision Adapter Section (Capacity Adapter) -->
  <div class="flex flex-col gap-3 pt-2">
    <div class="flex flex-col gap-1">
      <h3 class="text-sm font-medium text-text-main">Vision Adapter</h3>
      <p class="text-xs text-text-muted">
        Your model can't read image/audio? Auto-switches to a model in the pool below.
      </p>
      <ul class="text-[11px] text-text-muted flex flex-col gap-0.5 mt-0.5">
        <li><span class="font-medium text-text-main">Vision</span> — images (png, jpg, webp, …)</li>
        <li><span class="font-medium text-text-main">Audio</span> — audio input</li>
      </ul>
    </div>

    <div class="flex flex-col gap-3">
      <!-- Vision Adapter Card -->
      <div
        class="group rounded-xl border border-border bg-surface p-3.5 transition-all shadow-xs {!capacityAdapter.vision
          .enabled
          ? 'opacity-50'
          : ''}"
      >
        <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 flex-1 items-start gap-3 sm:items-center">
            <!-- Toggle switch -->
            <Toggle
              checked={capacityAdapter.vision.enabled}
              onChange={(v) => toggleAdapter('vision', v)}
            />

            <!-- Icon -->
            <div class="size-8 rounded-lg bg-brand-500/10 flex items-center justify-center shrink-0">
              <Eye class="w-4 h-4 text-brand-500" />
            </div>

            <!-- Text & Model Chips -->
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1.5">
                <code class="font-mono text-sm font-medium text-text-main">Vision</code>
                <span class="text-[10px] text-text-muted">— Images</span>
              </div>
              <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1">
                {#if capacityAdapter.vision.models.length === 0}
                  <span class="text-xs text-text-muted italic">No models</span>
                {:else}
                  {#each capacityAdapter.vision.models.slice(0, 4) as model, idx}
                    <code
                      class="group/chip inline-flex items-center gap-1 rounded bg-black/5 dark:bg-white/5 px-1.5 py-0.5 font-mono text-xs text-text-muted"
                    >
                      <span>{model}</span>
                      <Eye class="w-3 h-3 text-blue-500 shrink-0" title="Vision — Supports image input" />
                      <button
                        type="button"
                        onclick={() => moveAdapterModel('vision', idx, -1)}
                        disabled={idx === 0}
                        class="leading-none opacity-0 group-hover/chip:opacity-100 {idx === 0
                          ? 'text-text-muted/20 cursor-not-allowed'
                          : 'text-text-muted hover:text-brand-500'} cursor-pointer"
                        title="Move up"
                      >
                        <ArrowUp class="w-3 h-3" />
                      </button>
                      <button
                        type="button"
                        onclick={() => moveAdapterModel('vision', idx, 1)}
                        disabled={idx === capacityAdapter.vision.models.length - 1}
                        class="leading-none opacity-0 group-hover/chip:opacity-100 {idx ===
                        capacityAdapter.vision.models.length - 1
                          ? 'text-text-muted/20 cursor-not-allowed'
                          : 'text-text-muted hover:text-brand-500'} cursor-pointer"
                        title="Move down"
                      >
                        <ArrowDown class="w-3 h-3" />
                      </button>
                      <button
                        type="button"
                        onclick={() => removeAdapterModel('vision', idx)}
                        class="leading-none opacity-0 group-hover/chip:opacity-100 text-text-muted hover:text-red-500 cursor-pointer"
                        title="Remove model"
                      >
                        <X class="w-3 h-3" />
                      </button>
                    </code>
                  {/each}
                  {#if capacityAdapter.vision.models.length > 4}
                    <span class="text-[10px] text-text-muted">+{capacityAdapter.vision.models.length - 4} more</span>
                  {/if}
                {/if}
              </div>
            </div>
          </div>

          <!-- Actions: Round-robin toggle + Add Model -->
          <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center sm:gap-3 sm:shrink-0">
            <label class="flex items-center gap-1.5 text-xs text-text-muted cursor-pointer select-none">
              <Toggle
                checked={capacityAdapter.vision.roundRobin}
                disabled={!capacityAdapter.vision.enabled}
                onChange={(v) => toggleAdapterRoundRobin('vision', v)}
              />
              <span>Round</span>
            </label>
            <button
              type="button"
              onclick={() => openModelPicker('vision')}
              disabled={!capacityAdapter.vision.enabled}
              class="inline-flex items-center justify-center gap-1 px-2.5 py-1 text-xs font-medium text-brand-500 hover:bg-brand-500/10 rounded-lg transition-colors cursor-pointer disabled:opacity-40 disabled:pointer-events-none"
            >
              <Plus class="w-3.5 h-3.5" />
              <span>Add Model</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Audio Adapter Card -->
      <div
        class="group rounded-xl border border-border bg-surface p-3.5 transition-all shadow-xs {!capacityAdapter
          .audioInput.enabled
          ? 'opacity-50'
          : ''}"
      >
        <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 flex-1 items-start gap-3 sm:items-center">
            <!-- Toggle switch -->
            <Toggle
              checked={capacityAdapter.audioInput.enabled}
              onChange={(v) => toggleAdapter('audioInput', v)}
            />

            <!-- Icon -->
            <div class="size-8 rounded-lg bg-brand-500/10 flex items-center justify-center shrink-0">
              <Headphones class="w-4 h-4 text-brand-500" />
            </div>

            <!-- Text & Model Chips -->
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1.5">
                <code class="font-mono text-sm font-medium text-text-main">Audio</code>
                <span class="text-[10px] text-text-muted">— Audio input</span>
              </div>
              <div class="mt-1 flex min-w-0 flex-wrap items-center gap-1">
                {#if capacityAdapter.audioInput.models.length === 0}
                  <span class="text-xs text-text-muted italic">No models</span>
                {:else}
                  {#each capacityAdapter.audioInput.models.slice(0, 4) as model, idx}
                    <code
                      class="group/chip inline-flex items-center gap-1 rounded bg-black/5 dark:bg-white/5 px-1.5 py-0.5 font-mono text-xs text-text-muted"
                    >
                      <span>{model}</span>
                      <button
                        type="button"
                        onclick={() => moveAdapterModel('audioInput', idx, -1)}
                        disabled={idx === 0}
                        class="leading-none opacity-0 group-hover/chip:opacity-100 {idx === 0
                          ? 'text-text-muted/20 cursor-not-allowed'
                          : 'text-text-muted hover:text-brand-500'} cursor-pointer"
                        title="Move up"
                      >
                        <ArrowUp class="w-3 h-3" />
                      </button>
                      <button
                        type="button"
                        onclick={() => moveAdapterModel('audioInput', idx, 1)}
                        disabled={idx === capacityAdapter.audioInput.models.length - 1}
                        class="leading-none opacity-0 group-hover/chip:opacity-100 {idx ===
                        capacityAdapter.audioInput.models.length - 1
                          ? 'text-text-muted/20 cursor-not-allowed'
                          : 'text-text-muted hover:text-brand-500'} cursor-pointer"
                        title="Move down"
                      >
                        <ArrowDown class="w-3 h-3" />
                      </button>
                      <button
                        type="button"
                        onclick={() => removeAdapterModel('audioInput', idx)}
                        class="leading-none opacity-0 group-hover/chip:opacity-100 text-text-muted hover:text-red-500 cursor-pointer"
                        title="Remove model"
                      >
                        <X class="w-3 h-3" />
                      </button>
                    </code>
                  {/each}
                  {#if capacityAdapter.audioInput.models.length > 4}
                    <span class="text-[10px] text-text-muted">+{capacityAdapter.audioInput.models.length - 4} more</span>
                  {/if}
                {/if}
              </div>
            </div>
          </div>

          <!-- Actions: Round-robin toggle + Add Model -->
          <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center sm:gap-3 sm:shrink-0">
            <label class="flex items-center gap-1.5 text-xs text-text-muted cursor-pointer select-none">
              <Toggle
                checked={capacityAdapter.audioInput.roundRobin}
                disabled={!capacityAdapter.audioInput.enabled}
                onChange={(v) => toggleAdapterRoundRobin('audioInput', v)}
              />
              <span>Round</span>
            </label>
            <button
              type="button"
              onclick={() => openModelPicker('audio')}
              disabled={!capacityAdapter.audioInput.enabled}
              class="inline-flex items-center justify-center gap-1 px-2.5 py-1 text-xs font-medium text-brand-500 hover:bg-brand-500/10 rounded-lg transition-colors cursor-pointer disabled:opacity-40 disabled:pointer-events-none"
            >
              <Plus class="w-3.5 h-3.5" />
              <span>Add Model</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- Create / Edit Combo Modal -->
{#if isCreatingOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div
      class="bg-surface border border-border rounded-xl shadow-2xl max-w-lg w-full overflow-hidden flex flex-col max-h-[90vh]"
    >
      <div class="px-5 py-4 border-b border-border flex items-center justify-between">
        <h2 class="text-sm font-semibold text-text-main">
          {editingCombo ? 'Edit Combo' : 'Create Combo'}
        </h2>
        <button type="button" onclick={closeModal} class="text-text-muted hover:text-text-main cursor-pointer">
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-5 overflow-y-auto space-y-4 flex-1">
        <!-- Combo Name -->
        <div>
          <label for="comboName" class="text-xs font-medium text-text-main block mb-1">Combo Name</label>
          <input
            id="comboName"
            type="text"
            bind:value={modalName}
            oninput={() => validateModalName(modalName)}
            placeholder="my-combo"
            class="w-full bg-surface-2 border {modalNameError
              ? 'border-red-500'
              : 'border-border'} rounded-lg px-3 py-2 text-xs text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
          {#if modalNameError}
            <p class="text-[10px] text-red-500 mt-1">{modalNameError}</p>
          {:else}
            <p class="text-[10px] text-text-muted mt-0.5">Only letters, numbers, -, _ and . allowed</p>
          {/if}
        </div>

        <!-- Models -->
        <div>
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-xs font-medium text-text-main">Models ({modalModels.length})</span>
          </div>

          {#if modalModels.length === 0}
            <div
              class="text-center py-6 border border-dashed border-border rounded-lg bg-black/[0.01] dark:bg-white/[0.01]"
            >
              <Layers class="w-6 h-6 text-text-muted mx-auto mb-1 opacity-50" />
              <p class="text-xs text-text-muted">No models added yet</p>
            </div>
          {:else}
            <div class="flex flex-col gap-1.5 max-h-[220px] overflow-y-auto pr-1">
              {#each modalModels as model, idx}
                <div
                  class="group flex min-w-0 items-center gap-2 rounded-lg px-2.5 py-1.5 bg-surface-2 border border-border/70 hover:border-brand-500/40 transition-colors"
                >
                  <GripVertical class="w-3.5 h-3.5 text-text-muted cursor-grab shrink-0" />
                  <span class="text-[10px] font-medium text-text-muted w-3 text-center shrink-0">{idx + 1}</span>
                  <div class="min-w-0 flex-1 truncate font-mono text-xs text-text-main">
                    {model}
                  </div>
                  <div class="flex items-center gap-0.5 shrink-0">
                    <button
                      type="button"
                      disabled={idx === 0}
                      onclick={() => {
                        const arr = [...modalModels]
                        const t = arr[idx]
                        arr[idx] = arr[idx - 1]
                        arr[idx - 1] = t
                        modalModels = arr
                      }}
                      class="p-1 rounded text-text-muted hover:text-brand-500 hover:bg-black/5 dark:hover:bg-white/5 disabled:opacity-20 cursor-pointer"
                      title="Move up"
                    >
                      <ArrowUp class="w-3 h-3" />
                    </button>
                    <button
                      type="button"
                      disabled={idx === modalModels.length - 1}
                      onclick={() => {
                        const arr = [...modalModels]
                        const t = arr[idx]
                        arr[idx] = arr[idx + 1]
                        arr[idx + 1] = t
                        modalModels = arr
                      }}
                      class="p-1 rounded text-text-muted hover:text-brand-500 hover:bg-black/5 dark:hover:bg-white/5 disabled:opacity-20 cursor-pointer"
                      title="Move down"
                    >
                      <ArrowDown class="w-3 h-3" />
                    </button>
                    <button
                      type="button"
                      onclick={() => {
                        modalModels = modalModels.filter((_, i) => i !== idx)
                      }}
                      class="p-1 rounded text-text-muted hover:text-red-500 hover:bg-red-500/10 cursor-pointer"
                      title="Remove"
                    >
                      <X class="w-3 h-3" />
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}

          <!-- Add Model button -->
          <button
            type="button"
            onclick={() => openModelPicker('combo')}
            class="w-full mt-2 py-2 border border-dashed border-border hover:border-brand-500/50 rounded-lg text-xs text-brand-500 font-medium hover:bg-brand-500/5 transition-colors flex items-center justify-center gap-1 cursor-pointer"
          >
            <Plus class="w-4 h-4" />
            <span>Add Model</span>
          </button>
        </div>
      </div>

      <div class="px-5 py-3 border-t border-border flex items-center justify-end gap-2 bg-surface-2/50">
        <button
          type="button"
          onclick={closeModal}
          class="px-3.5 py-1.5 text-xs text-text-muted hover:text-text-main font-medium rounded-lg cursor-pointer transition"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleSaveCombo}
          disabled={!modalName.trim() || !!modalNameError || isSavingCombo}
          class="px-4 py-1.5 text-xs bg-brand-500 hover:bg-brand-600 text-white font-medium rounded-lg shadow-sm transition disabled:opacity-50 disabled:pointer-events-none cursor-pointer flex items-center gap-1.5"
        >
          {#if isSavingCombo}
            <Loader2 class="w-3.5 h-3.5 animate-spin" />
          {/if}
          <span>{editingCombo ? 'Save' : 'Create'}</span>
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Model Selector Modal -->
{#if showModelPicker}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div
      class="bg-surface border border-border rounded-xl shadow-2xl max-w-md w-full overflow-hidden flex flex-col max-h-[85vh]"
    >
      <div class="px-5 py-4 border-b border-border flex items-center justify-between">
        <h2 class="text-sm font-semibold text-text-main">
          {modelPickerTarget === 'vision'
            ? 'Add Vision Model'
            : modelPickerTarget === 'audio'
              ? 'Add Audio Model'
              : modelPickerTarget === 'judge'
                ? 'Select Judge Model'
                : 'Add Model to Combo'}
        </h2>
        <button
          type="button"
          onclick={() => (showModelPicker = false)}
          class="text-text-muted hover:text-text-main cursor-pointer"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-4 border-b border-border space-y-2">
        <div class="relative">
          <Search class="w-3.5 h-3.5 absolute left-3 top-2.5 text-text-muted pointer-events-none" />
          <input
            type="text"
            bind:value={modelPickerSearch}
            placeholder="Search active models..."
            class="w-full bg-surface-2 border border-border rounded-lg pl-9 pr-3 py-1.5 text-xs text-text-main placeholder:text-text-muted focus:outline-none focus:border-brand-500"
          />
        </div>

        <!-- Or custom model string -->
        <div class="flex items-center gap-1.5">
          <input
            type="text"
            bind:value={customModelInput}
            placeholder="Or type custom model ID..."
            class="flex-1 bg-surface-2 border border-border rounded-lg px-2.5 py-1 text-xs text-text-main placeholder:text-text-muted font-mono focus:outline-none focus:border-brand-500"
          />
          <button
            type="button"
            disabled={!customModelInput.trim()}
            onclick={() => {
              if (customModelInput.trim()) {
                selectModel(customModelInput.trim())
              }
            }}
            class="px-2.5 py-1 bg-brand-500 hover:bg-brand-600 text-white text-xs font-medium rounded-lg disabled:opacity-40 cursor-pointer shrink-0"
          >
            Add
          </button>
        </div>
      </div>

      <!-- Models List -->
      <div class="p-2 overflow-y-auto flex-1 max-h-[350px] space-y-1">
        {#if filteredPickerModels().length === 0}
          <div class="text-center py-8 text-xs text-text-muted">
            No matching models found. You can type a custom model above.
          </div>
        {:else}
          {#each filteredPickerModels() as item}
            <button
              type="button"
              onclick={() => selectModel(item.value)}
              class="w-full text-left flex items-center justify-between px-3 py-2 rounded-lg hover:bg-surface-2 transition-colors cursor-pointer border border-transparent hover:border-border"
            >
              <div class="min-w-0 flex-1 pr-2">
                <div class="flex items-center gap-1.5">
                  <span class="font-mono text-xs font-medium text-text-main truncate">{item.value}</span>
                  {#if item.vision}
                    <span
                      class="material-symbols-outlined text-[13px] text-blue-500 leading-none"
                      title="Vision — Supports image input">visibility</span
                    >
                  {/if}
                  {#if item.reasoning}
                    <span
                      class="material-symbols-outlined text-[13px] text-amber-500 leading-none"
                      title="Reasoning — Supports reasoning / thinking">neurology</span
                    >
                  {/if}
                </div>
                <span class="text-[10px] text-text-muted">{item.provider}</span>
              </div>
              <Plus class="w-4 h-4 text-text-muted hover:text-brand-500 shrink-0" />
            </button>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Confirm Delete Modal -->
{#if deletingCombo}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="bg-surface border border-border rounded-xl shadow-2xl max-w-sm w-full p-5 space-y-4">
      <div class="flex items-center gap-3">
        <div class="size-9 rounded-full bg-red-500/10 text-red-500 flex items-center justify-center shrink-0">
          <Trash2 class="w-4 h-4" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-text-main">Delete Combo</h3>
          <p class="text-xs text-text-muted mt-0.5">Are you sure you want to delete this combo?</p>
        </div>
      </div>

      <p class="font-mono text-xs bg-surface-2 p-2 rounded border border-border text-text-main truncate">
        {deletingCombo.name}
      </p>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button
          type="button"
          onclick={() => (deletingCombo = null)}
          class="px-3.5 py-1.5 text-xs text-text-muted hover:text-text-main font-medium rounded-lg cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleDeleteCombo}
          class="px-4 py-1.5 text-xs bg-red-500 hover:bg-red-600 text-white font-medium rounded-lg shadow-sm transition cursor-pointer"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}
