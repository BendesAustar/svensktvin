<!-- src/routes/register/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { ActionData, PageData } from './$types';

  export let data: PageData;
  export let form: ActionData;
</script>

<svelte:head>
  <title>Registrera dig — Svenskt Vin</title>
</svelte:head>

<main
  style="max-width:420px;margin:15vh auto;padding:0 1rem;font-family:sans-serif"
>
  {#if data.error}
    <!-- Invalid/expired invite -->
    <div style="background:#ffebee;padding:1rem;border-radius:4px;margin-bottom:1rem">
      <p style="margin:0;color:#c62828;font-size:0.95rem">❌ {data.error}</p>
    </div>
    <a href="/login" style="color:#2d6a2d;font-size:0.9rem">← Tillbaka till inloggning</a>
  {:else if data.hasAccount}
    <!-- User has existing account -->
    <h1 style="margin-bottom:1rem">Redan registrerad</h1>
    <div
      style="background:#fff3e0;padding:1rem;border-radius:4px;margin-bottom:1.5rem"
    >
      <p style="margin:0 0 0.5rem;font-size:0.95rem">
        Det finns redan ett konto för <strong>{data.email}</strong>.
      </p>
      <p style="margin:0;font-size:0.85rem;color:#555">
        Logga in med det kontot för att acceptera inbjudan till{' '}
        {data.invite?.vineyard.name}.
      </p>
    </div>
    <a
      href="/login"
      style="display:block;text-align:center;padding:0.7rem;background:#2d6a2d;color:#fff;border-radius:4px;text-decoration:none"
    >
      Logga in
    </a>
  {:else}
    <!-- New account registration -->
    <h1 style="margin-bottom:0.5rem">Skapa konto</h1>
    <p style="color:#555;margin-bottom:1.5rem;font-size:0.9rem">
      Du har blivit inbjuden att joina{' '}
      <strong>{data.invite?.vineyard.name}</strong> (roll:{' '}
      {data.invite?.role === 'editor' ? 'Redaktör' : 'Ägare'}).
    </p>

    <form method="POST" use:enhance>
      <input type="hidden" name="action" value="register" />
      <input type="hidden" name="invite_token" value={data.inviteToken} />

      <label
        for="email-input"
        style="display:block;margin-bottom:0.25rem;font-size:0.9rem"
        >E-postadress</label
      >
      <input
        id="email-input"
        type="email"
        name="email"
        required
        value={data.email}
        readonly
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box;background:#f5f5f5"
      />
      <p style="margin:0.25rem 0 1rem;font-size:0.75rem;color:#888">
        Denna e-postadress kommer att bli ditt användarkonto.
      </p>

      <label
        for="name-input"
        style="display:block;margin-bottom:0.25rem;font-size:0.9rem"
        >Namn</label
      >
      <input
        id="name-input"
        type="text"
        name="name"
        required
        minlength="2"
        placeholder="Ditt namn"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />

      {#if form?.error}
        <p
          style="color:#c62828;margin:0.75rem 0 0.5rem;font-size:0.85rem"
        >
          {form.error}
        </p>
      {/if}

      <button
        type="submit"
        style="width:100%;margin-top:1rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Skapa konto och joina
      </button>
    </form>
  {/if}
</main>
