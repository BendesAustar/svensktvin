<!-- src/routes/vineyard/[id]/harvest/new/+page.svelte -->
<script lang="ts">
    import { enhance } from '$app/forms';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import type { ActionData, PageData } from './$types';

  const { form, data }: { form: ActionData; data: PageData } = $props();
  const { blocks, years, lock } = data;
  let loading = $state(true);

  onMount(async () => {
    const blockId = $page.url.searchParams.get('block_id');
    if (!blockId || lock) {
      loading = false;
      return;
    }

    // Try to acquire lock if not already held
    const res = await fetch(`/vineyard/{$page.params.id}/blocks/${blockId}/harvest/lock`, {
      method: 'POST'
    }).catch(() => null);

    if (res) {
      const json = await res.json().catch(() => ({}));
      if (res.status === 409) {
        alert(json.message || 'Blocket redigeras just nu av någon annan. Välj ett annat block.');
        await goto(`/vineyard/{$page.params.id}`);
      }
    }
    loading = false;
  });

  function lockBlock() {
    const blockId = $page.url.searchParams.get('block_id');
    if (!blockId) return;

    fetch(`/vineyard/{$page.params.id}/blocks/${blockId}/harvest/lock`, {
      method: 'POST'
    }).then(async (res) => {
      if (res.ok) {
        // Reload to pick up new lock state
        await goto(`/vineyard/{$page.params.id}/harvest/new?block_id=${blockId}`);
      } else {
        const body = await res.json().catch(() => ({}));
        alert(body.message || 'Kunde inte låsa blocket.');
      }
    }).catch(() => alert('Något gick fel. Försök igen.'));
  }

  function unlockBlock() {
    const blockId = $page.url.searchParams.get('block_id');
    if (!blockId) return;

    fetch(`/vineyard/{$page.params.id}/blocks/${blockId}/harvest/lock`, {
      method: 'DELETE'
    }).then(async (res) => {
      if (res.ok) {
        await goto(`/vineyard/{$page.params.id}`);
      } else {
        const body = await res.json().catch(() => ({}));
        alert(body.message || 'Kunde inte låsa upp blocket.');
      }
    }).catch(() => alert('Något gick fel. Försök igen.'));
  }

  function relockBlock() {
    const blockId = $page.url.searchParams.get('block_id');
    if (!blockId) return;

    fetch(`/vineyard/{$page.params.id}/blocks/${blockId}/harvest/lock`, {
      method: 'POST'
    }).then(async (res) => {
      if (res.ok) {
        alert('Låset har förlängts.');
      } else {
        const body = await res.json().catch(() => ({}));
        alert(body.message || 'Kunde inte förlänga låset.');
      }
    }).catch(() => alert('Något gick fel. Försök igen.'));
  }

  function minutesLeft() {
    if (!lock) return null;
    const diff = new Date(lock.expiresAt).getTime() - Date.now();
    return Math.max(0, Math.ceil(diff / 60000));
  }
</script>

<svelte:head><title>Registrera skörd — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/vineyard/{$page.params.id}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Registrera skörd</h1>

  {#if loading}
    <div style="padding:2rem;text-align:center;color:#888">Läser in...</div>
  {:else}
    {#if lock}
      <div style="background:#e3f2fd;padding:0.75rem 1rem;border-radius:4px;margin-bottom:1rem;display:flex;justify-content:space-between;align-items:center;gap:0.5rem;flex-wrap:wrap">
        <span style="font-size:0.9rem;color:#1565c0">
          🔒 Blocket är låst av dig · {minutesLeft()} min kvar
        </span>
        <div style="display:flex;gap:0.5rem">
          <button onclick={relockBlock}
            style="padding:0.3rem 0.75rem;background:#fff;color:#1565c0;border:1px solid #1565c0;border-radius:3px;font-size:0.85rem;cursor:pointer">
            Förläng lås
          </button>
          <button onclick={unlockBlock}
            style="padding:0.3rem 0.75rem;background:#fff;color:#c62828;border:1px solid #c62828;border-radius:3px;font-size:0.85rem;cursor:pointer">
            Lås upp
          </button>
        </div>
      </div>
    {:else if form?.error && form.error.includes('lå')}
      <div style="background:#ffebee;padding:0.75rem 1rem;border-radius:4px;margin-bottom:1rem">
        <p style="margin:0;color:#c62828;font-size:0.9rem">{form.error}</p>
        <button onclick={lockBlock}
          style="margin-top:0.5rem;padding:0.4rem 1rem;background:#c62828;color:#fff;border:none;border-radius:3px;font-size:0.85rem;cursor:pointer">
          Försök låsa igen
        </button>
      </div>
    {/if}

    {#if form?.error && !form.error.includes('lå')}
      <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
    {/if}

    <form method="POST" use:enhance>
      {#if form?.error}
        <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
      {/if}

      <label for="block_id" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Block <span style="color:#c62828">*</span></label>
      <select id="block_id" name="block_id" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj block</option>
        {#each blocks as block}
          <option value={block.id}>{block.block_name} ({block.variety_name})</option>
        {/each}
      </select>

      <label for="harvest_year" style="display:block;margin-top:1rem;margin-bottom:0.25rem;font-size:0.9rem">Skördeår <span style="color:#c62828">*</span></label>
      <select id="harvest_year" name="harvest_year" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj år</option>
        {#each years as year}
          <option value={year}>{year}</option>
        {/each}
      </select>

      <label for="harvest_date" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Skördedatum</label>
      <input id="harvest_date" type="date" name="harvest_date"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-top:1rem;margin-bottom:1rem">
        <legend style="font-weight:600;padding:0 0.5rem">Skördedata</legend>

        <label for="yield_kg" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Skördevikt (kg) <span style="color:#c62828">*</span></label>
        <input id="yield_kg" type="number" name="yield_kg" step="0.01" min="0.01" required
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="brix" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Brix (sockerhalt)</label>
        <input id="brix" type="number" name="brix" step="0.1" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="acid_g_l" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Total syra (g/L)</label>
        <input id="acid_g_l" type="number" name="acid_g_l" step="0.01" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="vine_health_rating" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Vinstockarnas hälsa (1–5)</label>
        <select id="vine_health_rating" name="vine_health_rating"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
          <option value="">Välj</option>
          <option value="1">1 — Dålig</option>
          <option value="2">2 — Under medel</option>
          <option value="3">3 — Medel</option>
          <option value="4">4 — God</option>
          <option value="5">5 — Exceptionell</option>
        </select>

        <label for="notes" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Anteckningar</label>
        <textarea id="notes" name="notes" rows="3"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box;resize:vertical"></textarea>
      </fieldset>

      <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1rem">
        <legend style="font-weight:600;padding:0 0.5rem">Fate-of-fruit (valfritt)</legend>

        <label for="still_wine_l" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Stillvin (L)</label>
        <input id="still_wine_l" type="number" name="still_wine_l" step="0.1" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="sparkling_l" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Petillant/Brut (L)</label>
        <input id="sparkling_l" type="number" name="sparkling_l" step="0.1" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="juice_l" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Juice (L)</label>
        <input id="juice_l" type="number" name="juice_l" step="0.1" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="sold_kg" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Sålt (kg)</label>
        <input id="sold_kg" type="number" name="sold_kg" step="0.01" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

        <label for="discarded_kg" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Kvar/kastad (kg)</label>
        <input id="discarded_kg" type="number" name="discarded_kg" step="0.01" min="0"
          style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
      </fieldset>

      <button type="submit"
        style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
        Spara skörd
      </button>
    </form>
  {/if}
</main>
