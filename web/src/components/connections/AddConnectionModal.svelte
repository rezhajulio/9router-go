<script lang="ts">
  import { X } from 'lucide-svelte'
  import Button from '../../lib/ui/Button.svelte'

  interface Props {
    isOpen: boolean
    title: string
    isSubmitting?: boolean
    onClose: () => void
    onSubmit: (keyName: string, apiKey: string) => Promise<void> | void
  }

  let {
    isOpen,
    title,
    isSubmitting = false,
    onClose,
    onSubmit,
  }: Props = $props()

  let formKeyName = $state('')
  let formApiKey = $state('')

  $effect(() => {
    if (isOpen) {
      formKeyName = ''
      formApiKey = ''
    }
  })

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault()
    if (!formApiKey.trim()) return
    onSubmit(formKeyName.trim(), formApiKey.trim())
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="w-full max-w-md bg-surface border border-border rounded-2xl shadow-2xl overflow-hidden animate-scale-in">
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="font-bold text-text-main text-base">
          {title}
        </h3>
        <button type="button" onclick={onClose} class="text-text-muted hover:text-text-main cursor-pointer">
          <X class="w-5 h-5" />
        </button>
      </div>

      <form onsubmit={handleSubmit} class="p-6 flex flex-col gap-4">
        <div>
          <label for="connLabel" class="block text-xs font-medium text-text-muted mb-1">Connection Label (Optional)</label>
          <input
            id="connLabel"
            type="text"
            bind:value={formKeyName}
            placeholder="e.g. Secondary Account / Prod"
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
          />
        </div>

        <div>
          <label for="connApiKey" class="block text-xs font-medium text-text-muted mb-1">API Key / Token</label>
          <input
            id="connApiKey"
            type="password"
            required
            bind:value={formApiKey}
            placeholder="sk-..."
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
        </div>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button size="sm" variant="ghost" onclick={onClose}>Cancel</Button>
          <Button size="sm" variant="primary" type="submit" disabled={isSubmitting || !formApiKey.trim()}>
            {isSubmitting ? 'Saving...' : 'Save Connection'}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}
