<!-- src/routes/+page.svelte -->
<script lang="ts">
  import type { PageData } from './$types';
  export let data: PageData;
</script>

<svelte:head><title>Välj vingård — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="margin-bottom:0.25rem;font-size:1.3rem">🍷 Svenskt Vin</h1>
  <p style="color:#555;margin-bottom:1rem">Välj vingård</p>
  {#each data.vineyards as v}
    <a href="/vineyard/{v.vineyard_id}" style="display:block;padding:0.75rem;margin-bottom:0.5rem;background:#f9f9f9;border:1px solid #ddd;border-radius:4px;text-decoration:none;color:#333">
      {v.vineyard_name} <span style="color:#888;font-size:0.85rem">({v.role})</span>
    </a>
  {/each}
</main>
