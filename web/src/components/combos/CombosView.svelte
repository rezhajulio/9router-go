<script lang="ts">
  import { Layers, Plus } from 'lucide-svelte'
  import { api, type Combo, type ProviderConnection, type ProviderNode } from '../../api/client'
  import {
    clearJudgeModel,
    computeAvailableModels,
    getComboModels,
    parseCapacityAdapterSettings,
    updateComboStrategy,
    updateJudgeModel,
    type CapacityAdapterState,
    type ComboStrategyInfo
  } from './types'
  import ComboCard from './ComboCard.svelte'
  import CombosHeader from './CombosHeader.svelte'
  import CreateComboModal from './CreateComboModal.svelte'
  import ModelPickerModal from './ModelPickerModal.svelte'
  import CapacityAdapterSection from './CapacityAdapterSection.svelte'
  import DeleteComboModal from './DeleteComboModal.svelte'

  interface Props {
    combos?: Combo[]
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
    onRefresh: () => void
    isCreatingOpen?: boolean
  }

  let {
    combos = [],
    connections = [],
    providerNodes = [],
    onRefresh,
    isCreatingOpen = $bindable(false),
  }: Props = $props()

  let comboStrategies = $state<Record<string, ComboStrategyInfo>>({})
  let capacityAdapter = $state<CapacityAdapterState>({
    vision: { enabled: true, roundRobin: false, models: ['ag/gemini-3.7-flash-high'] },
    audioInput: { enabled: true, roundRobin: false, models: [] },
  })
  let copiedId = $state<string | null>(null)

  // Edit / Create Modal state
  let editingCombo = $state<Combo | null>(null)
  let modalModels = $state<string[]>([])
  let isSavingCombo = $state(false)

  // Model Picker Modal state
  let showModelPicker = $state(false)
  let modelPickerTarget = $state<'combo' | 'vision' | 'audio' | 'judge'>('combo')

  // Confirm Delete Modal state
  let deletingCombo = $state<Combo | null>(null)

  async function loadSettings() {
    try {
      const s = await api.getSettings()
      if (s?.comboStrategies && typeof s.comboStrategies === 'object') {
        comboStrategies = s.comboStrategies as Record<string, ComboStrategyInfo>
      }
      if (s?.capacityAdapter && typeof s.capacityAdapter === 'object') {
        capacityAdapter = parseCapacityAdapterSettings(s.capacityAdapter as Record<string, unknown>)
      }
    } catch (e) {
      console.error('Failed to load settings:', e)
    }
  }

  $effect(() => {
    loadSettings()
  })

  $effect(() => {
    if (isCreatingOpen && !editingCombo) {
      modalModels = []
    }
  })

  function copyName(name: string, id: string) {
    navigator.clipboard.writeText(name)
    copiedId = id
    setTimeout(() => {
      if (copiedId === id) copiedId = null
    }, 2000)
  }

  async function handleSetStrategy(combo: Combo, newStrategy: string) {
    const updated = updateComboStrategy(comboStrategies, combo.name, newStrategy)
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
    const updated = updateJudgeModel(comboStrategies, comboName, judgeModel)
    comboStrategies = updated
    try {
      await api.patchSettings({ comboStrategies: updated })
    } catch (e) {
      console.error('Failed to update judge model:', e)
    }
  }

  async function clearJudge(comboName: string) {
    const updated = clearJudgeModel(comboStrategies, comboName)
    comboStrategies = updated
    try {
      await api.patchSettings({ comboStrategies: updated })
    } catch (e) {
      console.error('Failed to clear judge:', e)
    }
  }

  async function saveCapacityAdapter(next: CapacityAdapterState) {
    capacityAdapter = next
    try {
      await api.patchSettings({ capacityAdapter: next })
    } catch (e) {
      console.error('Failed to update capacity adapter:', e)
    }
  }

  function openCreateModal() {
    editingCombo = null
    modalModels = []
    isCreatingOpen = true
  }

  function openEditModal(combo: Combo) {
    editingCombo = combo
    modalModels = [...getComboModels(combo)]
    isCreatingOpen = true
  }

  function closeModal() {
    isCreatingOpen = false
    editingCombo = null
    modalModels = []
  }

  async function handleSaveCombo(name: string, models: string[]) {
    isSavingCombo = true
    try {
      if (editingCombo) {
        await api.updateCombo(editingCombo.id, { name, models })
      } else {
        await api.createCombo({ name, models, strategy: 'fallback' })
      }
      closeModal()
      onRefresh()
    } catch (e) {
      console.error('Failed to save combo:', e)
    } finally {
      isSavingCombo = false
    }
  }

  function openModelPicker(target: 'combo' | 'vision' | 'audio' | 'judge') {
    modelPickerTarget = target
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
            models: [...capacityAdapter.vision.models, modelValue],
          },
        })
      }
    } else if (modelPickerTarget === 'audio') {
      if (!capacityAdapter.audioInput.models.includes(modelValue)) {
        saveCapacityAdapter({
          ...capacityAdapter,
          audioInput: {
            ...capacityAdapter.audioInput,
            models: [...capacityAdapter.audioInput.models, modelValue],
          },
        })
      }
    } else if (modelPickerTarget === 'judge' && editingCombo) {
      handleSetJudge(editingCombo.name, modelValue)
    }
    showModelPicker = false
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

  let availableModels = $derived(computeAvailableModels(providerNodes, combos))
</script>

<div class="flex min-w-0 flex-col gap-6 px-1 sm:px-0">
  <CombosHeader onCreateClick={openCreateModal} />

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
        <ComboCard
          {combo}
          strategyInfo={comboStrategies[combo.name]}
          {copiedId}
          onSetStrategy={handleSetStrategy}
          onOpenJudgePicker={(c) => {
            editingCombo = c
            openModelPicker('judge')
          }}
          onClearJudge={clearJudge}
          onCopy={copyName}
          onEdit={openEditModal}
          onDelete={(c) => (deletingCombo = c)}
        />
      {/each}
    </div>
  {/if}

  <!-- Vision / Audio Adapter Section -->
  <CapacityAdapterSection
    {capacityAdapter}
    onSaveAdapter={saveCapacityAdapter}
    onOpenModelPicker={openModelPicker}
  />
</div>

<!-- Create / Edit Combo Modal -->
<CreateComboModal
  isOpen={isCreatingOpen}
  {editingCombo}
  models={modalModels}
  isSaving={isSavingCombo}
  onClose={closeModal}
  onSave={handleSaveCombo}
  onOpenModelPicker={() => openModelPicker('combo')}
  onUpdateModels={(newModels) => (modalModels = newModels)}
/>

<!-- Model Picker Modal -->
<ModelPickerModal
  isOpen={showModelPicker}
  target={modelPickerTarget}
  {availableModels}
  onSelect={selectModel}
  onClose={() => (showModelPicker = false)}
/>

<!-- Confirm Delete Modal -->
<DeleteComboModal
  combo={deletingCombo}
  onClose={() => (deletingCombo = null)}
  onConfirm={handleDeleteCombo}
/>
