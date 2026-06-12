<!-- src/routes/vineyard/[id]/+layout.svelte -->
<script lang="ts">
  import { page } from '$app/stores';
  $: vineyard = $page?.data?.vineyard;
</script>

{#if vineyard}
  <nav style="max-width:900px;margin:0 auto;padding:0.75rem 1rem;border-bottom:1px solid #eee;display:flex;align-items:center;gap:0.5rem">
    <a href="/vineyard/{vineyard.id}" style="color:#555;font-size:0.9rem;text-decoration:none">
      {vineyard.name}
    </a>
    <span style="color:#ccc">/</span>
    <slot />
  </nav>
{/if}
<slot />
