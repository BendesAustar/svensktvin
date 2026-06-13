<!-- src/routes/vineyard/[id]/settings/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { ActionData, PageData } from './$types';
  export let form: ActionData;
  export let data: PageData;

  const { vineyard, ownerCount, members } = data;
</script>

<svelte:head><title>Inställningar: {vineyard.name} — Svenskt Vin</title></svelte:head>

<main style="max-width:700px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/vineyard/{vineyard.id}" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Inställningar</h1>

  <!-- ════════════════ Vineyard Settings Form ════════════════ -->
  <form method="POST" use:enhance>
    <input type="hidden" name="action" value="update_vineyard" />

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Vingårdsuppgifter</legend>

      <label for="name" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Namn <span style="color:#c62828">*</span></label>
      <input id="name" type="text" name="name" required value={vineyard.name}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="county" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Län <span style="color:#c62828">*</span></label>
      <select id="county" name="county" required
        value={vineyard.county ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj län</option>
        <option>Skåne</option><option>Blekinge</option><option>Gotland</option>
        <option>Kalmar</option><option>Kronoberg</option><option>Östergötland</option>
        <option>Västergötland</option><option>Stockholm</option><option>Uppsala</option>
        <option>Västmanland</option><option>Södermanland</option><option>Halland</option>
        <option>Gävleborg</option><option>Värmland</option><option>Dalarna</option>
        <option>Örebro</option>
      </select>

      <label for="municipality" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Kommun</label>
      <input id="municipality" type="text" name="municipality" value={vineyard.municipality}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="established_year" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Uppstartsår</label>
      <input id="established_year" type="number" name="established_year" min="1800" max="2030" value={vineyard.established_year ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="total_area_ha" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Total area (ha)</label>
      <input id="total_area_ha" type="number" name="total_area_ha" step="0.01" min="0" value={vineyard.total_area_ha ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="legal_id_type" style="display:block;margin-top:1rem;margin-bottom:0.25rem;font-size:0.9rem">Rättighetstyp</label>
      <select id="legal_id_type" name="legal_id_type"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj</option>
        <option value="enskild" selected={vineyard.legal_id_type === 'enskild'}>enskild</option>
        <option value="ab" selected={vineyard.legal_id_type === 'ab'}>ab</option>
        <option value="handelsbolag" selected={vineyard.legal_id_type === 'handelsbolag'}>handelsbolag</option>
        <option value="swealagsbolag" selected={vineyard.legal_id_type === 'swealagsbolag'}>swealagsbolag</option>
        <option value="stiftelse" selected={vineyard.legal_id_type === 'stiftelse'}>stiftelse</option>
        <option value="kommun" selected={vineyard.legal_id_type === 'kommun'}>kommun</option>
        <option value="other" selected={vineyard.legal_id_type === 'other'}>other</option>
      </select>

      <label for="legal_id" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Organisationsnummer / Reg.nr</label>
      <input id="legal_id" type="text" name="legal_id" value={vineyard.legal_id ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="legal_name" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Företagsnamn</label>
      <input id="legal_name" type="text" name="legal_name" value={vineyard.legal_name ?? ''}
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Produktionsmetod</legend>
      <label style="display:flex;align-items:center;margin-bottom:0.5rem;cursor:pointer">
        <input type="checkbox" name="organic" value="on" checked={vineyard.organic} style="margin-right:0.5rem;font-size:1.2rem" />
        Ekologisk
      </label>
      <label style="display:flex;align-items:center;cursor:pointer">
        <input type="checkbox" name="biodynamic" value="on" checked={vineyard.biodynamic} style="margin-right:0.5rem;font-size:1.2rem" />
        Biodynamisk
      </label>
    </fieldset>

    <button type="submit"
      style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
      Spara ändringar
    </button>
  </form>

  <!-- ════════════════ Member Management ════════════════ -->
  <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
    <legend style="font-weight:600;padding:0 0.5rem">Medlemmar</legend>

    <table style="width:100%;border-collapse:collapse;margin-bottom:1rem">
      <thead>
        <tr style="border-bottom:2px solid #ddd;text-align:left;font-size:0.85rem;color:#555">
          <th style="padding:0.5rem">Namn</th>
          <th style="padding:0.5rem">E-post</th>
          <th style="padding:0.5rem">Roll</th>
          <th style="padding:0.5rem"></th>
        </tr>
      </thead>
      <tbody>
        {#each members as m (m.id)}
          <tr style="border-bottom:1px solid #f0f0f0">
            <td style="padding:0.5rem"><strong>{m.name}</strong></td>
            <td style="padding:0.5rem">{m.email}</td>
            <td style="padding:0.5rem">{m.role}</td>
            <td style="padding:0.5rem">
              {#if m.role === 'owner' && ownerCount <= 1}
                <!-- Last owner — disabled state with explanation -->
                <button type="button" disabled
                  style="padding:0.3rem 0.6rem;background:#fdd;border:1px solid #ef9;border-radius:3px;font-size:0.8rem;color:#8d2;font-weight:500;cursor:not-allowed;white-space:nowrap">
                  ⚠️ Sist ägare
                </button>
              {:else}
                <!-- Separate form for delete (no nested forms allowed) -->
                <form method="POST" use:enhance>
                  <input type="hidden" name="action" value="remove_member" />
                  <input type="hidden" name="user_id" value={m.id} />
                  <button type="submit"
                    style="padding:0.3rem 0.6rem;background:#ef5350;color:#fff;border:none;border-radius:3px;font-size:0.8rem;cursor:pointer">
                    Ta bort
                  </button>
                </form>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>

    <form method="POST" use:enhance style="display:flex;gap:0.5rem;align-items:flex-end;flex-wrap:wrap">
      <input type="hidden" name="action" value="invite_member" />
      <div style="flex:1;min-width:180px">
        <label for="invite-email" style="display:block;margin-bottom:0.25rem;font-size:0.85rem">E-postadress</label>
        <input id="invite-email" type="email" name="email" required placeholder="namn@exempel.se"
          style="width:100%;padding:0.5rem;border:1px solid #ccc;border-radius:4px;font-size:0.9rem" />
      </div>
      <div style="min-width:120px">
        <label for="invite-role" style="display:block;margin-bottom:0.25rem;font-size:0.85rem">Roll</label>
        <select id="invite-role" name="role"
          style="width:100%;padding:0.5rem;border:1px solid #ccc;border-radius:4px;font-size:0.9rem">
          <option value="editor">Redaktör</option>
          <option value="owner">Ägare</option>
        </select>
      </div>
      <button type="submit"
        style="padding:0.5rem 1rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:0.9rem;cursor:pointer;white-space:nowrap">Bjud in</button>
    </form>
    <p style="margin:0.75rem 0 0;font-size:0.8rem;color:#666">
      En inbjudan skickas per e-post. Personen kan acceptera efter att ha loggat in eller skapat ett konto.
    </p>

    {#if form?.success}
      <div style="background:#e8f5e9;padding:0.75rem 1rem;border-radius:4px;margin-top:1rem">
        <p style="margin:0;color:#2d6a2d;font-size:0.9rem">✅ Inbjudan skickad!</p>
      </div>
    {/if}
  </fieldset>
</main>
