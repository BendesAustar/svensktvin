<!-- src/routes/vineyard/[id]/+page.svelte -->
<script lang="ts">
  import type { PageData } from './$types';
  export let data: PageData;

  const { vineyard, blocks, benchmarkTeaser, role } = data;
</script>

<svelte:head>
  <title>{vineyard.name} — Svenskt Vin</title>
</svelte:head>

<main style="max-width:900px;margin:0 auto;padding:1rem;font-family:sans-serif">
  <!-- Vineyard header -->
  <div style="margin-bottom:2rem">
    <h1 style="margin:0 0 0.25rem 0">{vineyard.name}</h1>
    <p style="color:#555;margin:0">
      {vineyard.county} · {vineyard.municipality}
      {#if vineyard.established_year} · Startad {vineyard.established_year}{/if}
      {#if vineyard.total_area_ha} · {vineyard.total_area_ha} ha{/if}
    </p>
    <p style="color:#555;margin:0.25rem 0">
      {#if vineyard.organic}🌿 Ekologisk{/if}
      {#if vineyard.organic && vineyard.biodynamic} · {/if}
      {#if vineyard.biodynamic}🌀 Biodynamisk{/if}
    </p>
    {#if role === 'owner'}
      <a href="/vineyard/{vineyard.id}/settings" style="font-size:0.85rem;color:#2d6a2d">⚙️ Inställningar</a>
    {/if}
  </div>

  <!-- Benchmark teaser -->
  {#if benchmarkTeaser}
    <div style="background:#e8f5e9;padding:1rem;border-radius:4px;margin-bottom:2rem">
      <h3 style="margin:0 0 0.5rem 0;font-size:1rem">Benchmark — {benchmarkTeaser.variety_name}</h3>
      <p style="margin:0;font-size:0.9rem">
        Din skörd: <strong>{benchmarkTeaser.user_yield_kg_ha}</strong> kg/ha
        <span style="color:#888"> · {benchmarkTeaser.vineyard_count} vingårdar i {vineyard.county}</span>
      </p>
    </div>
  {/if}

  <!-- Blocks -->
  <div style="margin-bottom:1.5rem">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem">
      <h2 style="margin:0">Block</h2>
      <a href="/vineyard/{vineyard.id}/blocks/new"
        style="padding:0.5rem 1rem;background:#2d6a2d;color:#fff;border-radius:4px;text-decoration:none;font-size:0.9rem">
        + Nytt block
      </a>
    </div>

    {#if blocks.length === 0}
      <p style="color:#888;text-align:center;padding:2rem">Inga block ännu. Skapa ditt första block!</p>
    {:else}
      <table style="width:100%;border-collapse:collapse">
        <thead>
          <tr style="border-bottom:2px solid #eee;text-align:left">
            <th style="padding:0.5rem;font-size:0.85rem;color:#555">Namn</th>
            <th style="padding:0.5rem;font-size:0.85rem;color:#555">Sort</th>
            <th style="padding:0.5rem;font-size:0.85rem;color:#555">Area</th>
            <th style="padding:0.5rem;font-size:0.85rem;color:#555">Senaste skörden</th>
            <th style="padding:0.5rem;font-size:0.85rem;color:#555"></th>
          </tr>
        </thead>
        <tbody>
          {#each blocks as block}
            <tr style="border-bottom:1px solid #f0f0f0">
              <td style="padding:0.75rem 0.5rem">
                <strong>{block.block_name}</strong>
                {#if !block.is_active}
                  <span style="color:#999;font-size:0.8rem"> (inaktiv)</span>
                {/if}
              </td>
              <td style="padding:0.75rem 0.5rem">
                <span style="{block.variety_status === 'approved' ? 'color:#2d6a2d' : 'color:#856404'}">
                  {block.variety_name}
                </span>
                {#if block.variety_status === 'review_needed'}
                  <span style="font-size:0.75rem;color:#856404"> (granskas)</span>
                {/if}
              </td>
              <td style="padding:0.75rem 0.5rem;color:#555">{block.area_ha} ha</td>
              <td style="padding:0.75rem 0.5rem;color:#555">
                {#if block.latest_harvest}
                  {block.latest_harvest.harvest_year}: {block.latest_harvest.yield_kg} kg
                {:else}
                  <span style="color:#ccc">—</span>
                {/if}
              </td>
              <td style="padding:0.75rem 0.5rem;text-align:right">
                <a href="/vineyard/{vineyard.id}/blocks/{block.id}/edit" style="color:#2d6a2d;font-size:0.85rem">Redigera</a>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <!-- Harvest actions -->
  <a href="/vineyard/{vineyard.id}/harvest/new"
    style="display:inline-block;padding:0.75rem 1.5rem;background:#2d6a2d;color:#fff;border-radius:4px;text-decoration:none;font-size:1rem;margin-right:0.5rem">
    🌾 Registrera skörd
  </a>
</main>
