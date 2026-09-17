<script lang="ts">
  import type { ModelItem } from './types'

  interface Props {
    disabledModels: ModelItem[]
    onEnableModel: (modelId: string) => void
  }

  let { disabledModels = [], onEnableModel }: Props = $props()
</script>

{#if disabledModels.length > 0}
  <div class="pt-3 border-t border-border flex flex-col gap-2">
    <div class="text-xs font-medium text-text-muted">
      Disabled models ({disabledModels.length})
    </div>
    <div class="flex flex-wrap gap-1.5">
      {#each disabledModels as m (m.id)}
        <button
          type="button"
          onclick={() => onEnableModel(m.id)}
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
