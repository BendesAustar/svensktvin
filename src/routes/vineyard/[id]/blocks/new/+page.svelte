<!-- src/routes/vineyard/[id]/blocks/new/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import { page } from '$app/stores';
  import type { ActionData, PageData } from './$types';
  export let form: ActionData;
  export let data: PageData;

  let searchQuery = '';
  let searchResults: Array<{ id: number; name: string; score: number; piwi: boolean; color: string }> = [];
  let highConfidence = false;
  let selectedVarietyId: number | null = null;
  let customVarietyName = '';
  let searching = false;
  let searchError = '';

  async function searchVarieties(q: string) {
    if (!q || q.length < 2) {
      searchResults = [];
      highConfidence = false;
      return;
    }
    searching = true;
    searchError = '';
    try {
      const res = await fetch(`/api/varieties/search?q=${encodeURIComponent(q)}`);
      const data = await res.json() as { matches: typeof searchResults; high_confidence: boolean };
      searchResults = data.matches;
      highConfidence = data.high_confidence;
      if (data.high_confidence && data.matches.length > 0) {
        selectVariety(data.matches[0].id, data.matches[0].name);
      } else {
        selectedVarietyId = null;
      }
    } catch {
      searchError = 'Sökningen misslyckades.';
      searchResults = [];
    } finally {
      searching = false;
    }
  }

  function selectVariety(id: number, name: string) {
    selectedVarietyId = id;
    searchQuery = name;
    searchResults = [];
    highConfidence = false;
    customVarietyName = '';
  }

  function useCustom() {
    selectedVarietyId = null;
    customVarietyName = searchResults[0]?.name ?? '';
    searchResults = [];
    highConfidence = false;
  }
</script>

<svelte:head><title>Nytt block — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/vineyard/{data.vineyard.id}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Nytt block</h1>

  <form method="POST" use:enhance>
    {#if form?.error}
      <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
    {/if}

    <label for="block_name" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Blocknamn <span style="color:#c62828">*</span></label>
    <input id="block_name" type="text" name="block_name" required
      style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

    <label for="variety-search" style="display:block;margin-top:1rem;margin-bottom:0.25rem;font-size:0.9rem">Sort <span style="color:#c62828">*</span></label>
    <div style="display:flex;gap:0.5rem;margin-bottom:0.5rem">
      <input
        id="variety-search"
        type="text"
        placeholder="Sök sort..."
        value={searchQuery}
        oninput={(e) => { searchQuery = (e.target as HTMLInputElement).value; searchVarieties(searchQuery); }}
        style="flex:1;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />
    </div>

    {#if searching}
      <p style="color:#888;font-size:0.9rem">Söker...</p>
    {:else if searchError}
      <p style="color:#c62828;font-size:0.9rem">{searchError}</p>
    {:else if selectedVarietyId !== null}
      <p style="color:#2d6a2d;font-size:0.9rem;margin:0.25rem 0">✓ {searchQuery}</p>
    {:else if searchResults.length > 0}
      <ul style="list-style:none;padding:0;margin-bottom:0.5rem;border:1px solid #eee;border-radius:4px">
        {#each searchResults as result}
          <button type="button" tabindex="0" style="display:block;width:100%;text-align:left;padding:0.5rem 0.75rem;border-bottom:1px solid #f0f0f0;cursor:pointer;background:none;border-left:none;border-right:none;border-top:none"
              onclick={() => selectVariety(result.id, result.name)}>
            {result.name}
            <span style="color:#888;font-size:0.8rem"> ({result.color}{#if result.piwi} · PIWI{/if})</span>
          </button>
        {/each}
      </ul>
      <button type="button" onclick={useCustom}
        style="padding:0.4rem 0.8rem;background:none;border:1px solid #ccc;border-radius:4px;cursor:pointer;font-size:0.85rem">
        Ingen av dessa — använd detta namn
      </button>
    {:else if searchQuery.length >= 2}
      <p style="color:#888;font-size:0.9rem">Inga träffar. Sorten läggs till för granskning.</p>
    {/if}

    <!-- Hidden input for variety_id -->
    <input type="hidden" name="variety_id" value={selectedVarietyId ?? ''} />
    <input type="hidden" name="variety_name" value={customVarietyName} />

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-top:1rem;margin-bottom:1rem">
      <legend style="font-weight:600;padding:0 0.5rem">Blockinformation</legend>

      <label for="area_ha" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Area (ha) <span style="color:#c62828">*</span></label>
      <input id="area_ha" type="number" name="area_ha" step="0.001" min="0.01" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="vine_count" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Vinstockar (valfritt)</label>
      <input id="vine_count" type="number" name="vine_count" min="0"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="planting_year" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Planteringsår</label>
      <input id="planting_year" type="number" name="planting_year" min="1800" max="2030"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="training_system" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Uppbindningssystem</label>
      <input id="training_system" type="text" name="training_system" placeholder="t.ex. VSP, GDC"
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
      <input id="slope_degrees" type="number" name="slope_degrees" step="0.1" min="0" max="90"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="elevation_m" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Höjmö (m)</label>
      <input id="elevation_m" type="number" name="elevation_m" min="0"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <button type="submit"
      style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
      Skapa block
    </button>
  </form>
</main>
