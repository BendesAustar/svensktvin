<!-- src/routes/vineyard/[id]/benchmark/+page.svelte -->
<script lang="ts">
  import type { PageData } from './$types';
  export let data: PageData;

  const { vineyard, userYields, regionalBenchmarks, timeline } = data;
</script>

<svelte:head><title>Benchmark — {vineyard.name} — Svenskt Vin</title></svelte:head>

<main style="max-width:900px;margin:0 auto;padding:1rem;font-family:sans-serif">
  <a href="/vineyard/{vineyard.id}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Benchmark & Historik</h1>

  <!-- Regional benchmarks -->
  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.75rem">Regional benchmark — {vineyard.county}</h2>

    {#if regionalBenchmarks.length === 0}
      <p style="color:#888;padding:1rem;background:#f9f9f9;border-radius:4px">
        {#if userYields.length > 0}
          Det krävs minst 3 aktiva vingårdar i regionen för att visa benchmark. Fyll i fler skörder!
        {:else}
          Det krävs minst 3 aktiva vingårdar i din region för att visa benchmark.
        {/if}
      </p>
    {:else}
      <table style="width:100%;border-collapse:collapse;font-size:0.9rem">
        <thead>
          <tr style="border-bottom:2px solid #eee;text-align:left">
            <th style="padding:0.5rem">Sort</th>
            <th style="padding:0.5rem">År</th>
            <th style="padding:0.5rem">Regional medel (kg/ha)</th>
            <th style="padding:0.5rem">Medel Brix</th>
            <th style="padding:0.5rem">Vingårdar</th>
          </tr>
        </thead>
        <tbody>
          {#each regionalBenchmarks as bench}
            <tr style="border-bottom:1px solid #f0f0f0">
              <td style="padding:0.5rem">{bench.variety_name}</td>
              <td style="padding:0.5rem">{bench.harvest_year}</td>
              <td style="padding:0.5rem">
                {Number(bench.county_avg_kg_ha)}
                {#if Number(bench.county_avg_kg_ha) >= 1500}
                  <span style="color:#2d6a2d;font-size:0.8rem"> ✓</span>
                {:else}
                  <span style="color:#888;font-size:0.8rem"></span>
                {/if}
              </td>
              <td style="padding:0.5rem">{bench.county_avg_brix}</td>
              <td style="padding:0.5rem;color:#888">{bench.vineyard_count}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>

  <!-- User yields -->
  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.75rem">Dina skörder</h2>
    {#if userYields.length === 0}
      <p style="color:#888;padding:1rem;background:#f9f9f9;border-radius:4px">Inga skörder registrerade ännu.</p>
    {:else}
      <table style="width:100%;border-collapse:collapse;font-size:0.9rem">
        <thead>
          <tr style="border-bottom:2px solid #eee;text-align:left">
            <th style="padding:0.5rem">Sort</th>
            <th style="padding:0.5rem">År</th>
            <th style="padding:0.5rem">Medel kg/ha</th>
            <th style="padding:0.5rem">Medel Brix</th>
            <th style="padding:0.5rem">Skörder</th>
          </tr>
        </thead>
        <tbody>
          {#each userYields as y}
            <tr style="border-bottom:1px solid #f0f0f0">
              <td style="padding:0.5rem">{y.variety_name}</td>
              <td style="padding:0.5rem">{y.harvest_year}</td>
              <td style="padding:0.5rem">{y.avg_yield_kg_ha}</td>
              <td style="padding:0.5rem">{y.avg_brix}</td>
              <td style="padding:0.5rem;color:#888">{y.harvest_count}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>

  <!-- Harvest timeline -->
  <section>
    <h2 style="font-size:1.1rem;margin-bottom:0.75rem">Skördhistorik</h2>
    {#if timeline.length === 0}
      <p style="color:#888;padding:1rem;background:#f9f9f9;border-radius:4px">Inga skörder registrerade.</p>
    {:else}
      <table style="width:100%;border-collapse:collapse;font-size:0.9rem">
        <thead>
          <tr style="border-bottom:2px solid #eee;text-align:left">
            <th style="padding:0.5rem">År</th>
            <th style="padding:0.5rem">Datum</th>
            <th style="padding:0.5rem">Block</th>
            <th style="padding:0.5rem">Sort</th>
            <th style="padding:0.5rem">Vikt (kg)</th>
            <th style="padding:0.5rem">Brix</th>
            <th style="padding:0.5rem">Hälsa</th>
          </tr>
        </thead>
        <tbody>
          {#each timeline as t}
            <tr style="border-bottom:1px solid #f0f0f0">
              <td style="padding:0.5rem">{t.harvest_year}</td>
              <td style="padding:0.5rem">{t.harvest_date}</td>
              <td style="padding:0.5rem">{t.block_name}</td>
              <td style="padding:0.5rem">{t.variety_name}</td>
              <td style="padding:0.5rem">{t.yield_kg}</td>
              <td style="padding:0.5rem">{t.brix}</td>
              <td style="padding:0.5rem">
                {#if t.vine_health_rating}
                  {t.vine_health_rating}/5
                {:else}
                  <span style="color:#ccc">—</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>
</main>
