<!-- src/routes/auth/set-password/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData } from './$types';

  interface Form {
    error?: string;
    success?: boolean;
    email?: string;
    password?: string;
    confirmPassword?: string;
    passwordErrors?: string[];
    token?: string;
  }

  export let form: Form;
  export let data: PageData;

  function validatePasswords(form: Form): string[] {
    const errors: string[] = [];
    if (form.password !== form.confirmPassword) {
      errors.push('Lösenorden matchar inte.');
    }
    return errors;
  }
</script>

<svelte:head><title>Ställ in lösenord — Svenskt Vin</title></svelte:head>

<main style="max-width:400px;margin:10vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="font-size:1.5rem;margin-bottom:0.5rem">Svenskt Vin</h1>

  {#if form?.success}
    <p style="background:#e8f5e9;padding:1rem;border-radius:4px;margin-bottom:1rem">
      Ditt lösenord har ställts in. Du kan nu logga in med din e-postadress och lösenord.
    </p>
    <p style="margin:1rem 0 0;text-align:center">
      <a href="/login" style="color:#2d6a2d;font-size:0.9rem">Gå till inloggning →</a>
    </p>
  {:else}
    {#if form?.error}
      <p style="color:#c62828;margin-bottom:0.5rem">{form?.error}</p>
    {/if}

    {#if form?.passwordErrors && form.passwordErrors.length > 0}
      <ul style="color:#c62828;margin-bottom:1rem;font-size:0.9rem;padding-left:1.2rem">
        {#each form.passwordErrors as err}
          <li>{err}</li>
        {/each}
      </ul>
    {/if}

    <p style="color:#555;margin-bottom:1.5rem;font-size:0.95rem">
      Ställ in ditt lösenord för att logga in med e-post och lösenord.
      {#if data.email}
        <br>
        <strong>{data.email}</strong>
      {/if}
    </p>

    <form method="POST" use:enhance>
      {#if form?.token}
        <input type="hidden" name="token" value="{form?.token ?? ''}" />
      {/if}
      {#if data.email}
        <input type="hidden" name="email" value="{data.email}" />
      {/if}

      <label for="password" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Nytt lösenord</label>
      <input
        id="password"
        type="password"
        name="password"
        required
        value="{form?.password ?? ''}"
        minlength="8"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />

      <label for="confirmPassword" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Bekräfta lösenord</label>
      <input
        id="confirmPassword"
        type="password"
        name="confirmPassword"
        required
        value="{form?.confirmPassword ?? ''}"
        minlength="8"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />

      <button
        type="submit"
        style="width:100%;margin-top:1rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Ställ in lösenord
      </button>
    </form>

    <p style="margin:1.5rem 0 0;font-size:0.85rem;color:#666">
      Lösenordet måste vara minst 8 tecken, innehålla en stor bokstav, en liten bokstav och en siffra.
    </p>
  {/if}
</main>
