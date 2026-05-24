<script lang="ts">
  // Renders /icons/<slug>.png (served by the Go binary). If `slug` is a
  // full http(s) URL it is used directly. Falls back to a globe SVG on
  // load error.
  export let slug: string = '';
  export let alt: string = '';
  let errored = false;
  $: src = slug.startsWith('http') ? slug : `/icons/${slug}.png`;
</script>

{#if !slug || errored}
  <svg class="icon icon-fallback" viewBox="0 0 24 24" aria-label={alt}>
    <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <path d="M2 12h20M12 2c3 3 3 17 0 20M12 2c-3 3-3 17 0 20" fill="none" stroke="currentColor" stroke-width="1.5"/>
  </svg>
{:else}
  <img class="icon" {src} alt={alt} on:error={() => (errored = true)} />
{/if}

<style>
  .icon { width: 40px; height: 40px; border-radius: 8px; object-fit: contain; }
  .icon-fallback { color: var(--fg-muted); }
</style>
