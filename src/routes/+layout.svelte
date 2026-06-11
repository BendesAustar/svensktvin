<!-- src/routes/+layout.svelte -->
<script lang="ts">
  import type { LayoutData } from './$types';
  export let data: LayoutData;
</script>

{#if data.user}
  <header style="padding:0.5rem 1rem;background:#f5f5f5;display:flex;justify-content:space-between;align-items:center;font-family:sans-serif">
    <span style="font-weight:600">Svenskt Vin</span>
    <form method="POST" action="/logout">
      <button type="submit" style="background:none;border:none;cursor:pointer;color:#555">
        Logga ut ({data.user.email})
      </button>
    </form>
  </header>
{/if}

<slot />
