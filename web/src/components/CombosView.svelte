<script lang="ts">
  import {
    Activity,
    ArrowDown,
    ArrowRight,
    ArrowUp,
    Check,
    Copy,
    Cpu,
    ExternalLink,
    Eye,
    FileText,
    GitBranch,
    GripVertical,
    Headphones,
    Layers,
    Loader2,
    Mic,
    Play,
    Plus,
    RefreshCw,
    Save,
    Trash2,
    Zap
  } from 'lucide-svelte'
  import { api, getAuthHeaders, type Combo } from '../api/client'

  let {
    combos = [],
    onRefresh,
    isCreatingOpen = $bindable(false)
  }: {
    combos: Combo[]
    onRefresh: () => void
    isCreatingOpen?: boolean
  } = $props()

  let selectedComboId = $state<string | null>(null)
  let newComboName = $state('')
  let newComboStrategy = $state('fallback')

  let editingModels = $state<string[]>([])
  let editingStrategy = $state<string>('fallback')
  let modelInput = $state('')
  let isSaving = $state(false)
  let copiedName = $state<string | null>(null)

  // Live Test Playground
  let testPrompt = $state('Say hello in 3 words')
  let testOutput = $state('')
  let isTesting = $state(false)
  let testLatency = $state<number | null>(null)

  // Filters
  let searchFilter = $state('')
  let strategyFilter = $state('ALL')

  let selectedCombo = $derived(combos.find((c) => c.id === selectedComboId))

  $effect(() => {
    if (selectedComboId === null && combos.length > 0) {
      selectedComboId = combos[0].id
    }
    if (selectedCombo) {
      try {
        const parsed = JSON.parse(selectedCombo.models)
        editingModels = Array.isArray(parsed) ? parsed : [selectedCombo.models]
      } catch {
        editingModels = selectedCombo.models ? [selectedCombo.models] : []
      }
      editingStrategy = selectedCombo.strategy || 'fallback'
      testOutput = ''
      testLatency = null
    }
  })

  function copyAlias(name: string) {
    navigator.clipboard.writeText(name)
    copiedName = name
    setTimeout(() => (copiedName = null), 2000)
  }

  function handleMoveModel(index: number, delta: number) {
    const newIdx = index + delta
    if (newIdx < 0 || newIdx >= editingModels.length) return
    const updated = [...editingModels]
    const temp = updated[index]
    updated[index] = updated[newIdx]
    updated[newIdx] = temp
    editingModels = updated
  }

  function handleRemoveModel(index: number) {
    editingModels = editingModels.filter((_, i) => i !== index)
  }

  function handleAddModel() {
    if (!modelInput.trim()) return
    editingModels = [...editingModels, modelInput.trim()]
    modelInput = ''
  }

  async function handleSaveCombo() {
    if (!selectedCombo) return
    try {
      isSaving = true
      await api.updateCombo(selectedCombo.id, {
        models: JSON.stringify(editingModels),
        strategy: editingStrategy,
      })
      onRefresh()
      alert('Combo pipeline saved successfully!')
    } catch (err) {
      alert(`Failed to save combo: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSaving = false
    }
  }

  async function handleCreateCombo(e: SubmitEvent) {
    e.preventDefault()
    if (!newComboName.trim()) return
    try {
      isSaving = true
      await api.createCombo({
        name: newComboName.trim(),
        models: JSON.stringify([]),
        strategy: newComboStrategy,
      })
      isCreatingOpen = false
      newComboName = ''
      onRefresh()
    } catch (err) {
      alert(`Failed to create combo: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      isSaving = false
    }
  }

  async function handleDeleteCombo(id: string) {
    if (!confirm('Are you sure you want to delete this combo?')) return
    try {
      await api.deleteCombo(id)
      onRefresh()
      selectedComboId = combos.find((c) => c.id !== id)?.id || null
    } catch (err) {
      alert(`Failed to delete combo: ${err instanceof Error ? err.message : String(err)}`)
    }
  }

  async function handleRunLiveTest() {
    if (!selectedCombo) return
    isTesting = true
    testOutput = ''
    testLatency = null
    const startTime = performance.now()

    try {
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          model: selectedCombo.name,
          messages: [{ role: 'user', content: testPrompt }],
          stream: true,
          max_tokens: 150,
        }),
      })

      if (!res.ok) {
        const errText = await res.text()
        throw new Error(errText || `HTTP ${res.status}`)
      }

      const reader = res.body?.getReader()
      const decoder = new TextDecoder()
      if (!reader) return

      let buffer = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed || trimmed.startsWith(':')) continue
          if (trimmed === 'data: [DONE]') continue
          if (trimmed.startsWith('data: ')) {
            try {
              const chunk = JSON.parse(trimmed.slice(6))
              const delta = chunk.choices?.[0]?.delta?.content || ''
              const reasoning = chunk.choices?.[0]?.delta?.reasoning_content || ''
              if (reasoning) {
                testOutput += `[thinking: ${reasoning}]`
              }
              if (delta) {
                testOutput += delta
              }
            } catch {
              // ignore
            }
          }
        }
      }
      testLatency = Math.round(performance.now() - startTime)
    } catch (err) {
      testOutput = `Error: ${err instanceof Error ? err.message : String(err)}`
    } finally {
      isTesting = false
    }
  }

  let filteredCombos = $derived(
    combos.filter((c) => {
      if (searchFilter.trim() && !c.name.toLowerCase().includes(searchFilter.toLowerCase())) {
        return false
      }
      if (strategyFilter !== 'ALL' && c.strategy !== strategyFilter) {
        return false
      }
      return true
    })
  )
