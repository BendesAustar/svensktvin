<!-- src/routes/+layout.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import type { LayoutData } from './$types';
  export let data: LayoutData;

  let showCookieNotice = false;

  onMount(() => {
    if (!document.cookie.includes('cookie_consent=')) {
      showCookieNotice = true;
    }
  });

  function acceptCookies() {
    document.cookie = 'cookie_consent=accepted;path=/;max-age=31536000';
    showCookieNotice = false;
  }
</script>

{#if data.user}
  <header style="padding:0.5rem 1rem;background:#f5f5f5;border-bottom:1px solid #ddd;display:flex;justify-content:space-between;align-items:center;font-family:sans-serif">
    <a href="/" style="text-decoration:none;font-weight:600;font-size:1.05rem;color:#2d6a2d">🍷 Svenskt Vin</a>
    <form method="POST" action="/logout">
      <button type="submit" style="background:none;border:none;cursor:pointer;color:#555;font-size:0.85rem">
        Logga ut
      </button>
    </form>
  </header>
{/if}

<slot />

{#if showCookieNotice}
  <div style="position:fixed;bottom:0;left:0;right:0;background:#1a1a1a;color:#fff;padding:1rem 1.5rem;z-index:1000;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:0.75rem;font-family:sans-serif;font-size:0.9rem">
    <div style="flex:1;min-width:200px">
      <strong style="margin-right:0.5rem">Cookies</strong>
      Svenskt Vin använder endast nödvändiga sessionscookies för inloggning.
      Ingen spårning eller tredjepartscookie.
      <a href="/privacy" style="color:#8fbc8f;margin-left:0.5rem;text-decoration:underline">Läs mer</a>.
    </div>
    <div style="display:flex;gap:0.5rem">
      <button on:click={acceptCookies}
        style="padding:0.5rem 1.2rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:0.9rem;cursor:pointer;font-weight:600">
        Acceptera
      </button>
    </div>
  </div>
{/if}
