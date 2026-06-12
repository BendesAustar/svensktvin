<!-- src/routes/vineyard/[id]/settings/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData } from './$types';
  export let data: PageData;

  const { vineyard, members } = data;
</script>

<svelte:head><title>Inställningar: {vineyard.name} — Svenskt Vin</title></svelte:head>

<main style="max-width:700px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/vineyard/{vineyard.id}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Inställningar</h1>

  <form method="POST" use:enhance>
    <input type="hidden" name="action" value="update_vineyard" />

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Vingårdsuppgifter</legend>

      <label for="name" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Namn <span style="color:#c62828">*</span></label>
      <input id="name" type="text" name="name" required value={vineyard.name}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="county" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Län <span style="color:#c62828">*</span></label>
      <select id="county" name="county" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj län</option>
        <option>Skåne</option><option>Blekinge</option><option>Gotland</option>
        <option>Kalmar</option><option>Kronoberg</option><option>Östergötland</option>
        <option>Västergötland</option><option>Stockholm</option><option>Uppsala</option>
        <option>Västmanland</option><option>Södermanland</option><option>Halland</option>
        <option>Gävleborg</option><option>Värmland</option><option>Dalarna</option>
        <option>Örebro</option>
      </select>

      <label for="municipality" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Kommun</label>
      <input id="municipality" type="text" name="municipality" value={vineyard.municipality}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="established_year" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Uppstartsår</label>
      <input id="established_year" type="number" name="established_year" min="1800" max="2030" value={vineyard.established_year ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="total_area_ha" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Total area (ha)</label>
      <input id="total_area_ha" type="number" name="total_area_ha" step="0.01" min="0" value={vineyard.total_area_ha ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Produktionsmetod</legend>
      <label style="display:flex;align-items:center;margin-bottom:0.5rem;cursor:pointer">
        <input type="checkbox" name="organic" value="on" checked={vineyard.organic} style="margin-right:0.5rem;font-size:1.2rem" />
        Ekologisk
      </label>
      <label style="display:flex;align-items:center;cursor:pointer">
        <input type="checkbox" name="biodynamic" value="on" checked={vineyard.biodynamic} style="margin-right:0.5rem;font-size:1.2rem" />
        Biodynamisk
      </label>
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Medlemmar</legend>
      {#if members.length > 0}
        <ul style="list-style:none;padding:0;margin:0">
          {#each members as m}
            <li style="padding:0.5rem 0;border-bottom:1px solid #f0f0f0">
              <strong>{m.name}</strong> ({m.email}) — {m.role}
            </li>
          {/each}
        </ul>
      {:else}
        <p style="color:#888;margin:0">Inga medlemmar ännu.</p>
      {/if}
    </fieldset>

    <button type="submit"
      style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
      Spara ändringar
    </button>
  </form>
</main>
