<!-- src/routes/onboard/+page.svelte -->
<script lang="ts">
  import { enhance } from '$app/forms';
  import type { ActionData } from './$types';
  export let form: ActionData;

  let geolocationError: string | null = null;

  function requestLocation() {
    geolocationError = null;
    if (!navigator.geolocation) {
      geolocationError = 'Geolocation stöds inte i denna webbläsare.';
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        (document.getElementById('lat') as HTMLInputElement).value = pos.coords.latitude.toFixed(6);
        (document.getElementById('lon') as HTMLInputElement).value = pos.coords.longitude.toFixed(6);
      },
      () => {
        geolocationError = 'Kunde inte hämta plats. Ange län och kommun manuellt.';
      }
    );
  }
</script>

<svelte:head><title>Registrera vingård — Svenskt Vin</title></svelte:head>

<main style="max-width:600px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <h1 style="margin-bottom:0.5rem">Registrera din vingård</h1>
  <p style="color:#555;margin-bottom:1.5rem">För att delta i Svenskt Vin behöver vi registrera din vingård. All data delas anonymt i benchmark-grupper.</p>

  {#if geolocationError}
    <p style="background:#fff3cd;padding:0.75rem;border-radius:4px;margin-bottom:1rem;color:#856404">{geolocationError}</p>
  {/if}

  <form method="POST" use:enhance>
    {#if form?.error}
      <p style="color:#c62828;margin-bottom:1rem">{form.error}</p>
    {/if}

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Grundläggande information</legend>

      <label for="name" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Vingårdsnamn <span style="color:#c62828">*</span></label>
      <input id="name" type="text" name="name" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="established_year" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Uppstartsår</label>
      <input id="established_year" type="number" name="established_year" min="1800" max="2030"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="total_area_ha" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Total area (ha)</label>
      <input id="total_area_ha" type="number" name="total_area_ha" step="0.01" min="0"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Plats</legend>

      <button type="button" onclick={requestLocation}
        style="margin-bottom:0.75rem;padding:0.5rem 1rem;background:#e8f5e9;border:1px solid #2d6a2d;color:#2d6a2d;border-radius:4px;cursor:pointer;font-size:0.9rem">
        📍 Hämta GPS-position
      </button>

      <input id="lat" type="hidden" name="lat" />
      <input id="lon" type="hidden" name="lon" />

      <label for="county" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Län <span style="color:#c62828">*</span></label>
      <select id="county" name="county" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box">
        <option value="">Välj län</option>
        <option>Skåne</option>
        <option>Blekinge</option>
        <option>Gotland</option>
        <option>Kalmar</option>
        <option>Kronoberg</option>
        <option>Östergötland</option>
        <option>Västergötland</option>
        <option>Stockholm</option>
        <option>Uppsala</option>
        <option>Västmanland</option>
        <option>Södermanland</option>
        <option>Halland</option>
        <option>Gävleborg</option>
        <option>Värmland</option>
        <option>Dalarna</option>
        <option>Örebro</option>
      </select>

      <label for="municipality" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Kommun <span style="color:#c62828">*</span></label>
      <input id="municipality" type="text" name="municipality" required
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Rättighet</legend>

      <label for="legal_id_type" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Rättighetstyp</label>
      <select id="legal_id_type" name="legal_id_type"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box;margin-bottom:0.75rem">
        <option value="">Välj</option>
        <option>enskild</option>
        <option>ab</option>
        <option>handelsbolag</option>
        <option>other</option>
      </select>

      <label for="legal_id" style="display:block;margin-bottom:0.25rem;font-size:0.9rem">Organisationsnummer / Reg.nr</label>
      <input id="legal_id" type="text" name="legal_id"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />

      <label for="legal_name" style="display:block;margin-top:0.75rem;margin-bottom:0.25rem;font-size:0.9rem">Företagsnamn (valfritt)</label>
      <input id="legal_name" type="text" name="legal_name"
        style="width:100%;padding:0.6rem;border:1px solid #ccc;border-radius:4px;font-size:1rem;box-sizing:border-box" />
    </fieldset>

    <fieldset style="border:1px solid #ddd;padding:1rem;border-radius:4px;margin-bottom:1.5rem">
      <legend style="font-weight:600;padding:0 0.5rem">Produktionsmetod</legend>
      <label style="display:flex;align-items:center;margin-bottom:0.5rem;cursor:pointer">
        <input type="checkbox" name="organic" style="margin-right:0.5rem;font-size:1.2rem" />
        Ekologisk
      </label>
      <label style="display:flex;align-items:center;cursor:pointer">
        <input type="checkbox" name="biodynamic" style="margin-right:0.5rem;font-size:1.2rem" />
        Biodynamisk
      </label>
    </fieldset>

    <button type="submit"
      style="width:100%;padding:0.85rem;background:#2d6a2d;color:#fff;border:none;border-radius:4px;font-size:1rem;cursor:pointer;font-weight:600">
      Registrera vingård
    </button>
  </form>
</main>
