<script lang="ts">
  import { Trash2 } from 'lucide-svelte'
  import type { Combo } from '../../api/client'

  interface Props {
    combo: Combo | null
    onClose: () => void
    onConfirm: () => void
  }

  let { combo, onClose, onConfirm }: Props = $props()
</script>

{#if combo}
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
        {combo.name}
      </p>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button
          type="button"
          onclick={onClose}
          class="px-3.5 py-1.5 text-xs text-text-muted hover:text-text-main font-medium rounded-lg cursor-pointer"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={onConfirm}
          class="px-4 py-1.5 text-xs bg-red-500 hover:bg-red-600 text-white font-medium rounded-lg shadow-sm transition cursor-pointer"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}
