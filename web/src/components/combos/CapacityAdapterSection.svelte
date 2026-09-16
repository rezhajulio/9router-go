<script lang="ts">
  import { ArrowDown, ArrowUp, Eye, Headphones, Plus, X } from 'lucide-svelte'
  import Toggle from '../../lib/ui/Toggle.svelte'
  import type { CapacityAdapterState } from './types'

  interface Props {
    capacityAdapter: CapacityAdapterState
    onSaveAdapter: (next: CapacityAdapterState) => void
    onOpenModelPicker: (target: 'vision' | 'audio') => void
  }

  let {
    capacityAdapter,
    onSaveAdapter,
    onOpenModelPicker,
  }: Props = $props()

  function toggleAdapter(type: 'vision' | 'audioInput', enabled: boolean) {
    onSaveAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], enabled },
    })
  }

  function toggleAdapterRoundRobin(type: 'vision' | 'audioInput', roundRobin: boolean) {
    onSaveAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], roundRobin },
    })
  }

  function moveAdapterModel(type: 'vision' | 'audioInput', index: number, delta: number) {
    const arr = [...capacityAdapter[type].models]
    const target = index + delta
    if (target < 0 || target >= arr.length) return
    const temp = arr[index]
    arr[index] = arr[target]
    arr[target] = temp
    onSaveAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], models: arr },
    })
  }

  function removeAdapterModel(type: 'vision' | 'audioInput', index: number) {
    const arr = capacityAdapter[type].models.filter((_, i) => i !== index)
    onSaveAdapter({
      ...capacityAdapter,
      [type]: { ...capacityAdapter[type], models: arr },
    })
  }
</script>

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
      class="group rounded-xl border border-border bg-surface p-3.5 transition-all shadow-xs {!capacityAdapter.vision.enabled
        ? 'opacity-50'
        : ''}"
    >
      <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex min-w-0 flex-1 items-start gap-3 sm:items-center">
          <Toggle
            checked={capacityAdapter.vision.enabled}
            onChange={(v) => toggleAdapter('vision', v)}
          />

          <div class="size-8 rounded-lg bg-brand-500/10 flex items-center justify-center shrink-0">
            <Eye class="w-4 h-4 text-brand-500" />
          </div>

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
            onclick={() => onOpenModelPicker('vision')}
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
      class="group rounded-xl border border-border bg-surface p-3.5 transition-all shadow-xs {!capacityAdapter.audioInput.enabled
        ? 'opacity-50'
        : ''}"
    >
      <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex min-w-0 flex-1 items-start gap-3 sm:items-center">
          <Toggle
            checked={capacityAdapter.audioInput.enabled}
            onChange={(v) => toggleAdapter('audioInput', v)}
          />

          <div class="size-8 rounded-lg bg-brand-500/10 flex items-center justify-center shrink-0">
            <Headphones class="w-4 h-4 text-brand-500" />
          </div>

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
            onclick={() => onOpenModelPicker('audio')}
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