</script>

<div class="space-y-6">
  <!-- Page header -->
  <div class="flex flex-col lg:flex-row lg:items-end justify-between gap-4">
    <div class="space-y-1.5 max-w-2xl">
      <div class="flex items-center gap-2">
        <span class="font-code text-[10px] uppercase tracking-wider text-brand-500 px-2 py-0.5 rounded bg-brand-500/10 border border-brand-500/25 font-bold">
          Virtualization Layer
        </span>
        <span class="text-text-subtle">•</span>
        <span class="font-code text-[11px] text-text-subtle">Cluster 9Router-East</span>
      </div>
      <h1 class="font-headline text-2xl sm:text-3xl font-bold text-text-main tracking-tight">
        Model Combos & Intelligent Routing
      </h1>
      <p class="font-body text-xs sm:text-sm text-text-muted leading-relaxed">
        Group multiple LLMs under unified virtual endpoints with automated failover, load balancing, or consensus fusion across multi-cloud credentials.
      </p>
    </div>

    <!-- Quick Metrics Summary Pills -->
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface border border-border font-code text-xs">
        <span class="text-text-muted">Active Endpoints:</span>
        <span class="text-success font-bold">{combos.length} Online</span>
      </div>

      <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-surface border border-border font-code text-xs">
        <span class="text-text-muted">Failover Speed:</span>
        <span class="text-info font-bold">&lt;48ms</span>
      </div>

      <button
        type="button"
        onclick={() => (isCreatingOpen = true)}
        class="flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-body text-xs font-bold shadow-md shadow-brand-500/25 transition cursor-pointer"
      >
        <Plus class="w-4 h-4" />
        <span>Create New Combo</span>
      </button>
    </div>
  </div>

  <!-- Strategy Blueprint Cards (Stitch Design) -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
    <!-- Fallback Chain -->
    <div class="p-4 rounded-xl bg-surface border border-border space-y-2">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="p-1.5 rounded-lg bg-brand-500/15 text-brand-500">
            <GitBranch class="w-4 h-4" />
          </div>
          <span class="font-headline text-xs font-bold text-text-main">Fallback Chain</span>
        </div>
        <span class="font-code text-[10px] text-brand-500 bg-brand-500/10 px-2 py-0.5 rounded-full border border-brand-500/20 font-semibold">
          Default
        </span>
      </div>
      <p class="font-body text-[11px] text-text-muted leading-relaxed">
        Queries models sequentially. If primary model returns 429, 5xx, or timeouts, seamlessly switches downstream without socket drops.
      </p>
      <div class="font-code text-[10px] text-text-subtle flex items-center gap-1.5 pt-1">
        <span class="w-1.5 h-1.5 rounded-full bg-brand-500"></span>
        <span>Policy: Next-on-failure</span>
      </div>
    </div>

    <!-- Round Robin -->
    <div class="p-4 rounded-xl bg-surface border border-border space-y-2">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="p-1.5 rounded-lg bg-info/15 text-info">
            <RefreshCw class="w-4 h-4" />
          </div>
          <span class="font-headline text-xs font-bold text-text-main">Round Robin</span>
        </div>
        <span class="font-code text-[10px] text-info bg-info/10 px-2 py-0.5 rounded-full border border-info/20 font-semibold">
          Load Spread
        </span>
      </div>
      <p class="font-body text-[11px] text-text-muted leading-relaxed">
        Rotates requests across candidate keys and regional instances to maximize TPM quotas and minimize rate-limit throttling.
      </p>
      <div class="font-code text-[10px] text-text-subtle flex items-center gap-1.5 pt-1">
        <span class="w-1.5 h-1.5 rounded-full bg-info"></span>
        <span>Policy: Weighted distribution</span>
      </div>
    </div>

    <!-- Consensus Fusion -->
    <div class="p-4 rounded-xl bg-surface border border-border space-y-2">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="p-1.5 rounded-lg bg-success/15 text-success">
            <Zap class="w-4 h-4" />
          </div>
          <span class="font-headline text-xs font-bold text-text-main">Consensus Fusion</span>
        </div>
        <span class="font-code text-[10px] text-success bg-success/10 px-2 py-0.5 rounded-full border border-success/20 font-semibold">
          Max Quality
        </span>
      </div>
      <p class="font-body text-[11px] text-text-muted leading-relaxed">
        Queries parallel LLM nodes simultaneously, using fast-evaluator judge to select or synthesize the most coherent response.
      </p>
      <div class="font-code text-[10px] text-text-subtle flex items-center gap-1.5 pt-1">
        <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
        <span>Policy: Parallel Judge (N+1)</span>
      </div>
    </div>
  </div>

  <!-- Filter & Realtime Query Toolbar (Stitch Screenshot) -->
  <div class="flex flex-col sm:flex-row items-center justify-between gap-3 p-2 rounded-xl bg-surface border border-border">
    <div class="relative w-full sm:w-80 flex items-center">
      <Search class="absolute left-3 w-4 h-4 text-text-subtle pointer-events-none" />
      <input
        type="text"
        bind:value={searchFilter}
        placeholder="Filter combos by alias, tag, or backing provider..."
        class="w-full bg-surface-2 border border-border rounded-lg pl-9 pr-3 py-1.5 font-body text-xs text-text-main placeholder:text-text-subtle focus:outline-none focus:border-brand-500 transition"
      />
    </div>

    <div class="flex items-center gap-2">
      <div class="flex items-center gap-1.5 text-xs font-code text-text-muted">
        <span>Strategy:</span>
        <select
          bind:value={strategyFilter}
          class="bg-surface-2 border border-border rounded px-2.5 py-1 text-xs text-text-main focus:outline-none"
        >
          <option value="ALL">All Strategies ({combos.length})</option>
          <option value="fallback">Fallback Only</option>
          <option value="round-robin">Round Robin Only</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Combo Rows List & Pipeline Visualization (Stitch Design) -->
  <div class="space-y-3">
    {#each filteredCombos as c (c.id)}
      {@const isSelected = c.id === selectedComboId}
      {@const parsedModels = (() => {
        try {
          const parsed = JSON.parse(c.models)
          return Array.isArray(parsed) ? parsed : [c.models]
        } catch {
          return c.models ? [c.models] : []
        }
      })()}

      <div
        class="rounded-xl border transition-all overflow-hidden {isSelected
          ? 'bg-surface-2 border-brand-500/50 shadow-lg shadow-brand-500/10'
          : 'bg-surface border-border hover:border-border'}"
      >
        <!-- Combo Row Header -->
        <div class="p-4 flex flex-col md:flex-row md:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-lg bg-bg border border-border flex items-center justify-center text-brand-500">
              <Layers class="w-4 h-4" />
            </div>

            <div>
              <div class="flex items-center gap-2">
                <span class="font-headline text-sm font-bold text-text-main">{c.name}</span>
                <span class="font-code text-[10px] px-2 py-0.5 rounded bg-brand-500/15 text-brand-400 font-bold border border-brand-500/25">
                  model: "{c.name}"
                </span>
                <span class="font-code text-[10px] text-success bg-success/10 px-2 py-0.5 rounded border border-success/20">
                  99.8% uptime
                </span>
              </div>
              <div class="text-[11px] text-text-muted font-code pt-0.5">
                0 failover drops • Virtual Gateway Endpoint
              </div>
            </div>
          </div>

          <!-- Pipeline Flow Preview -->
          <div class="flex items-center gap-2 overflow-x-auto py-1">
            <span class="font-code text-[10px] text-text-subtle uppercase font-bold">Pipeline:</span>
            {#if parsedModels.length > 0}
              {#each parsedModels as m, idx}
                <div class="flex items-center gap-1.5 font-code text-xs">
                  <span class="px-2 py-0.5 rounded bg-bg border border-border text-text-main">
                    <strong class="text-brand-500">{idx + 1}</strong> {m}
                  </span>
                  {#if idx < parsedModels.length - 1}
                    <ArrowRight class="w-3 h-3 text-text-subtle" />
                  {/if}
                </div>
              {/each}
            {:else}
              <span class="text-xs text-text-subtle italic">No upstream models assigned yet</span>
            {/if}
          </div>

          <!-- Right Actions -->
          <div class="flex items-center gap-2 self-end md:self-auto">
            <span class="font-code text-[10px] px-2 py-1 rounded bg-bg text-info border border-border capitalize">
              {c.strategy === 'round-robin' ? 'Round Robin - spread load' : 'Fallback - try in order'}
            </span>

            <button
              type="button"
              onclick={() => copyAlias(c.name)}
              class="p-1.5 rounded-lg text-text-muted hover:text-text-main bg-bg border border-border cursor-pointer"
              title="Copy Model Name"
            >
              {#if copiedName === c.name}
                <Check class="w-3.5 h-3.5 text-success" />
              {:else}
                <Copy class="w-3.5 h-3.5" />
              {/if}
            </button>

            <button
              type="button"
              onclick={() => (selectedComboId = isSelected ? null : c.id)}
              class="px-3 py-1 rounded-lg text-xs font-semibold cursor-pointer transition {isSelected
                ? 'text-text-main'
                : 'bg-surface-3 hover:bg-surface-3 text-text-main'}"
            >
              {isSelected ? 'Close Editor' : 'Edit Pipeline'}
            </button>
          </div>
        </div>

        <!-- Expanded Pipeline Editor & Live Test (If Selected) -->
        {#if isSelected}
          <div class="p-5 border-t border-border bg-surface-2 space-y-5">
            <!-- Pipeline Reorder Sequence -->
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <div class="font-body text-xs font-semibold text-text-main">
                  Model Priority Pipeline (Top-to-Bottom Execution)
                </div>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    onclick={() => handleDeleteCombo(c.id)}
                    class="text-xs text-hover:text-danger hover:underline cursor-pointer"
                  >
                    Delete Combo
                  </button>
                  <button
                    type="button"
                    onclick={handleSaveCombo}
                    disabled={isSaving}
                    class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-success hover:brightness-110 text-black font-bold text-xs shadow-md transition cursor-pointer"
                  >
                    {#if isSaving}
                      <Loader2 class="w-3.5 h-3.5 animate-spin" />
                    {:else}
                      <Save class="w-3.5 h-3.5" />
                    {/if}
                    <span>Save Pipeline Changes</span>
                  </button>
                </div>
              </div>

              <div class="space-y-2">
                {#each editingModels as model, idx}
                  <div class="flex items-center justify-between p-3 rounded-lg bg-surface border border-border">
                    <div class="flex items-center gap-3">
                      <span class="w-6 h-6 rounded bg-surface-2 text-text-main flex items-center justify-center font-bold text-xs font-code">
                        {idx + 1}
                      </span>
                      <span class="font-code text-xs text-text-main">{model}</span>
                    </div>

                    <div class="flex items-center gap-1">
                      <button
                        type="button"
                        onclick={() => handleMoveModel(idx, -1)}
                        disabled={idx === 0}
                        class="p-1 rounded text-text-muted hover:text-text-main disabled:opacity-30 cursor-pointer"
                        title="Move Up"
                      >
                        <ArrowUp class="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        onclick={() => handleMoveModel(idx, 1)}
                        disabled={idx === editingModels.length - 1}
                        class="p-1 rounded text-text-muted hover:text-text-main disabled:opacity-30 cursor-pointer"
                        title="Move Down"
                      >
                        <ArrowDown class="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        onclick={() => handleRemoveModel(idx)}
                        class="p-1 rounded text-text-muted hover:text-hover:text-danger ml-2 cursor-pointer"
                        title="Remove"
                      >
                        <Trash2 class="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                {/each}
              </div>

              <!-- Add model input -->
              <div class="flex gap-2 pt-1">
                <input
                  type="text"
                  placeholder="Enter model string (e.g. fb/z-ai/glm-5.3-flash, ag/gemini-2.5-flash-high, deepseek-chat)"
                  bind:value={modelInput}
                  onkeydown={(e) => e.key === 'Enter' && handleAddModel()}
                  class="flex-1 px-3 py-2 rounded-lg bg-surface border border-border font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
                />
                <button
                  type="button"
                  onclick={handleAddModel}
                  class="px-4 py-2 rounded-lg bg-surface-3 hover:bg-surface-3 text-text-main font-body text-xs font-bold flex items-center gap-1.5 transition cursor-pointer"
                >
                  <Plus class="w-3.5 h-3.5" />
                  <span>Add Upstream Model</span>
                </button>
              </div>
            </div>

            <!-- Live Test Playground (Stitch Style) -->
            <div class="pt-4 border-t border-border space-y-3">
              <div class="flex items-center justify-between">
                <div class="font-headline text-xs font-bold text-text-main flex items-center gap-2">
                  <Play class="w-3.5 h-3.5 text-success" />
                  <span>Live Test Playground: Testing "{c.name}"</span>
                </div>
                {#if testLatency !== null}
                  <span class="font-code text-[11px] text-success">RTT Latency: {testLatency}ms</span>
                {/if}
              </div>

              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={testPrompt}
                  placeholder="Test prompt..."
                  class="flex-1 px-3 py-2 rounded-lg bg-surface border border-border font-body text-xs text-text-main focus:outline-none focus:border-brand-500"
                />
                <button
                  type="button"
                  onclick={handleRunLiveTest}
                  disabled={isTesting}
                  class="px-4 py-2 rounded-lg text-text-main font-body text-xs font-bold flex items-center gap-1.5 transition cursor-pointer"
                >
                  {#if isTesting}
                    <Loader2 class="w-3.5 h-3.5 animate-spin" />
                  {:else}
                    <Play class="w-3.5 h-3.5" />
                  {/if}
                  <span>Run Live Test</span>
                </button>
              </div>

              {#if testOutput}
                <div class="p-3.5 rounded-lg bg-bg border border-border font-code text-xs text-success whitespace-pre-wrap max-h-48 overflow-y-auto">
                  {testOutput}
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Modality Switch Adapters (From Stitch Screenshot) -->
  <div class="p-5 rounded-xl bg-surface border border-border space-y-3">
    <div class="space-y-0.5">
      <h3 class="font-headline text-sm font-bold text-text-main flex items-center gap-2">
        <Cpu class="w-4 h-4 text-info" />
        <span>Modality Switch Adapters</span>
      </h3>
      <p class="font-body text-xs text-text-muted">
        If your request carries image, audio, or binary attachments and the designated model cannot parse them, 9Router will intelligently intercept and swap downstream models on the fly.
      </p>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1">
      <!-- Vision Adapter -->
      <div class="p-3.5 rounded-lg bg-surface-2 border border-border flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-lg bg-info/15 text-info">
            <Eye class="w-4 h-4" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <span class="font-headline text-xs font-bold text-text-main">VisionAdapter</span>
              <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-success/15 text-success font-semibold">Active</span>
            </div>
            <div class="font-body text-[11px] text-text-muted">Intercepts PNG, JPG, WEBP, and PDF frames</div>
            <div class="font-code text-[10px] text-info pt-0.5">Swaps to: ag/gemini-2.5-flash-high</div>
          </div>
        </div>

        <div class="w-2 h-2 rounded-full bg-success animate-pulse"></div>
      </div>

      <!-- Audio Adapter -->
      <div class="p-3.5 rounded-lg bg-surface-2 border border-border flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-lg bg-brand-500/15 text-brand-500">
            <Mic class="w-4 h-4" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <span class="font-headline text-xs font-bold text-text-main">AudioInputAdapter</span>
              <span class="font-code text-[9px] px-1.5 py-0.2 rounded bg-brand-400/15 text-brand-400 font-semibold">Standby</span>
            </div>
            <div class="font-body text-[11px] text-text-muted">Intercepts MP3, WAV, FLAC streams</div>
            <div class="font-code text-[10px] text-brand-400 pt-0.5">Swaps to: openai/whisper-large-v3-turbo</div>
          </div>
        </div>

        <div class="w-2 h-2 rounded-full bg-[#ff8469]"></div>
      </div>
    </div>
  </div>

  <!-- Create Modal (Stitch Mac-Style) -->
  {#if isCreatingOpen}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4">
      <div class="w-full max-w-md p-6 rounded-2xl bg-surface-2 border border-border shadow-2xl space-y-4">
        <div class="flex items-center justify-between pb-2 border-b border-border">
          <div class="flex items-center gap-2">
            <button
              type="button"
              aria-label="Close dialog"
              onclick={() => (isCreatingOpen = false)}
              class="w-3 h-3 rounded-full bg-[#ff5f56] cursor-pointer"
            ></button>
            <div class="w-3 h-3 rounded-full bg-[#ffbd2e]"></div>
            <div class="w-3 h-3 rounded-full bg-[#27c93f]"></div>
            <span class="ml-2 font-headline text-sm font-bold text-text-main">
              Create Virtual Model Combo
            </span>
          </div>
        </div>

        <form onsubmit={handleCreateCombo} class="space-y-3 font-body text-xs">
          <div>
            <label for="create-combo-name" class="block font-semibold text-text-muted mb-1">Combo Name *</label>
            <input
              id="create-combo-name"
              type="text"
              placeholder="e.g. smart-combo, code-fast"
              bind:value={newComboName}
              required
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
            />
          </div>

          <div>
            <label for="create-strategy-select" class="block font-semibold text-text-muted mb-1">Routing Strategy</label>
            <select
              id="create-strategy-select"
              bind:value={newComboStrategy}
              class="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
            >
              <option value="fallback">Sequential Fallback (Recommended)</option>
              <option value="round-robin">Round Robin - Load Spread</option>
            </select>
          </div>

          <div class="flex justify-end gap-2 pt-3 border-t border-border">
            <button
              type="button"
              onclick={() => (isCreatingOpen = false)}
              class="px-4 py-2 rounded-lg text-text-muted hover:text-text-main cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSaving}
              class="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-brand-500 hover:bg-brand-600 text-white font-bold shadow-md shadow-brand-500/25 cursor-pointer"
            >
              <Check class="w-3.5 h-3.5" />
              <span>Create Combo</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
