<script lang="ts">
  import Icon from './Icon.svelte';
  import { ICON_SLUGS } from '$lib/icons';

  export let value: string = '';
  export let onchange: ((detail: { value: string }) => void) | undefined = undefined;

  $: query = value.toLowerCase();
  $: suggestions = !query || value.startsWith('http')
    ? []
    : ICON_SLUGS.filter((s) => s.includes(query)).slice(0, 8);

  function handleInput(e: Event) {
    const v = (e.target as HTMLInputElement).value;
    value = v;
    onchange?.({ value: v });
  }

  function pick(slug: string) {
    value = slug;
    onchange?.({ value: slug });
  }
</script>

<div class="picker">
  <div class="row">
    <Icon slug={value} alt="preview" />
    <input
      type="text"
      placeholder="icon slug or URL"
      value={value}
      on:input={handleInput}
    />
  </div>
  {#if suggestions.length > 0}
    <ul class="suggestions" role="listbox">
      {#each suggestions as s}
        <li role="option" on:click={() => pick(s)}>
          <Icon slug={s} alt={s} />
          <span>{s}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .picker { display: flex; flex-direction: column; gap: 8px; }
  .row { display: flex; gap: 8px; align-items: center; }
  .row input { flex: 1; padding: 6px 8px; border-radius: 4px; border: 1px solid var(--border); background: var(--bg-card); color: var(--fg); }
  .suggestions { list-style: none; padding: 4px; margin: 0; border: 1px solid var(--border); border-radius: 6px; max-height: 220px; overflow-y: auto; }
  .suggestions li { display: flex; gap: 8px; align-items: center; padding: 4px 6px; cursor: pointer; border-radius: 4px; }
  .suggestions li:hover { background: var(--bg-card-hover); }
</style>
