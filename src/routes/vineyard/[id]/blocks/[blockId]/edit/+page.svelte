<!-- src/routes/vineyard/[id]/blocks/[blockId]/edit/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { ActionData, PageData } from './$types';
  export let form: ActionData;
  export let data: PageData;

  const { block, varieties } = data;
  const { id: vineyardId } = block;
</script>

<svelte:head><title>Redigera block: {block.block_name} — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/vineyard/{vineyardId}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Redigera block</h1>

  <form method="POST" use:enhance>
    {#if form?.error}
      <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
    {/if}

    <label for="block_name" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Blocknamn <span style="color:#c62828">*</span></label>
    <input id="block_name" type="text" name="block_name" required value={block.block_name}
      style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

    <label for="variety_id" style="display:block;margin-top:1rem;margin-bottom:0.25rem;font-size:0.9rem">Sort <span style="color:#c62828">*</span></label>
    <select id="variety_id" name="variety_id" required
      style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
      <option value="">Välj sort</option>
      {#each varieties as variety}
        <option value={variety.id}>
          {variety.name} ({variety.color}{#if variety.piwi} · PIWI{/if})
        </option>
      {/each}
    </select>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-top:1rem;margin-bottom:1rem">
      <legend style="font-weight:600;padding:0 0.5rem">Blockinformation</legend>

      <label for="area_ha" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Area (ha) <span style="color:#c62828">*</span></label>
      <input id="area_ha" type="number" name="area_ha" step="0.001" min="0.01" required value={block.area_ha}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="vine_count" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Vinstockar</label>
      <input id="vine_count" type="number" name="vine_count" min="0" value={block.vine_count ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="planting_year" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Planteringsår</label>
      <input id="planting_year" type="number" name="planting_year" min="1800" max="2030" value={block.planting_year ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="training_system" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Uppbindningssystem</label>
      <input id="training_system" type="text" name="training_system" value={block.training_system ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="aspect" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Exposition</label>
      <select id="aspect" name="aspect"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj</option>
        <option>N</option>
        <option>NE</option>
        <option>E</option>
        <option>SE</option>
        <option>S</option>
        <option>SW</option>
        <option>W</option>
        <option>NW</option>
      </select>

      <label for="slope_degrees" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Sluttning (grader)</label>
      <input id="slope_degrees" type="number" name="slope_degrees" step="0.1" min="0" max="90" value={block.slope_degrees ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="elevation_m" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Höjmö (m)</label>
      <input id="elevation_m" type="number" name="elevation_m" min="0" value={block.elevation_m ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <button type="submit"
      style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
      Spara ändringar
    </button>
  </form>
</main>
