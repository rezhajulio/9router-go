<script lang="ts">
  import { RotateCcw } from 'lucide-svelte'
  import type { FreebuffSessionStatusResponse } from '../../api/client'

  interface Props {
    session: FreebuffSessionStatusResponse | null
    isLoading: boolean
    expiresInMin: number | null
    onRefresh: () => void
  }

  let { session, isLoading, expiresInMin, onRefresh }: Props = $props()
</script>

{#if session?.status === 'active' && session?.currentModel}
  <div class="mt-4 p-4 rounded-xl border border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-200 text-xs flex items-start gap-3 leading-relaxed shadow-xs">
    <span class="text-base shrink-0 leading-none">🔒</span>
    <div class="flex-1 min-w-0">
      <div class="flex items-center justify-between gap-2 flex-wrap">
        <p class="font-semibold text-sm text-emerald-900 dark:text-emerald-100">
          Active Session: <span class="font-mono bg-emerald-500/20 px-1.5 py-0.5 rounded text-xs">{session.currentModel}</span>
          {#if expiresInMin !== null}
            <span class="font-normal text-xs text-emerald-700 dark:text-emerald-300 ml-1">
              ({expiresInMin > 0 ? `Expires in ${expiresInMin} min` : 'Expires soon'})
            </span>
          {/if}
        </p>
        <button
          type="button"
          title="Refresh session status"
          disabled={isLoading}
          onclick={onRefresh}
          class="p-1 rounded-md text-emerald-700 dark:text-emerald-300 hover:bg-emerald-500/20 transition-colors cursor-pointer"
        >
          <RotateCcw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
        </button>
      </div>
      <p class="mt-1 text-emerald-700/90 dark:text-emerald-300/90">
        Freebuff limits each account to 1 model per hour. Other models are locked until this session expires.
      </p>
    </div>
  </div>
{/if}
