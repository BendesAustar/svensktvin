<!-- src/routes/login/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { PageData } from './$types';

  interface Form {
    error?: string;
    sent?: boolean;
    inviteToken?: string | null;
    email?: string;
    membershipSent?: boolean;
    showPassword?: boolean;
    password?: string;
    membershipEmail?: string;
  }

  export let form: Form;
  export let data: PageData;

  function togglePassword() {
    form.showPassword = !form.showPassword;
  }
</script>

<svelte:head><title>Logga in — Svenskt Vin</title></svelte:head>

<main style="max-width:400px;margin:10vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="font-size:1.5rem;margin-bottom:0.5rem">Svenskt Vin</h1>

  <!-- ─── Success: magic link sent ─── -->
  {#if form?.sent}
    <p style="background:#e8f5e9;padding:1rem;border-radius:4px;margin-bottom:1rem">
      Om ett konto finns för den adressen har du fått en inloggningslänk via e-post.
    </p>
    <p style="font-size:0.9rem;color:#555">
      <a href="/" style="color:#2d6a2d">← Tillbaka</a>
    </p>

  <!-- ─── Success: membership request sent ─── -->
  {:else if form?.membershipSent}
    <p style="background:#e8f5e9;padding:1rem;border-radius:4px;margin-bottom:1rem">
      Tack! Vi skickar en inbjudningslänk så snart vi har godkänt din förfrågan.
    </p>
    <p style="font-size:0.9rem;color:#555">
      <a href="/" style="color:#2d6a2d">← Tillbaka</a>
    </p>

  <!-- ─── Error state ─── -->
  {:else if form?.error}
    <p style="color:#c62828;margin-bottom:0.5rem">{form?.error}</p>

    <!-- Show vineyard context if invite token present -->
    {#if data.vineyard}
      <div style="background:#fff3e0;padding:0.75rem;border-radius:4px;margin-bottom:1rem;font-size:0.85rem;color:#555">
        Du har blivit inbjuden till <strong>{data.vineyard.name}</strong>.
      </div>
    {/if}

    <!-- Password login form (error state) -->
    <form method="POST" use:enhance>
      <label for="email-input" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">E-postadress</label>
      <input
        id="email-input"
        type="email"
        name="email"
        required
        value="{form?.email ?? ''}"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />

      <div style="margin-top:0.75rem;position:relative">
        <label for="password-input" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Lösenord</label>
        <input
          id="password-input"
          type="{form?.showPassword ? 'text' : 'password'}"
          name="password"
          value="{form?.password ?? ''}"
          style="width:100%;padding:0.6rem 2.5rem 0.6rem 0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
        />
        <button
          type="button"
          on:click={togglePassword}
          style="position:absolute;right:0.5rem;top:50%;transform:translateY(-50%);background:none;border:none;cursor:pointer;font-size:0.85rem;color:#666"
        >
          {form?.showPassword ? 'Dölj' : 'Visa'}
        </button>
      </div>

      <button
        type="submit"
        name="action"
        value="login_password"
        style="width:100%;margin-top:0.75rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Logga in
      </button>
    </form>

    <p style="margin:1rem 0 0.5rem;text-align:center">
      <a href="/auth/forgot-password" style="color:#2d6a2d;font-size:0.9rem">Glömt lösenord?</a>
    </p>

  <!-- ─── Default: email-only magic link form ─── -->
  {:else}
    <!-- Show vineyard invitation context if invite token present -->
    {#if data.vineyard}
      <div style="background:#e3f2fd;padding:1rem;border-radius:4px;margin-bottom:1rem;border-left:4px solid #1976d2">
        <p style="margin:0 0 0.25rem;font-size:0.85rem;color:#555">Du har blivit inbjuden att gå med i</p>
        <p style="margin:0;font-size:1.1rem;font-weight:600">{data.vineyard.name}</p>
      </div>
    {/if}

    <!-- Password login form (default) -->
    <form method="POST" use:enhance>
      <label for="email-input" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">E-postadress</label>
      <input
        id="email-input"
        type="email"
        name="email"
        required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
      />

      <div style="margin-top:0.75rem;position:relative">
        <label for="password-input" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Lösenord</label>
        <input
          id="password-input"
          type="password"
          name="password"
          style="width:100%;padding:0.6rem 2.5rem 0.6rem 0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
        />
        <button
          type="button"
          on:click={togglePassword}
          style="position:absolute;right:0.5rem;top:50%;transform:translateY(-50%);background:none;border:none;cursor:pointer;font-size:0.85rem;color:#666"
        >
          Visa
        </button>
      </div>

      <button
        type="submit"
        name="action"
        value="login_password"
        style="width:100%;margin-top:0.75rem;padding:0.7rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer"
      >
        Logga in
      </button>
    </form>

    <p style="margin:1rem 0 0.5rem;text-align:center">
      <a href="/auth/forgot-password" style="color:#2d6a2d;font-size:0.9rem">Glömt lösenord?</a>
    </p>

    <div style="border-top:1px solid #ddd;margin-top:1.5rem;padding-top:1.5rem">
      <p style="color:#555;margin:0 0 0.5rem;font-size:0.9rem;text-align:center">Har du inget konto?</p>

      {#if data.inviteToken}
        <!-- Already have an invite — go straight to register -->
        <a
          href="/register?invite={data.inviteToken}"
          style="display:block;text-align:center;padding:0.7rem;background:#fff;color:#2d6a2d;border:2px solid #2d6a2d;border-radius:4px;font-size:1rem;cursor:pointer;text-decoration:none"
        >
          Skapa konto med inbjudan
        </a>
      {:else}
        <!-- Two options: create account OR request membership -->
        <button
          type="button"
          on:click={() => { form.showPassword = false; form.membershipSent = false; }}
          style="display:block;text-align:center;padding:0.7rem;width:100%;background:#fff;color:#2d6a2d;border:2px solid #2d6a2d;border-radius:4px;font-size:1rem;cursor:pointer;text-decoration:none;margin-bottom:0.5rem"
        >
          Skapa konto
        </button>

        <p style="margin:0 0 0.75rem;font-size:0.8rem;color:#666;text-align:center">
          Eller om du behöver bli inbjuden först:
        </p>
        <a
          href="#membership"
          style="display:block;text-align:center;padding:0.6rem;width:100%;background:#f5f5f5;color:#555;border:1px solid #ddd;border-radius:4px;font-size:0.9rem;cursor:pointer;text-decoration:none"
        >
          Begär medlemskap
        </a>
      {/if}
    </div>

    <!-- Membership request form (inline toggle) -->
    {#if form?.showMembershipForm}
      <div id="membership" style="margin-top:1rem;padding-top:1rem;border-top:1px solid #ddd">
        <p style="font-size:0.9rem;color:#555;margin-bottom:0.75rem">
          Skicka en förfrågan till oss. Vi godkänner och skickar en inbjudningslänk via e-post.
        </p>
        <form method="POST" use:enhance>
          <label for="membership-email" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">E-postadress</label>
          <input
            id="membership-email"
            type="email"
            name="email"
            required
            value="{form?.membershipEmail ?? ''}"
            style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
            placeholder="din@exempel.se"
          />

          <label for="membership-name" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Namn</label>
          <input
            id="membership-name"
            type="text"
            name="name"
            required
            style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box"
            placeholder="Ditt namn"
          />

          <button
            type="submit"
            name="action"
            value="request_membership"
            style="width:100%;margin-top:1rem;padding:0.7rem;background:#f0f0f0;color:#333;border:1px solid #ccc;border-radius:4px;font-size:1rem;cursor:pointer"
          >
            Skicka förfrågan
          </button>
        </form>
      </div>
    {/if}
  {/if}
</main>
