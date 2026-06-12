<script lang="ts">
  import { page } from '$app/stores';
</script>

<svelte:head><title>Fel — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:15vh auto;padding:0 1rem;font-family:sans-serif;text-align:center">
  <h1 style="color:#c62828">{#if $page.status}{$$props.error?.status ?? $page.status}{:else}Fel{/if}</h1>
  <p style="color:#555;margin:1rem 0">
    {#if $page.status === 404}
      Sidan hittades inte.
    {:else}
      Något gick fel. Försök igen senare.
    {/if}
  </p>
  <a href="/" style="display:inline-block;padding:0.6rem 1.5rem;background:#2d6a2d;color:#fff;border-radius:4px;text-decoration:none;font-size:1rem">
    Till startsidan
  </a>
</main>
