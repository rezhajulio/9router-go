<script lang="ts">
  import { Maximize2, Minus, Plus } from 'lucide-svelte'
  import Card from '../../lib/ui/Card.svelte'

  interface Props {
    activeNodeCount?: number
  }

  let { activeNodeCount = 0 }: Props = $props()
</script>

<!-- Interactive Provider Topology Card -->
<Card padding="md" class="flex min-w-0 flex-col justify-between overflow-hidden relative" style="min-height: 480px">
  <div class="flex items-center justify-between pb-3 border-b border-border/80 z-10">
    <div class="flex items-center gap-2">
      <span class="w-2 h-2 rounded-full bg-success"></span>
      <span class="font-headline text-xs font-bold text-text-main uppercase tracking-wider">Provider Topology</span>
      <span class="font-code text-[11px] text-text-muted">• {activeNodeCount} Nodes Active</span>
    </div>
    <span class="font-code text-[10px] text-info bg-surface-2 px-2 py-0.5 rounded border border-border">
      Interactive Mesh
    </span>
  </div>

  <!-- Node Graph Canvas -->
  <div class="relative w-full flex-1 flex items-center justify-center my-4 overflow-hidden select-none">
    <div class="z-20 p-3 sm:p-4 rounded-xl bg-surface border-2 border-brand-500 shadow-xl shadow-brand-500/20 text-center flex flex-col items-center">
      <div class="flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-brand-500 animate-ping"></span>
        <span class="font-headline text-xs sm:text-sm font-bold text-text-main">9Router</span>
      </div>
      <span class="font-code text-[10px] text-text-muted pt-0.5">:20130 Gateway</span>
    </div>

    <svg class="absolute inset-0 w-full h-full pointer-events-none stroke-border/70 dark:stroke-white/10" viewBox="0 0 600 360">
      <line x1="300" y1="180" x2="160" y2="70" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="440" y2="70" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="110" y2="180" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="490" y2="180" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="160" y2="290" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="440" y2="290" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="300" y2="50" stroke-width="1.5" stroke-dasharray="4 2" />
      <line x1="300" y1="180" x2="300" y2="310" stroke-width="1.5" stroke-dasharray="4 2" />
    </svg>

    <div class="absolute top-4 left-1/2 -translate-x-1/2 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
      <span>NVIDIA NIM</span>
    </div>
    <div class="absolute top-10 left-8 sm:left-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
      <span>OpenCode Free</span>
    </div>
    <div class="absolute top-10 right-8 sm:right-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
      <span>Antigravity</span>
    </div>
    <div class="absolute top-1/2 -translate-y-1/2 left-2 sm:left-6 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-warning"></span>
      <span>MiMo Code</span>
    </div>
    <div class="absolute top-1/2 -translate-y-1/2 right-2 sm:right-6 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
      <span>OpenRouter</span>
    </div>
    <div class="absolute bottom-10 left-8 sm:left-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-brand-500"></span>
      <span>Grok CLI</span>
    </div>
    <div class="absolute bottom-10 right-8 sm:right-16 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
      <span>ClinePass</span>
    </div>
    <div class="absolute bottom-4 left-1/2 -translate-x-1/2 z-10 px-2.5 py-1 rounded-lg bg-surface border border-border font-code text-[11px] text-text-main shadow-sm flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
      <span>inference.dahl.global</span>
    </div>
  </div>

  <div class="flex items-center justify-between pt-2 border-t border-border/80 text-[10px] font-code text-text-muted z-10">
    <div class="flex items-center gap-3">
      <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-success"></span> Active</span>
      <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-info"></span> Standby</span>
      <span class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-brand-500"></span> Fallback</span>
    </div>
    <div class="flex items-center gap-1">
      <button type="button" class="p-1 rounded hover:bg-surface-2 border border-border cursor-pointer"><Plus class="w-3 h-3" /></button>
      <button type="button" class="p-1 rounded hover:bg-surface-2 border border-border cursor-pointer"><Minus class="w-3 h-3" /></button>
      <button type="button" class="p-1 rounded hover:bg-surface-2 border border-border cursor-pointer"><Maximize2 class="w-3 h-3" /></button>
    </div>
  </div>
</Card>
