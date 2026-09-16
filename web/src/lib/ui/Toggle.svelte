<script lang="ts">
  // Port of decolua/9router src/shared/components/Toggle.js (switch)
  let {
    checked = false,
    disabled = false,
    onChange
  }: {
    checked?: boolean
    disabled?: boolean
    onChange?: (value: boolean) => void
  } = $props()
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked}
  disabled={disabled}
  onclick={() => onChange?.(!checked)}
  class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed {checked
    ? 'bg-brand-500'
    : 'bg-surface-3'}"
>
  <span
    class="inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow transition-transform {checked
      ? 'translate-x-[18px]'
      : 'translate-x-[3px]'}"
  />
</button>
