<script lang="ts">
  import IconPicker from './IconPicker.svelte';
  import type { Bookmark } from '$lib/api/types';

  export let group: string;
  export let oncreate: ((detail: Bookmark) => void) | undefined = undefined;
  export let onclose: (() => void) | undefined = undefined;

  let name = '';
  let url = '';
  let icon = '';

  function slugify(s: string): string {
    return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
  }

  function submit() {
    if (!name || !url) return;
    oncreate?.({
      id: slugify(name) + '-' + Math.random().toString(36).slice(2, 6),
      name, url, icon, group,
    });
  }
</script>

<div class="dialog" role="dialog">
  <h3>Add bookmark to <em>{group}</em></h3>
  <label>Name<input bind:value={name} aria-label="Name" /></label>
  <label>URL<input bind:value={url} aria-label="URL" /></label>
  <div>
    <div class="label">Icon</div>
    <IconPicker value={icon} onchange={(detail) => (icon = detail.value)} />
  </div>
  <footer>
    <button on:click={() => onclose?.()}>Cancel</button>
    <button class="primary" on:click={submit}>Add</button>
  </footer>
</div>

<style>
  .dialog {
    position: fixed; inset: 0; margin: auto;
    width: 380px; height: fit-content;
    padding: 16px; background: var(--bg);
    border: 1px solid var(--border); border-radius: 8px;
    display: flex; flex-direction: column; gap: 10px;
    z-index: 100;
  }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--fg-muted); }
  label input { padding: 6px 8px; border-radius: 4px; border: 1px solid var(--border); background: var(--bg-card); color: var(--fg); }
  .label { font-size: 12px; color: var(--fg-muted); }
  footer { display: flex; justify-content: flex-end; gap: 8px; }
  button { background: none; border: 1px solid var(--border); color: var(--fg); padding: 6px 12px; border-radius: 4px; cursor: pointer; }
  button.primary { background: var(--accent); color: var(--accent-fg); border: none; }
</style>
