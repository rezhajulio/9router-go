<script lang="ts">
  import { Info, Layers, Search, X } from 'lucide-svelte'
  import type { Combo, ProviderConnection, ProviderNode } from '../../api/client'
  import ModelPill from './ModelPill.svelte'
  import {
    resolveFilteredCombos,
    resolveFilteredGroups,
    resolveModelPickerGroups,
  } from './pickerData'

  interface Props {
    isOpen: boolean
    target: 'combo' | 'vision' | 'audio' | 'judge'
    connections?: ProviderConnection[]
    combos?: Combo[]
    providerNodes?: ProviderNode[]
    currentComboName?: string
    addedModelValues?: string[]
    onSelect: (modelValue: string) => void
    onDeselect?: (modelValue: string) => void
    onClose: () => void
  }

  let {
    isOpen,
    target,
    connections = [],
    combos = [],
    providerNodes = [],
    currentComboName,
    addedModelValues = [],
    onSelect,
    onDeselect,
    onClose,
  }: Props = $props()

  let searchQuery = $state('')
  let customModelInput = $state('')

  $effect(() => {
    if (isOpen) {
      searchQuery = ''
      customModelInput = ''
    }
  })

  let groups = $derived(resolveModelPickerGroups(connections, providerNodes))
  let filteredCombos = $derived(
    resolveFilteredCombos(combos, currentComboName, searchQuery, target)
  )
  let filteredGroups = $derived(
    resolveFilteredGroups(groups, searchQuery, target)
  )

  function handleToggle(val: string) {
    if (addedModelValues.includes(val)) {
      if (onDeselect) {
        onDeselect(val)
      } else {
        onSelect(val)
      }
    } else {
      onSelect(val)
    }
  }

  function handleCustomAdd() {
    const val = customModelInput.trim()
    if (val) {
      handleToggle(val)
      customModelInput = ''
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-60 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="fixed inset-0" onclick={onClose} aria-hidden="true"></div>
    <div
      class="relative bg-surface border border-border rounded-xl shadow-2xl max-w-lg w-full overflow-hidden flex flex-col max-h-[85vh] z-10"
      role="dialog"
      aria-modal="true"
    >
      <!-- Header -->
      <div class="px-5 py-4 border-b border-border flex items-center justify-between">
        <h2 class="text-sm font-semibold text-text-main">
          {target === 'vision'
            ? 'Add Vision Model'
            : target === 'audio'
              ? 'Add Audio Model'
              : target === 'judge'
                ? 'Select Judge Model'
                : 'Add Model to Combo'}
        </h2>
        <button
          type="button"
          onclick={onClose}
          aria-label="Close"
          class="text-text-muted hover:text-text-main cursor-pointer"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <!-- Search & Custom Model Area -->
      <div class="p-4 border-b border-border">
        <!-- Info bar -->
        <div class="flex items-center gap-2 mb-3 px-2.5 py-2 bg-brand-500/10 border border-brand-500/20 rounded-lg text-xs text-text-muted">
          <Info class="w-3.5 h-3.5 text-brand-500 shrink-0" />
          <span>Click to add, click again to remove. Changes are saved automatically.</span>
        </div>

        <!-- Search -->
        <div class="mb-2.5">
          <div class="relative">
            <Search class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
            <input
              type="text"
              placeholder="Search..."
              bind:value={searchQuery}
              class="w-full bg-surface-2 border border-border rounded-lg pl-8 pr-3 py-1.5 text-xs text-text-main placeholder:text-text-muted focus:outline-none focus:border-brand-500"
            />
          </div>
        </div>

        <!-- Custom model input -->
        <div class="flex items-center gap-1.5">
          <input
            type="text"
            bind:value={customModelInput}
            placeholder="Or type custom model ID..."
            class="flex-1 bg-surface-2 border border-border rounded-lg px-2.5 py-1 text-xs text-text-main placeholder:text-text-muted font-mono focus:outline-none focus:border-brand-500"
            onkeydown={(e) => {
              if (e.key === 'Enter') handleCustomAdd()
            }}
          />
          <button
            type="button"
            disabled={!customModelInput.trim()}
            onclick={handleCustomAdd}
            class="px-2.5 py-1 bg-brand-500 hover:bg-brand-600 text-white text-xs font-medium rounded-lg disabled:opacity-40 cursor-pointer shrink-0"
          >
            Add
          </button>
        </div>
      </div>

      <!-- Categories & Models List -->
      <div class="p-4 overflow-y-auto flex-1 max-h-[400px] space-y-3">
        <!-- Combos section - always first -->
        {#if filteredCombos.length > 0}
          <div>
            <div class="flex items-center gap-1.5 mb-1.5 sticky top-0 bg-surface py-0.5 z-10">
              <Layers class="w-3.5 h-3.5 text-brand-500 shrink-0" />
              <span class="text-xs font-medium text-brand-500">Combos</span>
              <span class="text-[10px] text-text-muted">({filteredCombos.length})</span>
            </div>
            <div class="flex flex-wrap gap-1.5">
              {#each filteredCombos as combo (combo.id)}
                <ModelPill
                  label={combo.name}
                  value={combo.name}
                  isAdded={addedModelValues.includes(combo.name)}
                  onClick={() => handleToggle(combo.name)}
                />
              {/each}
            </div>
          </div>
        {/if}

        <!-- Provider sections -->
        {#each filteredGroups as group (group.id)}
          <div>
            <div class="flex items-center gap-1.5 mb-1.5 sticky top-0 bg-surface py-0.5 z-10">
              <span class="text-xs font-medium text-brand-500">
                {group.name}
              </span>
              <span class="text-[10px] text-text-muted">
                ({group.models.length})
              </span>
            </div>
            <div class="flex flex-wrap gap-1.5">
              {#each group.models as model (model.value)}
                <ModelPill
                  label={model.name}
                  value={model.value}
                  isAdded={addedModelValues.includes(model.value)}
                  caps={model.caps}
                  onClick={() => handleToggle(model.value)}
                />
              {/each}
            </div>
          </div>
        {/each}

        {#if filteredCombos.length === 0 && filteredGroups.length === 0}
          <div class="text-center py-6 text-xs text-text-muted">
            No models found. You can type a custom model above.
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-5 py-3 border-t border-border flex items-center justify-end bg-surface-2/50">
        <button
          type="button"
          onclick={onClose}
          class="px-4 py-1.5 text-xs bg-brand-500 hover:bg-brand-600 text-white font-medium rounded-lg shadow-sm transition cursor-pointer"
        >
          Done
        </button>
      </div>
    </div>
  </div>
{/if}
