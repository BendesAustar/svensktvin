<!-- src/routes/auth/forgot-password/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData } from './$types';

  interface Form {
    error?: string;
    sent?: boolean;
    email?: string;
  }

  export let form: Form;
</script>

<svelte:head><title>Glömt lösenord — Svenskt Vin</title></svelte:head>

<main style="max-width:400px;margin:10vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="font-size:1.5rem;margin-bottom:0.5rem">Svenskt Vin</h1>

  {#if form?.sent}
    <p style="background:#e8f5e9;padding:1rem;border-radius:4px;margin-bottom:1rem">
      Om ett konto finns för den adressen har du fått ett mejl med instruktioner för att återställa ditt lösenord.
    </p>
    <p style="margin:1rem 0 0;text-align:center">
      <a href="/login" style="color:#2d6a2d;font-size:0.9rem">← Tillbaka till inloggning</a>
    </p>
  {:else}
    {#if form?.error}
      <p style="color:#c62828;margin-bottom:0.5rem">{form.error}</p>
    {/if}

    <p style="color:#555;margin-bottom:1.5rem;font-size:0.95rem">
      Ange din e-postadress så skickar vi en länk för att återställa ditt lösenord.
    </p>

    <form method="POST" use:enhance>
      <label for="email" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">E-postadress</label>
      <input
        id="email"
        type="email"
        name="email"
        required
        value="{form?.email ?? ''}"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />
      <button
        type="submit"
        style="width:100%;margin-top:0.75rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Skicka återställningslänk
      </button>
    </form>

    <p style="margin:1.5rem 0 0;text-align:center">
      <a href="/login" style="color:#2d6a2d;font-size:0.9rem">← Tillbaka till inloggning</a>
    </p>
  {/if}
</main>
