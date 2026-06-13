<!-- src/routes/login/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  interface Form {
    error?: string;
    sent?: boolean;
    inviteToken?: string | null;
    email?: string;
  }
  export let form: Form;
</script>

<svelte:head><title>Logga in — Svenskt Vin</title></svelte:head>

<main style="max-width:400px;margin:10vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="font-size:1.5rem;margin-bottom:0.5rem">Svenskt Vin</h1>

  {#if form?.sent}
    <p style="background:#e8f5e9;padding:1rem;border-radius:4px">
      Om ett konto finns för den adressen har du fått en inloggningslänk via e-post.
    </p>
  {:else if form?.error}
    <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
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
