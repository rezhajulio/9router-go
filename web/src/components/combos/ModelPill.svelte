<script lang="ts">
  import { Brain, Check, Eye } from 'lucide-svelte'

  interface Props {
    label: string
    value: string
    isAdded: boolean
    caps?: { vision?: boolean; reasoning?: boolean }
    onClick: () => void
  }

  let { label, value, isAdded, caps, onClick }: Props = $props()
</script>

<button
  type="button"
  {value}
  onclick={onClick}
  class={`px-2 py-1 rounded-xl text-xs font-medium transition-all border hover:cursor-pointer flex items-center gap-1 ${
    isAdded
      ? 'bg-brand-500 text-white border-brand-500 hover:bg-brand-600'
      : 'bg-surface border-border text-text-main hover:border-brand-500/50 hover:bg-brand-500/5'
  }`}
>
  {#if isAdded}
    <Check class="w-3 h-3 shrink-0" />
  {/if}
  <span class="truncate">{label}</span>
  {#if caps?.vision}
    <Eye
      class={isAdded ? 'w-3 h-3 text-white/90 shrink-0' : 'w-3 h-3 text-blue-500 shrink-0'}
      title="Vision — Supports image input"
    />
  {/if}
  {#if caps?.reasoning}
    <Brain
      class={isAdded ? 'w-3 h-3 text-white/90 shrink-0' : 'w-3 h-3 text-amber-500 shrink-0'}
      title="Reasoning / Thinking"
    />
  {/if}
</button>
