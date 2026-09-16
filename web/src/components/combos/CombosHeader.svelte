<script lang="ts">
  import { Plus } from 'lucide-svelte'

  interface Props {
    onCreateClick: () => void
  }

  let { onCreateClick }: Props = $props()
</script>

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
    onclick={onCreateClick}
    class="inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white text-xs font-medium shadow-sm transition-colors cursor-pointer w-full sm:w-auto shrink-0"
  >
    <Plus class="w-4 h-4" />
    <span>Create Combo</span>
  </button>
</div>
