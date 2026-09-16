<script lang="ts">
  import { X } from 'lucide-svelte'
  import Button from '../../lib/ui/Button.svelte'

  interface Props {
    isOpen: boolean
    type: 'openai-compatible' | 'anthropic-compatible'
    isSubmitting?: boolean
    onClose: () => void
    onSubmit: (data: {
      name: string
      prefix: string
      baseUrl: string
      apiType?: 'chat' | 'responses'
      apiKey?: string
      type: string
    }) => Promise<void> | void
  }

  let {
    isOpen,
    type,
    isSubmitting = false,
    onClose,
    onSubmit,
  }: Props = $props()

  let customFormName = $state('')
  let customFormPrefix = $state('')
  let customFormBaseUrl = $state('')
  let customFormApiType = $state<'chat' | 'responses'>('chat')
  let customFormApiKey = $state('')

  let isOpenAI = $derived(type === 'openai-compatible')

  $effect(() => {
    if (isOpen) {
      customFormName = ''
      customFormPrefix = ''
      customFormBaseUrl = isOpenAI ? 'https://api.openai.com/v1' : 'https://api.anthropic.com/v1'
      customFormApiType = 'chat'
      customFormApiKey = ''
    }
  })

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault()
    if (!customFormName.trim() || !customFormPrefix.trim()) return
    onSubmit({
      name: customFormName.trim(),
      prefix: customFormPrefix.trim(),
      baseUrl: customFormBaseUrl.trim(),
      apiType: isOpenAI ? customFormApiType : undefined,
      apiKey: customFormApiKey.trim() || undefined,
      type,
    })
  }
</script>

{#if isOpen}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
    <div class="w-full max-w-md bg-surface border border-border rounded-2xl shadow-2xl overflow-hidden animate-scale-in">
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="font-bold text-text-main text-base">
          {isOpenAI ? 'Add OpenAI Compatible' : 'Add Anthropic Compatible'}
        </h3>
        <button type="button" onclick={onClose} class="text-text-muted hover:text-text-main cursor-pointer">
          <X class="w-5 h-5" />
        </button>
      </div>

      <form onsubmit={handleSubmit} class="p-6 flex flex-col gap-4">
        <div>
          <label for="nodeName" class="block text-xs font-medium text-text-muted mb-1">Name</label>
          <input
            id="nodeName"
            type="text"
            required
            bind:value={customFormName}
            placeholder={isOpenAI ? 'e.g. sumopod, local-vllm' : 'e.g. anthropic-prod, mini-claude'}
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
          />
        </div>

        {#if isOpenAI}
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label for="nodePrefixOAI" class="block text-xs font-medium text-text-muted mb-1">Prefix</label>
              <input
                id="nodePrefixOAI"
                type="text"
                required
                bind:value={customFormPrefix}
                placeholder="e.g. sp, vllm"
                class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
              />
            </div>
            <div>
              <label for="nodeApiType" class="block text-xs font-medium text-text-muted mb-1">API Type</label>
              <select
                id="nodeApiType"
                bind:value={customFormApiType}
                class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
              >
                <option value="chat">Chat Completions</option>
                <option value="responses">Responses API</option>
              </select>
            </div>
          </div>
        {:else}
          <div>
            <label for="nodePrefixAnth" class="block text-xs font-medium text-text-muted mb-1">Prefix</label>
            <input
              id="nodePrefixAnth"
              type="text"
              required
              bind:value={customFormPrefix}
              placeholder="e.g. ac, cl"
              class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main focus:outline-none focus:border-brand-500"
            />
          </div>
        {/if}

        <div>
          <label for="nodeBaseUrl" class="block text-xs font-medium text-text-muted mb-1">Base URL</label>
          <input
            id="nodeBaseUrl"
            type="text"
            required
            bind:value={customFormBaseUrl}
            placeholder={isOpenAI ? 'https://api.openai.com/v1' : 'https://api.anthropic.com/v1'}
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
          <p class="text-[11px] text-text-muted mt-1">
            {isOpenAI ? 'Use the base URL (ending in /v1) for your OpenAI compatible API.' : 'System will append /messages automatically.'}
          </p>
        </div>

        <div>
          <label for="nodeApiKey" class="block text-xs font-medium text-text-muted mb-1">Initial API Key (Optional)</label>
          <input
            id="nodeApiKey"
            type="password"
            bind:value={customFormApiKey}
            placeholder={isOpenAI ? 'sk-...' : 'sk-ant-...'}
            class="w-full px-3 py-2 text-xs rounded-xl bg-bg border border-border text-text-main font-mono focus:outline-none focus:border-brand-500"
          />
        </div>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button size="sm" variant="ghost" onclick={onClose}>Cancel</Button>
          <Button size="sm" variant="primary" type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Creating...' : 'Create Endpoint'}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}
