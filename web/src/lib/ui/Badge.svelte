<script lang="ts">
  // Port of decolua/9router src/shared/components/Badge.js
  import type { Snippet } from 'svelte'

  type Tone = 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info'

  let {
    tone = 'default',
    class: klass = '',
    children
  }: {
    tone?: Tone
    class?: string
    children?: Snippet
  } = $props()

  const tones: Record<Tone, string> = {
    default: 'bg-surface-2 text-text-muted border border-border',
    primary: 'bg-brand-500/10 text-brand-600 dark:text-brand-400 border border-brand-500/25',
    success: 'bg-success/10 text-success border border-success/25',
    warning: 'bg-warning/10 text-warning border border-warning/25',
    danger: 'bg-danger/10 text-danger border border-danger/25',
    info: 'bg-info/10 text-info border border-info/25',
  }
</script>

<span
  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium {tones[tone]} {klass}"
>
  {@render children?.()}
</span>
