<script lang="ts">
  import {
    Bot,
    Check,
    CheckCircle2,
    Copy,
    Eye,
    FlaskConical,
    Loader2,
    Sparkles,
    X,
    XCircle
  } from 'lucide-svelte'
  import type { ModelItem } from './types'

  interface Props {
    model: ModelItem
    fullModel: string
    displayModelText: string
    isTesting: boolean
    testResult?: 'ok' | 'error'
    isCopied: boolean
    isActiveSession?: boolean
    isLockedBySession?: boolean
    onTest: () => void
    onCopy: () => void
    onDisable: () => void
  }

  let {
    model,
    displayModelText,
    isTesting,
    testResult,
    isCopied,
    isActiveSession = false,
    isLockedBySession = false,
    onTest,
    onCopy,
    onDisable,
  }: Props = $props()
</script>

<div class="p-3 rounded-xl border bg-surface transition-colors flex items-center justify-between gap-2.5 {isActiveSession ? 'border-emerald-500/50 bg-emerald-500/5 shadow-xs' : isLockedBySession ? 'border-border/60 opacity-80 hover:opacity-100 hover:border-border' : 'border-border hover:border-brand-500/40'}">
  <div class="flex items-start gap-2.5 min-w-0 flex-1">
    <div class="mt-0.5 shrink-0">
      {#if isTesting}
        <Loader2 class="w-4 h-4 animate-spin text-brand-500" />
      {:else if testResult === 'ok'}
        <CheckCircle2 class="w-4 h-4 text-green-500" />
      {:else if testResult === 'error'}
        <XCircle class="w-4 h-4 text-red-500" />
      {:else}
        <Bot class="w-4 h-4 text-text-muted opacity-60" />
      {/if}
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-1.5 flex-wrap">
        <code class="text-xs font-mono font-medium text-text-main break-all">
          {displayModelText}
        </code>
        {#if isActiveSession}
          <span class="inline-flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            Active Session
          </span>
        {:else if isLockedBySession}
          <span class="inline-flex items-center gap-1 text-[10px] font-medium px-1.5 py-0.5 rounded bg-surface-2 text-text-muted/80 border border-border/50" title="Locked by active session">
            🔒 Locked
          </span>
        {/if}
      </div>
      <div class="flex items-center gap-2 flex-wrap mt-0.5">
        {#if model.name}
          <span class="text-[11px] italic text-text-muted truncate max-w-[130px]" title={model.name}>
            {model.name}
          </span>
        {/if}
        {#if model.caps.vision}
          <span class="inline-flex items-center gap-1 text-[10px] font-medium text-blue-500">
            <Eye class="w-3 h-3" />
            Vision
          </span>
        {/if}
        {#if model.caps.reasoning}
          <span class="inline-flex items-center gap-1 text-[10px] font-medium text-amber-500">
            <Sparkles class="w-3 h-3" />
            Reasoning
          </span>
        {/if}
      </div>
    </div>
  </div>

  <div class="flex items-center gap-1 shrink-0">
    <button
      type="button"
      title="Test Model"
      disabled={isTesting}
      onclick={onTest}
      class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-text-main hover:border-brand-500/40 transition-colors cursor-pointer disabled:opacity-50"
    >
      <FlaskConical class="w-3.5 h-3.5 {isTesting ? 'animate-pulse text-brand-500' : ''}" />
    </button>

    <button
      type="button"
      title="Copy Model Name"
      onclick={onCopy}
      class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-text-main hover:border-brand-500/40 transition-colors cursor-pointer"
    >
      {#if isCopied}
        <Check class="w-3.5 h-3.5 text-green-500" />
      {:else}
        <Copy class="w-3.5 h-3.5" />
      {/if}
    </button>

    <button
      type="button"
      title="Disable Model"
      onclick={onDisable}
      class="p-1.5 rounded-lg border border-border bg-surface text-text-muted hover:text-red-500 hover:border-red-500/40 transition-colors cursor-pointer"
    >
      <X class="w-3.5 h-3.5" />
    </button>
  </div>
</div>
