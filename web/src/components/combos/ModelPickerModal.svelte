<script lang="ts">
  import { Brain, Eye, Plus, Search, X } from 'lucide-svelte'
  import type { PickerModelItem } from './types'

  interface Props {
    isOpen: boolean
    target: 'combo' | 'vision' | 'audio' | 'judge'
    availableModels: PickerModelItem[]
    onSelect: (modelValue: string) => void
    onClose: () => void
  }

  let {
    isOpen,
    target,
    availableModels = [],
    onSelect,
    onClose,
  }: Props = $props()

  let search = $state('')
  let customModelInput = $state('')

  $effect(() => {
    if (isOpen) {
      search = ''
      customModelInput = ''
    }
  })

  let filteredModels = $derived(() => {
    return availableModels.filter((item) => {
      if (target === 'vision' && !item.vision) {
        if (!search.trim()) return item.vision
      }
      if (search.trim()) {
        const q = search.toLowerCase()
        return item.value.toLowerCase().includes(q) || item.provider.toLowerCase().includes(q)
      }
      return true
    })
  })

  function handleCustomAdd() {
    const val = customModelInput.trim()
    if (val) {
      onSelect(val)
    }
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="bg-surface border border-border rounded-xl shadow-2xl max-w-md w-full overflow-hidden flex flex-col max-h-[85vh]">
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
        <button type="button" onclick={onClose} class="text-text-muted hover:text-text-main cursor-pointer">
          <X class="w-4 h-4" />
        </button>
      </div>

      <div class="p-4 border-b border-border space-y-2">
        <div class="relative">
          <Search class="w-3.5 h-3.5 absolute left-3 top-2.5 text-text-muted pointer-events-none" />
          <input
            type="text"
            bind:value={search}
            placeholder="Search active models..."
            class="w-full bg-surface-2 border border-border rounded-lg pl-9 pr-3 py-1.5 text-xs text-text-main placeholder:text-text-muted focus:outline-none focus:border-brand-500"
          />
        </div>

        <!-- Custom model string -->
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
            onclick={handleCustomAdd}
            class="px-2.5 py-1 bg-brand-500 hover:bg-brand-600 text-white text-xs font-medium rounded-lg disabled:opacity-40 cursor-pointer shrink-0"
          >
            Add
          </button>
        </div>
      </div>

      <!-- Models List -->
      <div class="p-2 overflow-y-auto flex-1 max-h-[350px] space-y-1">
        {#if filteredModels().length === 0}
          <div class="text-center py-8 text-xs text-text-muted">
            No matching models found. You can type a custom model above.
          </div>
        {:else}
          {#each filteredModels() as item}
            <button
              type="button"
              onclick={() => onSelect(item.value)}
              class="w-full text-left flex items-center justify-between px-3 py-2 rounded-lg hover:bg-surface-2 transition-colors cursor-pointer border border-transparent hover:border-border"
            >
              <div class="min-w-0 flex-1 pr-2">
                <div class="flex items-center gap-1.5">
                  <span class="font-mono text-xs font-medium text-text-main truncate">{item.value}</span>
                  {#if item.vision}
                    <Eye class="w-3.5 h-3.5 text-blue-500 shrink-0" title="Vision — Supports image input" />
                  {/if}
                  {#if item.reasoning}
                    <Brain class="w-3.5 h-3.5 text-amber-500 shrink-0" title="Reasoning / Neuron — Supports thinking" />
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
