<script lang="ts">
  import Icon from './Icon.svelte';
  import type { Tile } from '$lib/api/types';

  export let tiles: Tile[];
  export let onrestore: ((detail: { id: string }) => void) | undefined = undefined;
</script>

<aside class="drawer">
  <h3>Hidden tiles</h3>
  {#if tiles.length === 0}
    <p class="muted">No hidden tiles.</p>
  {:else}
    <ul>
      {#each tiles as t (t.id)}
        <li on:click={() => onrestore?.({ id: t.id })}>
          <Icon slug={t.icon} alt={t.name} />
          <span>{t.name}</span>
        </li>
      {/each}
    </ul>
  {/if}
</aside>

<style>
  .drawer { padding: 16px; }
  h3 { margin-top: 0; }
  ul { list-style: none; padding: 0; margin: 0; }
  li { display: flex; gap: 8px; align-items: center; padding: 6px; border-radius: 4px; cursor: pointer; }
  li:hover { background: var(--bg-card-hover); }
  .muted { color: var(--fg-muted); }
</style>
