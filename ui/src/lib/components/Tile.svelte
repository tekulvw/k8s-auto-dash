<script lang="ts">
  import Icon from './Icon.svelte';
  import StatusDot from './StatusDot.svelte';
  import type { Tile } from '$lib/api/types';

  export let tile: Tile;
  export let editing: boolean = false;
  export let onedit: ((detail: { id: string }) => void) | undefined = undefined;
  export let onhide: ((detail: { id: string }) => void) | undefined = undefined;

  $: hostname = (() => {
    try { return new URL(tile.url).hostname; } catch { return tile.url; }
  })();

  function handleClick() {
    if (!editing && tile.url) window.open(tile.url, '_blank');
  }
</script>

<div
  class="tile"
  class:editing
  class:hidden={tile.hidden}
  role="button"
  tabindex="0"
  on:click={handleClick}
  on:keydown={(e) => e.key === 'Enter' && handleClick()}
>
  <Icon slug={tile.icon} alt={tile.name} />
  <div class="body">
    <div class="name">{tile.name}</div>
    <div class="host">{hostname}</div>
    {#if tile.description}<div class="desc">{tile.description}</div>{/if}
  </div>
  <StatusDot state={tile.status.state} title={`${tile.status.state}${tile.status.statusCode ? ' · ' + tile.status.statusCode : ''}`} />

  {#if editing}
    <div class="actions">
      <button aria-label="Edit tile" on:click|stopPropagation={() => onedit?.({ id: tile.id })}>✎</button>
      <button aria-label="Hide tile" on:click|stopPropagation={() => onhide?.({ id: tile.id })}>×</button>
    </div>
  {/if}
</div>

<style>
  .tile {
    display: flex; gap: 12px; align-items: center;
    padding: 12px;
    background: var(--bg-card);
    border-radius: 10px;
    cursor: pointer;
    transition: background 0.15s;
  }
  .tile:hover { background: var(--bg-card-hover); }
  .tile.hidden { opacity: 0.4; }
  .body { flex: 1; min-width: 0; }
  .name { font-weight: 600; color: var(--fg); }
  .host { font-size: 12px; color: var(--fg-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .desc { font-size: 12px; color: var(--fg-muted); margin-top: 2px; }
  .actions { display: flex; gap: 4px; }
  .actions button {
    background: none; border: 1px solid var(--border);
    color: var(--fg); cursor: pointer; padding: 2px 6px; border-radius: 4px;
  }
</style>
