<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Tile from './Tile.svelte';
  import type { Group, Tile as TTile } from '$lib/api/types';

  export let group: Group;
  export let tiles: TTile[];
  export let editing: boolean = false;

  const dispatch = createEventDispatcher<{
    editTile: { id: string };
    hideTile: { id: string };
    addBookmark: { group: string };
  }>();

  $: visible = editing ? tiles : tiles.filter((t) => !t.hidden);
  $: sorted = [...visible].sort((a, b) => a.order - b.order || a.name.localeCompare(b.name));
</script>

<section class="group">
  <header>
    <h2>{group.name}</h2>
    {#if editing}
      <button class="add" on:click={() => dispatch('addBookmark', { group: group.id })}>
        + bookmark
      </button>
    {/if}
  </header>

  <div class="grid">
    {#each sorted as t (t.id)}
      <Tile
        tile={t}
        {editing}
        onedit={(e) => dispatch('editTile', e)}
        onhide={(e) => dispatch('hideTile', e)}
      />
    {/each}
  </div>
</section>

<style>
  .group { margin-bottom: 32px; }
  header { display: flex; align-items: baseline; gap: 12px; margin-bottom: 12px; }
  h2 { margin: 0; font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; color: var(--fg-muted); }
  .add { background: none; border: 1px dashed var(--border); color: var(--fg-muted); padding: 2px 8px; border-radius: 4px; font-size: 12px; cursor: pointer; }
  .grid {
    display: grid; gap: 12px;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  }
</style>
