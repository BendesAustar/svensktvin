<!-- src/routes/vineyard/[id]/+layout.svelte -->
<script lang="ts">
  import { page } from '$app/stores';

  // Vineyard ID comes from URL params — always correct, never depends on
  // layout server data flow. This is the single source of truth for nav links.
  $: vineyardId = $page.params.id;
  $: vineyardName = $page.data.vineyard?.name;
  $: role = $page.data.role;
  $: path = $page.url.pathname;
  $: isHome = /^\/vineyard\/\d+$/.test(path);
  $: isHarvest = path.includes('/harvest');
  $: isBenchmark = path.includes('/benchmark');
  $: isSettings = path.includes('/settings');
</script>

{#if vineyardName}
  <nav style="max-width:900px;margin:0 auto;padding:0.5rem 1rem;border-bottom:1px solid #ddd;background:#fff;font-family:sans-serif;display:flex;justify-content:space-between;align-items:center">
    <div style="display:flex;gap:0.25rem;align-items:center">
      <a href="/vineyard/{vineyardId}" style="color:#555;font-size:0.85rem;text-decoration:none;padding:0.4rem 0.6rem;border-radius:4px;{isHome ? 'color:#2d6a2d;background:#e8f5e9;font-weight:500' : 'hover:#f5f5f5'}">
        ← {vineyardName}
      </a>
      <a href="/vineyard/{vineyardId}/harvest/new" style="color:#555;font-size:0.85rem;text-decoration:none;padding:0.4rem 0.6rem;border-radius:4px;{isHarvest ? 'color:#2d6a2d;background:#e8f5e9;font-weight:500' : ''}">
        Skörd
      </a>
      <a href="/vineyard/{vineyardId}/benchmark" style="color:#555;font-size:0.85rem;text-decoration:none;padding:0.4rem 0.6rem;border-radius:4px;{isBenchmark ? 'color:#2d6a2d;background:#e8f5e9;font-weight:500' : ''}">
        📊 Jämförelse
      </a>
    </div>
    {#if role === 'owner'}
      <a href="/vineyard/{vineyardId}/settings" style="color:#555;font-size:0.85rem;text-decoration:none;padding:0.4rem 0.6rem;border-radius:4px;{isSettings ? 'color:#2d6a2d;background:#e8f5e9;font-weight:500' : 'hover:#f5f5f5'}">
        ⚙️
      </a>
    {/if}
  </nav>
{/if}
<slot />
