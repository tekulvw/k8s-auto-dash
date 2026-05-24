<script lang="ts">
  import type { StatusState } from '$lib/api/types';
  export let state: StatusState = 'unknown';
  export let title: string = '';
</script>

<span class="dot dot-{state}" title={title} aria-label={state}></span>

<style>
  .dot {
    display: inline-block;
    width: 10px; height: 10px;
    border-radius: 50%;
    background: var(--fg-muted);
  }
  .dot-up       { background: #3fb950; }
  .dot-degraded { background: #d29922; }
  .dot-down     { background: #f85149; }
  .dot-unknown  { background: #6e7681; }
</style>
