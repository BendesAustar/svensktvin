<!-- src/routes/login/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData } from './$types';

  interface Form {
    error?: string;
    sent?: boolean;
    inviteToken?: string | null;
    email?: string;
  }

  export let form: Form;
  export let data: PageData;
</script>

<svelte:head><title>Logga in — Svenskt Vin</title></svelte:head>

<main style="max-width:400px;margin:10vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="font-size:1.5rem;margin-bottom:0.5rem">Svenskt Vin</h1>

  {#if form?.sent}
    <p style="background:#e8f5e9;padding:1rem;border-radius:4px">
      Om ett konto finns för den adressen har du fått en inloggningslänk via e-post.
    </p>
  {:else if form?.error}
    <p style="color:#c62828;margin-bottom:0.5rem">{form.error}</p>
    <!-- Show vineyard context if invite token present -->
    {#if data.vineyard}
      <div style="background:#fff3e0;padding:0.75rem;border-radius:4px;margin-bottom:1rem;font-size:0.85rem;color:#555">
        Du har blivit inbjuden till <strong>{data.vineyard.name}</strong>.
      </div>
    {/if}
    <form method="POST" use:enhance>
      <label for="email-input" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">E-postadress</label>
      <input
        id="email-input"
        type="email"
        name="email"
        required
        value="{form.email ?? ''}"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />
      <button
        type="submit"
        style="width:100%;margin-top:0.75rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Skicka inloggningslänk
      </button>
    </form>
  {:else}
    <!-- Show vineyard invitation context if invite token is present -->
    {#if data.vineyard}
      <div style="background:#e3f2fd;padding:1rem;border-radius:4px;margin-bottom:1rem;border-left:4px solid #1976d2">
        <p style="margin:0 0 0.25rem;font-size:0.85rem;color:#555">Du har blivit inbjuden att gå med i</p>
        <p style="margin:0;font-size:1.1rem;font-weight:600">{data.vineyard.name}</p>
      </div>
    {/if}
    <p style="color:#555;margin-bottom:1.5rem">Ange din e-postadress för att logga in.</p>
    <form method="POST" use:enhance>
      <label for="email-input" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">E-postadress</label>
      <input
        id="email-input"
        type="email"
        name="email"
        required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />
      <button
        type="submit"
        style="width:100%;margin-top:0.75rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Skicka inloggningslänk
      </button>
    </form>
  {/if}
</main>
