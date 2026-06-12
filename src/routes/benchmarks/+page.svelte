<!-- src/routes/benchmarks/+page.svelte -->
<script lang="ts">
  import type { PageData } from './$types';
  export let data: PageData;

  const { rows, years, varieties, selectedYear, selectedVariety } = data;

  function applyFilter(year: string, variety: string) {
    const p = new URLSearchParams();
    if (year) p.set('year', year);
    if (variety) p.set('variety', variety);
    const qs = p.toString();
    window.location.href = `/benchmarks${qs ? '?' + qs : ''}`;
  }
</script>

<svelte:head><title>Jämförelsedata — Svenskt Vin</title></svelte:head>

<main style="max-width:900px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/" style="color:#555;font-size:0.9rem;text-decoration:none">← Hem</a>
  <h1 style="margin:0.5rem 0">Jämförelsedata</h1>
  <p class="hint" style="color:#666;margin-bottom:1.5rem">
    Visar enbart kombinationer med minst 3 vingårdar. Din avkastning markeras i grönt.
  </p>

  <div class="filters" style="display:flex;gap:1rem;margin-bottom:1.5rem;flex-wrap:wrap">
    <label style="display:flex;align-items:center;gap:0.5rem;font-size:0.9rem">
      År
      <select
        onchange={e => applyFilter(e.currentTarget.value, selectedVariety ?? '')}
        style="padding:0.4rem 0.6rem;border:1px solid #ccc;border-radius:4px;font-size:0.9rem"
      >
        <option value="">Alla år</option>
        {#each years as y}
          <option value={y} selected={y === selectedYear}>{y}</option>
        {/each}
      </select>
    </label>

    <label style="display:flex;align-items:center;gap:0.5rem;font-size:0.9rem">
      Sort
      <select
        onchange={e => applyFilter(selectedYear?.toString() ?? '', e.currentTarget.value)}
        style="padding:0.4rem 0.6rem;border:1px solid #ccc;border-radius:4px;font-size:0.9rem"
      >
        <option value="">Alla sorter</option>
        {#each varieties as v}
          <option value={v} selected={v === selectedVariety}>{v}</option>
        {/each}
      </select>
    </label>
  </div>

  {#if rows.length === 0}
    <p style="color:#888;padding:2rem;text-align:center;background:#fafafa;border-radius:6px">
      Inga data uppfyller anonymiseringsgränsen ännu (minst 3 vingårdar per kombination).
    </p>
  {:else}
    <div style="overflow-x:auto">
      <table style="width:100%;border-collapse:collapse;font-size:0.9rem">
        <thead>
          <tr style="border-bottom:2px solid #ddd;text-align:left;color:#555;font-size:0.8rem">
            <th style="padding:0.6rem 0.5rem">Sort</th>
            <th style="padding:0.6rem 0.5rem">Län</th>
            <th style="padding:0.6rem 0.5rem">År</th>
            <th style="padding:0.6rem 0.5rem">Snitt kg/ha</th>
            <th style="padding:0.6rem 0.5rem">Min</th>
            <th style="padding:0.6rem 0.5rem">Max</th>
            <th style="padding:0.6rem 0.5rem">Vingårdar</th>
            <th style="padding:0.6rem 0.5rem">Din avkastning</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as r}
            <tr class:user-row={r.user_yield_kg_ha != null}>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.variety_name}</td>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.county}</td>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.harvest_year}</td>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.avg_yield_kg_ha}</td>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.min_yield_kg_ha}</td>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.max_yield_kg_ha}</td>
              <td style="padding:0.5rem;border-bottom:1px solid #f0f0f0">{r.vineyard_count}</td>
              <td class:highlight={r.user_yield_kg_ha != null} style="padding:0.5rem;border-bottom:1px solid #f0f0f0">
                {r.user_yield_kg_ha ?? '—'}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</main>

<style>
  :global(.highlight) { color: #2d6a2d; font-weight: 600; }
  :global(.user-row) { background-color: #f1f8e9; }
</style>
