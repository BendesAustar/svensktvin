<!-- src/routes/invite/confirm/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData } from './$types';

  interface Form {
    error?: string;
  }

  export let data: PageData;
  export let form: Form;
</script>

<svelte:head>
  <title>Acceptera inbjudan — Svenskt Vin</title>
</svelte:head>

<main style="max-width:450px;margin:15vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="font-size:1.5rem;margin-bottom:0.5rem">Acceptera inbjudan</h1>

  {#if form?.error}
    <p style="background:#ffebee;padding:1rem;border-radius:4px;margin-bottom:1rem;color:#c62828">
      ❌ {form.error}
    </p>
  {/if}

  <div style="background:#e3f2fd;padding:1rem;border-radius:4px;margin-bottom:1.5rem;border-left:4px solid #1976d2">
    <p style="margin:0 0 0.5rem;font-size:0.95rem">
      Du är inloggad som <strong>{data.currentEmail}</strong>, men inbjudan är adresserad till <strong>{data.inviteEmail}</strong>.
    </p>
    <p style="margin:0 0 0.5rem;font-size:0.9rem;color:#555">
      Vill du acceptera inbjudan och gå med i <strong>{data.vineyardName}</strong> som <strong>{data.role === 'editor' ? 'Redaktör' : 'Ägare'}</strong>?
    </p>
  </div>

  <form method="POST" use:enhance>
    <button
      type="submit"
      style="width:100%;padding:0.8rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
    >
      Ja, acceptera inbjudan
    </button>
  </form>

  <p style="margin-top:1rem;text-align:center">
    <a href="/login" style="color:#555;font-size:0.9rem">← Avbryt</a>
  </p>
</main>
