<script lang="ts">
  import { editMode } from '$lib/stores/edit';
  import { searchQuery } from '$lib/stores/search';
  import { theme, type Theme } from '$lib/stores/theme';

  export let title: string = 'Dashboard';

  function cycleTheme() {
    const order: Theme[] = ['dark', 'light', 'auto'];
    const next = order[(order.indexOf($theme) + 1) % order.length];
    theme.set(next);
  }
</script>

<header class="header">
  <h1>{title}</h1>
  <input
    class="search"
    type="search"
    placeholder="Search"
    bind:value={$searchQuery}
    aria-label="Search tiles"
  />
  <button class="icon-btn" aria-label="Cycle theme" on:click={cycleTheme}>
    {#if $theme === 'dark'}🌙{:else if $theme === 'light'}☀{:else}◐{/if}
  </button>
  <button
    class="icon-btn"
    aria-label="Toggle edit mode"
    class:active={$editMode}
    on:click={() => editMode.update((v) => !v)}
  >✎</button>
</header>

<style>
  .header {
    display: flex; align-items: center; gap: 12px;
    padding: 16px 24px;
    border-bottom: 1px solid var(--border);
  }
  h1 { margin: 0; font-size: 18px; flex: 0; }
  .search {
    flex: 1; max-width: 400px;
    padding: 6px 10px; border-radius: 6px;
    background: var(--bg-card); border: 1px solid var(--border);
    color: var(--fg);
  }
  .icon-btn {
    background: none; border: 1px solid var(--border);
    color: var(--fg); cursor: pointer;
    padding: 4px 10px; border-radius: 6px;
  }
  .icon-btn.active { background: var(--accent); color: var(--accent-fg); }
</style>
