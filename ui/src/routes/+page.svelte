<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import { subscribeEvents } from '$lib/api/events';
  import { createTilesStore } from '$lib/stores/tiles';
  import { editMode } from '$lib/stores/edit';
  import { searchQuery } from '$lib/stores/search';
  import Header from '$lib/components/Header.svelte';
  import Group from '$lib/components/Group.svelte';
  import TileEditor from '$lib/components/TileEditor.svelte';
  import HiddenDrawer from '$lib/components/HiddenDrawer.svelte';
  import BookmarkDialog from '$lib/components/BookmarkDialog.svelte';
  import type { Tile } from '$lib/api/types';

  const tiles = createTilesStore(api);
  let editingTileId: string | null = null;
  let addBookmarkGroup: string | null = null;
  let unsub: (() => void) | null = null;

  onMount(async () => {
    await tiles.load();
    unsub = subscribeEvents((e) => tiles.applyEvent(e));
  });
  onDestroy(() => unsub?.());

  $: editingTile = editingTileId
    ? $tiles.tiles.find((t) => t.id === editingTileId) ?? null
    : null;

  // Filter logic.
  function matches(t: Tile, q: string): boolean {
    if (!q) return true;
    const ql = q.toLowerCase();
    return (
      t.name.toLowerCase().includes(ql) ||
      t.url.toLowerCase().includes(ql) ||
      (t.description ?? '').toLowerCase().includes(ql)
    );
  }

  $: filteredTiles = $tiles.tiles.filter((t) => matches(t, $searchQuery));
  $: groupedTiles = (() => {
    const m = new Map<string, Tile[]>();
    for (const t of filteredTiles) {
      if (!$editMode && t.hidden) continue;
      const arr = m.get(t.group) ?? [];
      arr.push(t);
      m.set(t.group, arr);
    }
    return m;
  })();
  $: visibleGroups = $tiles.groups
    .filter((g) => groupedTiles.has(g.id))
    .sort((a, b) => a.order - b.order);
  $: hiddenTiles = $tiles.tiles.filter((t) => t.hidden);

  async function onSaveTile(detail: import('$lib/api/types').TileOverride) {
    await api.patchConfig({ tiles: [detail] });
    editingTileId = null;
    await tiles.load();
  }

  async function onResetTile(detail: { id: string }) {
    console.warn('reset-to-auto not yet wired; use hide for v1', detail.id);
    editingTileId = null;
  }

  async function onHideTile(detail: { id: string }) {
    await api.patchConfig({ tiles: [{ id: detail.id, hidden: true }] });
    await tiles.load();
  }

  async function onRestoreTile(detail: { id: string }) {
    await api.patchConfig({ tiles: [{ id: detail.id, hidden: false }] });
    await tiles.load();
  }

  async function onCreateBookmark(b: import('$lib/api/types').Bookmark) {
    await api.addBookmark(b);
    addBookmarkGroup = null;
    await tiles.load();
  }

  async function onReorder(detail: { group: string; items: import('$lib/api/types').Tile[] }) {
    const overrides = detail.items.map((t, i) => ({
      id: t.id,
      order: i,
      group: detail.group,
    }));
    await api.patchConfig({ tiles: overrides });
    await tiles.load();
  }
</script>

<Header title="Dashboard" />

<main>
  {#each visibleGroups as g (g.id)}
    <Group
      group={g}
      tiles={groupedTiles.get(g.id) ?? []}
      editing={$editMode}
      on:editTile={(e) => (editingTileId = e.detail.id)}
      on:hideTile={(e) => onHideTile(e.detail)}
      on:addBookmark={(e) => (addBookmarkGroup = e.detail.group)}
      on:reorder={(e) => onReorder(e.detail)}
    />
  {/each}

  {#if visibleGroups.length === 0}
    <p class="empty">
      {#if $searchQuery}No tiles match "{$searchQuery}".
      {:else}No tiles discovered yet.
      {/if}
    </p>
  {/if}
</main>

{#if $editMode}
  <HiddenDrawer tiles={hiddenTiles} onrestore={(detail) => onRestoreTile(detail)} />
{/if}

{#if editingTile}
  <TileEditor
    tile={editingTile}
    groups={$tiles.groups}
    onsave={(detail) => onSaveTile(detail)}
    onreset={(detail) => onResetTile(detail)}
    onclose={() => (editingTileId = null)}
  />
{/if}

{#if addBookmarkGroup}
  <BookmarkDialog
    group={addBookmarkGroup}
    oncreate={(detail) => onCreateBookmark(detail)}
    onclose={() => (addBookmarkGroup = null)}
  />
{/if}

<style>
  main { padding: 24px; }
  .empty { color: var(--fg-muted); text-align: center; padding: 60px; }
</style>
