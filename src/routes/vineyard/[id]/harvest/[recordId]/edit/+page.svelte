<!-- src/routes/vineyard/[id]/harvest/[recordId]/edit/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import { page } from '$app/stores';
  import type { ActionData, PageData } from './$types';
  export let form: ActionData;
  export let data: PageData;

  const { record, blocks } = data;
</script>

<svelte:head><title>Redigera skörd {record.harvest_year} — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/vineyard/{$page.params.id}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Redigera skörd {record.harvest_year}</h1>

  <form method="POST" use:enhance>
    {#if form?.error}
      <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
    {/if}

    <label for="block_id" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Block <span style="color:#c62828">*</span></label>
    <select id="block_id" name="block_id" required
      style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
      <option value="">Välj block</option>
      {#each blocks as block}
        <option value={block.id} selected={block.id === record.block_id}>
          {block.block_name} ({block.variety_name})
        </option>
      {/each}
    </select>

    <label for="harvest_year" style="display:block;margin-top:1rem;margin-bottom:0.25rem;font-size:0.9rem">Skördeår <span style="color:#c62828">*</span></label>
    <input id="harvest_year" type="number" name="harvest_year" min="1800" max="2100" required value={record.harvest_year}
      style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

    <label for="harvest_date" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Skördedatum</label>
    <input id="harvest_date" type="date" name="harvest_date" value={record.harvest_date ?? ''}
      style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-top:1rem;margin-bottom:1rem">
      <legend style="font-weight:600;padding:0 0.5rem">Skördedata</legend>

      <label for="yield_kg" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Skördevikt (kg) <span style="color:#c62828">*</span></label>
      <input id="yield_kg" type="number" name="yield_kg" step="0.01" min="0.01" required value={record.yield_kg}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="brix" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Brix</label>
      <input id="brix" type="number" name="brix" step="0.1" min="0" value={record.brix ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="acid_g_l" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Total syra (g/L)</label>
      <input id="acid_g_l" type="number" name="acid_g_l" step="0.01" min="0" value={record.acid_g_l ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="vine_health_rating" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Hälsa (1–5)</label>
      <select id="vine_health_rating" name="vine_health_rating"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj</option>
        <option value="1" selected={record.vine_health_rating === 1}>1 — Dålig</option>
        <option value="2" selected={record.vine_health_rating === 2}>2 — Under medel</option>
        <option value="3" selected={record.vine_health_rating === 3}>3 — Medel</option>
        <option value="4" selected={record.vine_health_rating === 4}>4 — God</option>
        <option value="5" selected={record.vine_health_rating === 5}>5 — Exceptionell</option>
      </select>

      <label for="notes" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Anteckningar</label>
      <textarea id="notes" name="notes" rows="3"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box;resize:vertical">{record.notes ?? ''}</textarea>
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1rem">
      <legend style="font-weight:600;padding:0 0.5rem">Fate-of-fruit (valfritt)</legend>

      <label for="still_wine_l" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Stillvin (L)</label>
      <input id="still_wine_l" type="number" name="still_wine_l" step="0.1" min="0" value={record.still_wine_l ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="sparkling_l" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Petillant/Brut (L)</label>
      <input id="sparkling_l" type="number" name="sparkling_l" step="0.1" min="0" value={record.sparkling_l ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="juice_l" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Juice (L)</label>
      <input id="juice_l" type="number" name="juice_l" step="0.1" min="0" value={record.juice_l ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="sold_kg" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Sålt (kg)</label>
      <input id="sold_kg" type="number" name="sold_kg" step="0.01" min="0" value={record.sold_kg ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="discarded_kg" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Kvar/kastad (kg)</label>
      <input id="discarded_kg" type="number" name="discarded_kg" step="0.01" min="0" value={record.discarded_kg ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <button type="submit"
      style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
      Spara ändringar
    </button>
  </form>
</main>
