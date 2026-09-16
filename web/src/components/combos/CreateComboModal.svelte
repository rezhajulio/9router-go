<script lang="ts">
  import {
    ArrowDown,
    ArrowUp,
    Brain,
    Eye,
    GripVertical,
    Layers,
    Loader2,
    Plus,
    X
  } from 'lucide-svelte'
  import type { Combo } from '../../api/client'
  import { hasReasoning, hasVision } from './types'

  interface Props {
    isOpen: boolean
    editingCombo: Combo | null
    models: string[]
    isSaving?: boolean
    onClose: () => void
    onSave: (name: string, models: string[]) => Promise<void> | void
    onOpenModelPicker: () => void
    onUpdateModels: (models: string[]) => void
  }

  let {
    isOpen,
    editingCombo,
    models,
    isSaving = false,
    onClose,
    onSave,
    onOpenModelPicker,
    onUpdateModels,
  }: Props = $props()

  let modalName = $state('')
  let modalNameError = $state('')

  const VALID_NAME_REGEX = /^[a-zA-Z0-9_.\-]+$/

  $effect(() => {
    if (isOpen) {
      if (editingCombo) {
        modalName = editingCombo.name
      } else {
        modalName = ''
      }
      modalNameError = ''
    }
  })

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

  function handleSave() {
    if (!validateModalName(modalName)) return
    onSave(modalName.trim(), models)
  }

  function moveModel(idx: number, delta: number) {
    const arr = [...models]
    const target = idx + delta
    if (target < 0 || target >= arr.length) return
    const temp = arr[idx]
    arr[idx] = arr[target]
    arr[target] = temp
    onUpdateModels(arr)
  }

  function removeModel(idx: number) {
    onUpdateModels(models.filter((_, i) => i !== idx))
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="bg-surface border border-border rounded-xl shadow-2xl max-w-lg w-full overflow-hidden flex flex-col max-h-[90vh]">
      <div class="px-5 py-4 border-b border-border flex items-center justify-between">
        <h2 class="text-sm font-semibold text-text-main">
          {editingCombo ? 'Edit Combo' : 'Create Combo'}
        </h2>
        <button type="button" onclick={onClose} class="text-text-muted hover:text-text-main cursor-pointer">
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
            <span class="text-xs font-medium text-text-main">Models ({models.length})</span>
          </div>

          {#if models.length === 0}
            <div class="text-center py-6 border border-dashed border-border rounded-lg bg-black/[0.01] dark:bg-white/[0.01]">
              <Layers class="w-6 h-6 text-text-muted mx-auto mb-1 opacity-50" />
              <p class="text-xs text-text-muted">No models added yet</p>
            </div>
          {:else}
            <div class="flex flex-col gap-1.5 max-h-[220px] overflow-y-auto pr-1">
              {#each models as model, idx}
                <div class="group flex min-w-0 items-center gap-2 rounded-lg px-2.5 py-1.5 bg-surface-2 border border-border/70 hover:border-brand-500/40 transition-colors">
                  <GripVertical class="w-3.5 h-3.5 text-text-muted cursor-grab shrink-0" />
                  <span class="text-[10px] font-medium text-text-muted w-3 text-center shrink-0">{idx + 1}</span>
                  <div class="min-w-0 flex-1 flex items-center gap-1.5 font-mono text-xs text-text-main truncate">
                    <span class="truncate">{model}</span>
                    {#if hasVision(model)}
                      <Eye class="w-3.5 h-3.5 text-blue-500 shrink-0" title="Vision — Supports image input" />
                    {/if}
                    {#if hasReasoning(model)}
                      <Brain class="w-3.5 h-3.5 text-amber-500 shrink-0" title="Reasoning / Neuron — Supports thinking" />
                    {/if}
                  </div>
                  <div class="flex items-center gap-0.5 shrink-0">
                    <button
                      type="button"
                      disabled={idx === 0}
                      onclick={() => moveModel(idx, -1)}
                      class="p-1 rounded text-text-muted hover:text-brand-500 hover:bg-black/5 dark:hover:bg-white/5 disabled:opacity-20 cursor-pointer"
                      title="Move up"
                    >
                      <ArrowUp class="w-3 h-3" />
                    </button>
                    <button
                      type="button"
                      disabled={idx === models.length - 1}
                      onclick={() => moveModel(idx, 1)}
                      class="p-1 rounded text-text-muted hover:text-brand-500 hover:bg-black/5 dark:hover:bg-white/5 disabled:opacity-20 cursor-pointer"
                      title="Move down"
                    >
                      <ArrowDown class="w-3 h-3" />
                    </button>
                    <button
                      type="button"
                      onclick={() => removeModel(idx)}
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
            onclick={onOpenModelPicker}
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
          onclick={onClose}
          class="px-3.5 py-1.5 text-xs text-text-muted hover:text-text-main font-medium rounded-lg cursor-pointer transition"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={handleSave}
          disabled={!modalName.trim() || !!modalNameError || isSaving}
          class="px-4 py-1.5 text-xs bg-brand-500 hover:bg-brand-600 text-white font-medium rounded-lg shadow-sm transition disabled:opacity-50 disabled:pointer-events-none cursor-pointer flex items-center gap-1.5"
        >
          {#if isSaving}
            <Loader2 class="w-3.5 h-3.5 animate-spin" />
          {/if}
          <span>{editingCombo ? 'Save' : 'Create'}</span>
        </button>
      </div>
    </div>
  </div>
{/if}
