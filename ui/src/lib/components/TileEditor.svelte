<script lang="ts">
  import IconPicker from './IconPicker.svelte';
  import type { Group, Tile, TileOverride } from '$lib/api/types';

  export let tile: Tile;
  export let groups: Group[];
  export let onsave: ((detail: TileOverride) => void) | undefined = undefined;
  export let onreset: ((detail: { id: string }) => void) | undefined = undefined;
  export let onclose: (() => void) | undefined = undefined;

  let name = tile.name;
  let url = tile.url;
  let icon = tile.icon;
  let description = tile.description ?? '';
  let group = tile.group;
  let insecure = !!tile.insecureSkipVerify;

  function save() {
    onsave?.({
      id: tile.id,
      name, url, icon, description, group,
      insecureSkipVerify: insecure,
    });
  }
</script>

<aside class="editor">
  <header>
    <h3>Edit tile</h3>
    <button aria-label="Close" on:click={onclose}>×</button>
  </header>

  <label>Name<input bind:value={name} aria-label="Name" /></label>
  <label>URL<input bind:value={url} aria-label="URL" /></label>
  <label>Description<input bind:value={description} aria-label="Description" /></label>
  <label>Group
    <select bind:value={group} aria-label="Group">
      {#each groups as g}<option value={g.id}>{g.name}</option>{/each}
    </select>
  </label>
  <div>
    <div class="label">Icon</div>
    <IconPicker value={icon} onchange={(detail) => (icon = detail.value)} />
  </div>
  <label class="check">
    <input type="checkbox" bind:checked={insecure} /> Insecure TLS (skip cert verification)
  </label>

  {#if tile.k8s}
    <div class="k8s">
      <div class="label">Kubernetes</div>
      <dl>
        <dt>Namespace</dt><dd>{tile.k8s.namespace}</dd>
        <dt>HTTPRoute</dt><dd>{tile.k8s.httpRouteName}</dd>
        <dt>Gateways</dt>
        <dd>
          {#each tile.k8s.gatewayRefs as g}
            <span>{g.namespace}/{g.name}</span>
          {/each}
        </dd>
      </dl>
    </div>
  {/if}

  <footer>
    <button on:click={() => onreset?.({ id: tile.id })}>Reset to auto</button>
    <button class="primary" on:click={save}>Save</button>
  </footer>
</aside>

<style>
  .editor {
    position: fixed; right: 0; top: 0; bottom: 0;
    width: 360px; background: var(--bg);
    border-left: 1px solid var(--border);
    padding: 16px; display: flex; flex-direction: column; gap: 12px;
    overflow-y: auto;
  }
  header, footer { display: flex; justify-content: space-between; gap: 8px; }
  h3 { margin: 0; }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--fg-muted); }
  label input, label select {
    padding: 6px 8px; border-radius: 4px;
    border: 1px solid var(--border); background: var(--bg-card); color: var(--fg);
  }
  .check { flex-direction: row; align-items: center; }
  .label { font-size: 12px; color: var(--fg-muted); margin-bottom: 4px; }
  .k8s dl { display: grid; grid-template-columns: max-content 1fr; gap: 4px 12px; font-size: 13px; }
  .k8s dt { color: var(--fg-muted); }
  button.primary { background: var(--accent); color: var(--accent-fg); border: none; padding: 6px 12px; border-radius: 4px; cursor: pointer; }
  button { background: none; border: 1px solid var(--border); color: var(--fg); padding: 6px 12px; border-radius: 4px; cursor: pointer; }
</style>
