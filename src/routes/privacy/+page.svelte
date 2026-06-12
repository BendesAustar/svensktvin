<!-- src/routes/privacy/+page.svelte -->
<svelte:head><title>Integritetspolicy — Svenskt Vin</title></svelte:head>

<main style="max-width:700px;margin:5vh auto;padding:0 1rem;font-family:sans-serif">
  <a href="/" style="color:#555;font-size:0.9rem">← Tillbaka</a>
  <h1 style="margin:0.5rem 0">Integritetspolicy</h1>
  <p style="color:#666;margin-bottom:2rem">Senast uppdaterad: {new Date().toISOString().split('T')[0]}</p>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">1. Vilken data vi samlar in</h2>
    <p style="line-height:1.6">
      Svenskt Vin samlar in följande personuppgifter:<br>
      <strong>E-postadress</strong> — för kontoregistrering och inloggning via magic link.<br>
      <strong>Namn</strong> — för visning i profil och medlemslistor.<br>
      <strong>Vingårdsdata</strong> — vingårdsnamn, län, kommun, uppstartsår, total area.<br>
      <strong>Produktionsdata</strong> — block, skörd, mätvärden (Brix, syra, skördeuppgifter).<br>
      <strong>Inloggningsdata</strong> — sessioner (30 dagars giltighet), magic-link-tokenhashar (15 minuters giltighet).
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">2. Vad vi använder datan till</h2>
    <p style="line-height:1.6">
      Datan används enbart för att tillhandahålla Svenskt Vins tjänst för vingårdsregistrering och benchmarking.
      Vi delar inte, säljer eller vidarebefordrar personuppgifter till tredje part.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">3. Lagringsperiod</h2>
    <p style="line-height:1.6">
      <strong>Sessioner</strong> — raderas automatiskt efter 30 dagar.<br>
      <strong>Magic-link-token</strong> — raderas efter användning eller 15 minuter.<br>
      <strong>Konto och vingårdsdata</strong> — sparas tills du raderar ditt konto.<br>
      <strong>Skörderecord</strong> — sparas permanent för benchmarking (undantas vid kontorsradering).
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">4. Dina rättigheter</h2>
    <p style="line-height:1.6">
      Enligt EU:s dataskyddsgeneralförordning (GDPR) har du rätt till:<br>
      <strong>Access</strong> — begär en utskrift av all data vi har om dig.<br>
      <strong>Radering</strong> — begär radering av ditt konto och all tillhörande data.<br>
      <strong>Portabilitet</strong> — begär datan i ett maskinläsbart format.<br>
      <strong>Invändning</strong> — invänd mot behandling av din data.<br>
      <strong>Återkalle av samtycke</strong> — när som helst via kontots inställningar.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">5. Cookies</h2>
    <p style="line-height:1.6">
      Svenskt Vin använder endast nödvändiga cookies (sessionscookie) för inloggning.
      Ingen spårning eller tredjepartscookie används.
    </p>
  </section>

  <section style="margin-bottom:2rem">
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">6. Kontakt</h2>
    <p style="line-height:1.6">
      För frågor om dataskydd eller för att utöva dina rättigheter, kontakta oss via
      <a href="mailto:privacy@svensktvin.se">privacy@svensktvin.se</a>.
    </p>
  </section>

  <section>
    <h2 style="font-size:1.1rem;margin-bottom:0.5rem">7. Ändringar i integritetspolicyn</h2>
    <p style="line-height:1.6">
      Vi kan uppdatera denna policy när som helst. Vid väsentliga ändrar vi meddelande via webbplatsen.
      Kontrollera datumet längst upp för att se när policyn senast uppdaterades.
    </p>
  </section>
</main>
