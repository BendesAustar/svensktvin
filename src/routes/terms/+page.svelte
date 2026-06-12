<!-- src/routes/terms/+page.svelte -->
<svelte:head><title>Användarvillkor — Svenskt Vin</title></svelte:head>

<main style="max-width:700px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Användarvillkor</h1>
  <p style="color:#666;margin-bottom:2rem">Senast uppdaterad: {new Date().toISOString().split('T')[0]}</p>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">1. Tjänsteintresse</h2>
    <p style="line-height:1.6">
      Svenskt Vin är en plattform för registrering och benchmarking av vingårdsdata i Sverige.
      Tjänsten riktar sig till vingårdar, vinodlare och vinintresserade som vill föra register
      över sina odlingar och jämföra data med andra användare (anonymiserat).
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">2. Registrering</h2>
    <p style="line-height:1.6">
      För att använda Svenskt Vin registrerar du dig med en e-postadress.
      Du bekräftar din inloggning via en magic link som skickas till din e-post.
      Du ansvarar för att hålla din e-postadress aktuell och för att säkerställa att
      endast behöriga personer har tillgång till ditt konto.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">3. Användarroller</h2>
    <p style="line-height:1.6">
      <strong>Ägare</strong> — har full behörighet över vingården, inklusive att redigera
      data, bjuda in nya medlemmar och radera vingården.<br>
      <strong>Redaktör</strong> — kan redigera vingårdsdata men inte hantera medlemmar
      eller radera vingården.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">4. Upphovsrätt och innehåll</h2>
    <p style="line-height:1.6">
      Du behåller äganderätten till din data. Genom att ladda upp data till Svenskt Vin
      beviljar du oss en begränsad licens att lagra, bearbeta och visa din data enbart
      för tjänstens syfte. Du garanterar att du har rätt att dela den data du laddar upp.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">5. Tjänstetillgänglighet</h2>
    <p style="line-height:1.6">
      Svenskt Vin tillhandahålls "i befintligt skick". Vi strävar efter hög tillgänglighet
      men garanterar inte obegränsad drifttid. Underhållsfönster kan ske med kort varsel.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">6. Ansvarsbegränsning</h2>
    <p style="line-height:1.6">
      Svenskt Vin ansvarar inte för direkta eller indirekta skador som uppstår till följd
      av användning av tjänsten, inklusive men inte begränsat till felaktiga data,
      bortfall av data eller förlorad inkomst.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">7. Kontorsradering</h2>
    <p style="line-height:1.6">
      Du kan när som helst radera ditt konto via kontots inställningar. Vid radering
      raderas alla tillhörande data permanent, förutom skörderecord som behålls
      anonymiserat för benchmarking.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">8. Tillämplig lag</h2>
    <p style="line-height:1.6">
      Dessa villkor styr av svensk lag. Tvister som uppstår i samband med användningen
      av Svenskt Vin ska lösas genom förhandling. Om förhandling inte räcker,
      är allmän domstol i Sverige behörig.
    </p>
  </section>

  <section>
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">9. Ändring av villkor</h2>
    <p style="line-height:1.6">
      Vi förbehåller oss rätten att ändra dessa villkor. Vid väsentliga ändringar
      meddelar vi via webbplatsen och/eller e-post.
    </p>
  </section>
</main>
