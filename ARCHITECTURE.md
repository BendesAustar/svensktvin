# Svenskt Vin — HTMX + Go Templates Architecture

**Date:** 2026-06-15  
**Author:** DesignBrain (SvensktVin Migration)  
**Status:** Blueprint

---

## 1. Architecture Overview

### 1.1 New Stack

| Layer | Technology |
|-------|-----------|
| Runtime | Go 1.22+ (single binary) |
| Template Engine | Go `html/template` (built-in, no build step) |
| HTMX | `htmx.org@2.x` via CDN (dev) / inline embed (prod) |
| Alpine.js | `alpinejs@3.x` via CDN (dev) / inline embed (prod) |
| CSS | Tailwind CSS via CDN (dev) / prebuild (prod) |
| Database | PostgreSQL (unchanged — same schema) |
| Auth | Go native — session cookies, bcrypt, magic links |
| Email | Go native — `net/smtp` (unchanged logic) |
| Deployment | Single Dockerfile → single Go binary serving templates + static files |

### 1.2 Architecture Diagram

```
┌──────────────────────────────────────────────────────────┐
│                    Browser (Single App)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │ Go HTML  │  │ HTMX     │  │ Alpine.js│  │ Tailwind │ │
│  │ Templates│  │ (swap/   │  │ (client  │  │ (style)  │ │
│  │ (SSR)    │◀─│  partial  │  │ state)   │  │          │ │
│  └──────────┘  │  render)  │  │          │  │          │ │
│                └──────────┘  └──────────┘  └──────────┘ │
└──────────────────────────────────────────────────────────┘
         ▲
         │ HTMX POST/GET (form submission / hx-get)
         │
┌────────┴─────────────────────────────────────────────────┐
│            Single Go Binary (HTTP Server)                │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │  Template   │  │  Template    │  │  JSON API      │  │
│  │  Handler    │  │  Action      │  │  Handler       │  │
│  │  (renders   │  │  Handler     │  │  (HTMX partial │  │
│  │   full page)│  │  (form POST  │  │   HTML swap    │  │
│  │             │  │   → swap)    │  │   HTML)        │  │
│  └──────┬──────┘  └──────┬───────┘  └───────┬────────┘  │
│         │                │                   │           │
│  ┌──────┴────────────────┴───────────────────┴────────┐  │
│  │  Auth Middleware (session cookie, bcrypt, magic    │  │
│  │  link token verification)                          │  │
│  └─────────────────────────┬──────────────────────────┘  │
│                            │                              │
│  ┌─────────────────────────┴──────────────────────────┐  │
│  │  PostgreSQL (via pgx v5 — same pool as core-api)   │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### 1.3 Key Design Decisions

1. **Go serves ALL HTML** — both full-page renders and HTMX partial swaps via `html/template`
2. **HTMX replaces Svelte `fetch()`** — every client-side API call becomes `hx-get`/`hx-post`/`hx-put`/`hx-delete`
3. **Alpine.js for ~5% client state** — variety search autocomplete, dropdown toggles, modal state, cookie consent
4. **No build step in production** — Go template files serve directly; Tailwind via CDN in dev, `tailwindcss -o` in CI for prod
5. **Single binary** — Go compiles templates at build time (`//go:embed`) or serves from filesystem
6. **Swedish text preserved** — all user-visible text stays in Swedish
7. **Session cookies unchanged** — httpOnly, secure, SameSite=Lax, 30-day expiry
8. **bcrypt ≥ 12 rounds** — same security model

### 1.4 Migration Strategy

| Phase | Scope | Effort |
|-------|-------|--------|
| 1 | Auth pages (login, register, invite, forgot-password, set-password) | 2-3 days |
| 2 | Vineyard layout + dashboard (blocks table, benchmark teaser) | 2-3 days |
| 3 | Block CRUD (new, edit) with variety search (Alpine + HTMX) | 2 days |
| 4 | Harvest CRUD (new, edit, lock) | 2 days |
| 5 | Benchmark page | 1 day |
| 6 | Settings (vineyard + members + password) | 2 days |
| 7 | Static pages (privacy, terms, onboard) | 1 day |
| 8 | Dockerfile + deploy config + CI | 1 day |
| **Total** | | **~13-15 days** |

---

## 2. Template Hierarchy

### 2.1 File Structure

```
templates/
├── base.html              # Base template with nav, flash, footer, HTMX/Alpine CDNs
├── nav.html               # Navigation partial (logged-in + logged-out variants)
├── flash.html             # Flash message partial
├── form-errors.html       # Form error display partial
├── auth/
│   ├── login.html
│   ├── register.html
│   ├── forgot-password.html
│   └── set-password.html
├── vineyard/
│   ├── index.html         # Vineyard list page
│   ├── dashboard.html     # Vineyard detail (blocks table)
│   ├── blocks/
│   │   ├── new.html
│   │   └── edit.html
│   ├── harvest/
│   │   ├── new.html
│   │   └── edit.html
│   ├── benchmark.html
│   └── settings.html
├── invite/
│   ├── confirm.html
│   └── success.html
├── onboard.html
├── static/
│   ├── privacy.html
│   └── terms.html
└── error.html             # Global error page

static/
├── css/
│   └── app.css            # Tailwind output (prod) or CDN link (dev)
└── js/
    └── app.js             # Alpine.js + HTMX event handlers (if needed beyond attributes)
```

### 2.2 Base Template (`templates/base.html`)

```html
{{define "base"}}
<!DOCTYPE html>
<html lang="sv" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}{{.Title}}{{end}}</title>
    
    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfMzorYmKGrM1lGbI0i6YcfnBDWRfz94B0nTDnO2UcBZcDR40XyXGNq" 
            crossorigin="anonymous"></script>
    
    <!-- Alpine.js -->
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.14.8/dist/cdn.min.js"></script>
    
    <!-- Tailwind (CDN for dev, prebuilt CSS for prod) -->
    <script src="https://cdn.tailwindcss.com"></script>
    
    <!-- Custom styles -->
    <link rel="stylesheet" href="/static/css/app.css">
    
    {{block "head" .}}{{end}}
</head>
<body class="min-h-full bg-gray-50 font-sans text-gray-900">
    
    {{if .User}}
    <header class="bg-gray-100 border-b border-gray-300">
        <div class="max-w-7xl mx-auto px-4 py-2 flex justify-between items-center">
            <a href="/" class="no-underline font-semibold text-lg text-[#2d6a2d]">🍷 Svenskt Vin</a>
            <form method="POST" action="/logout" class="inline">
                <button type="submit" class="text-sm text-gray-500 bg-none border-none cursor-pointer">
                    Logga ut
                </button>
            </form>
        </div>
    </header>
    {{end}}
    
    <main class="max-w-7xl mx-auto px-4">
        {{template "flash.html" .}}
        {{block "content" .}}{{end}}
    </main>
    
    {{if not .NoCookieNotice}}
    <div x-data="{ shown: !localStorage.getItem('cookie_consent') }" 
         x-show="shown"
         x-init="$watch('shown', val => { if (!val) localStorage.setItem('cookie_consent', '1') })"
         class="fixed bottom-0 left-0 right-0 bg-gray-900 text-white p-4 z-50 flex justify-between items-center flex-wrap gap-3 text-sm">
        <div class="flex-1 min-w-48">
            <strong class="mr-2">Cookies</strong>
            Svenskt Vin använder endast nödvändiga sessionscookies för inloggning.
            Ingen spårning eller tredjepartscookie.
            <a href="/privacy" class="text-green-300 ml-1 underline">Läs mer</a>.
        </div>
        <div class="flex gap-2">
            <button @click="shown = false"
                    class="px-4 py-2 bg-[#2d6a2d] text-white border-none rounded text-sm cursor-pointer font-semibold">
                Acceptera
            </button>
        </div>
    </div>
    {{end}}
    
    {{block "scripts" .}}{{end}}
</body>
</html>
{{end}}
```

### 2.3 Nav Partial (`templates/nav.html`)

```html
{{define "nav"}}
<nav class="max-w-7xl mx-auto px-4 py-2 border-b border-gray-300 bg-white flex justify-between items-center">
    <div class="flex gap-1 items-center">
        <a href="/vineyard/{{.VineyardID}}" 
           class="text-gray-500 text-sm no-underline px-3 py-2 rounded {{if .IsHome}}text-[#2d6a2d] bg-green-50 font-medium{{else}}hover:bg-gray-100{{end}}">
            ← {{.VineyardName}}
        </a>
        <a href="/vineyard/{{.VineyardID}}/harvest/new"
           class="text-gray-500 text-sm no-underline px-3 py-2 rounded {{if .IsHarvest}}text-[#2d6a2d] bg-green-50 font-medium{{else}}hover:bg-gray-100{{end}}">
            Skörd
        </a>
        <a href="/vineyard/{{.VineyardID}}/benchmark"
           class="text-gray-500 text-sm no-underline px-3 py-2 rounded {{if .IsBenchmark}}text-[#2d6a2d] bg-green-50 font-medium{{else}}hover:bg-gray-100{{end}}">
            📊 Jämförelse
        </a>
    </div>
    {{if eq .Role "owner"}}
    <a href="/vineyard/{{.VineyardID}}/settings"
       class="text-gray-500 text-sm no-underline px-3 py-2 rounded {{if .IsSettings}}text-[#2d6a2d] bg-green-50 font-medium{{else}}hover:bg-gray-100{{end}}">
        ⚙️
    </a>
    {{end}}
</nav>
{{end}}
```

### 2.4 Flash Partial (`templates/flash.html`)

```html
{{define "flash"}}
{{range .Flashes}}
<div class="mb-4 p-4 rounded">
    {{if eq .Type "success"}}
    <p class="m-0 text-green-700 bg-green-100">✅ {{.Message}}</p>
    {{else if eq .Type "error"}}
    <p class="m-0 text-red-700 bg-red-100">❌ {{.Message}}</p>
    {{else}}
    <p class="m-0 text-blue-700 bg-blue-100">{{.Message}}</p>
    {{end}}
</div>
{{end}}
{{end}}
```

### 2.5 Form Errors Partial (`templates/form-errors.html`)

```html
{{define "form-errors"}}
{{if .Error}}
<p class="text-red-700 mb-4">{{.Error}}</p>
{{end}}
{{range .FieldErrors}}
<p class="text-red-700 text-sm mb-1">{{.Field}}: {{.Issue}}</p>
{{end}}
{{end}}
```

### 2.6 Template Usage Pattern

Every page template uses:

```html
{{template "base.html" .}}  <!-- wraps the whole page in the base layout -->
```

With `{{block "title" .}}` and `{{block "content" .}}` defining the page-specific sections.

---

## 3. HTMX Route Map

This maps every SvelteKit route to its Go handler + HTMX pattern.

### 3.1 Auth Routes

| URL | Method | HTMX Attributes | Go Handler | Response Type | Swap Decision |
|-----|--------|-----------------|------------|---------------|---------------|
| `/login` | GET | — | `handleLoginGET` | Full page HTML | Full page — initial render |
| `/login` | POST | `hx-post="/login"`<br>`hx-swap="outerHTML"`<br>`hx-target="#login-form"` | `handleLoginPOST` | Full page HTML (with error/success state) | Full page — form submission needs redirect-like behavior for success |
| `/login` | POST | `hx-post="/login"`<br>`hx-swap="innerHTML"`<br>`hx-target="#login-form"` | `handleLoginPOST` (error state) | Form with errors | Partial swap — errors only |
| `/logout` | POST | `hx-post="/logout"`<br>`hx-swap="none"` | `handleLogoutPOST` | `HX-Redirect: /login` header | No swap, use redirect header |
| `/register` | GET | — | `handleRegisterGET` | Full page HTML | Full page |
| `/register` | POST | `hx-post="/register"`<br>`hx-swap="outerHTML"` | `handleRegisterPOST` | Full page HTML (error or success) | Full page — redirect after success |
| `/invite?token=xxx` | GET | — | `handleInviteGET` | Full page HTML | Full page — token validation |
| `/invite/confirm` | GET | — | `handleInviteConfirmGET` | Full page HTML | Full page |
| `/invite/confirm` | POST | `hx-post="/invite/confirm"`<br>`hx-swap="none"` | `handleInviteConfirmPOST` | `HX-Redirect: /vineyard/{id}` | Redirect header |
| `/auth/forgot-password` | GET | — | `handleForgotPasswordGET` | Full page HTML | Full page |
| `/auth/forgot-password` | POST | `hx-post="/auth/forgot-password"`<br>`hx-swap="outerHTML"` | `handleForgotPasswordPOST` | Full page HTML (success state) | Full page — success shows different content |
| `/auth/set-password` | GET | — | `handleSetPasswordGET` | Full page HTML | Full page |
| `/auth/set-password` | POST | `hx-post="/auth/set-password"`<br>`hx-swap="outerHTML"` | `handleSetPasswordPOST` | Full page HTML (success or error) | Full page |
| `/auth/verify` | POST | `hx-post="/auth/verify"`<br>`hx-swap="none"` | `handleVerifyPOST` | JSON + `HX-Redirect` | Redirect header |

### 3.2 Vineyard Routes

| URL | Method | HTMX Attributes | Go Handler | Response Type | Swap Decision |
|-----|--------|-----------------|------------|---------------|---------------|
| `/` | GET | — | `handleLandingGET` | Full page HTML (vineyard list) | Full page |
| `/vineyard` | GET | — | `handleVineyardListGET` | Full page HTML (redirect to first vineyard) | Full page |
| `/vineyard/{id}` | GET | — | `handleVineyardGET` | Full page HTML (blocks table) | Full page |
| `/vineyard/{id}/blocks/new` | GET | — | `handleBlockNewGET` | Full page HTML | Full page |
| `/vineyard/{id}/blocks/new` | POST | `hx-post="/vineyard/{id}/blocks/new"`<br>`hx-swap="none"` | `handleBlockNewPOST` | `HX-Redirect: /vineyard/{id}` | Redirect header |
| `/vineyard/{id}/blocks/{blockId}/edit` | GET | — | `handleBlockEditGET` | Full page HTML | Full page |
| `/vineyard/{id}/blocks/{blockId}/edit` | POST | `hx-post="/vineyard/{id}/blocks/{blockId}/edit"`<br>`hx-swap="none"` | `handleBlockEditPOST` | `HX-Redirect: /vineyard/{id}` | Redirect header |
| `/vineyard/{id}/harvest/new` | GET | — | `handleHarvestNewGET` | Full page HTML | Full page |
| `/vineyard/{id}/harvest/new` | POST | `hx-post="/vineyard/{id}/harvest/new"`<br>`hx-swap="none"` | `handleHarvestNewPOST` | `HX-Redirect: /vineyard/{id}` | Redirect header |
| `/vineyard/{id}/harvest/{recordId}/edit` | GET | — | `handleHarvestEditGET` | Full page HTML | Full page |
| `/vineyard/{id}/harvest/{recordId}/edit` | POST | `hx-post="/vineyard/{id}/harvest/{recordId}/edit"`<br>`hx-swap="none"` | `handleHarvestEditPOST` | `HX-Redirect: /vineyard/{id}` | Redirect header |
| `/vineyard/{id}/benchmark` | GET | — | `handleBenchmarkGET` | Full page HTML | Full page |
| `/vineyard/{id}/settings` | GET | — | `handleSettingsGET` | Full page HTML | Full page |
| `/vineyard/{id}/settings` | POST | `hx-post="/vineyard/{id}/settings"`<br>`hx-swap="outerHTML"`<br>`hx-target="#settings-form"` | `handleSettingsPOST` | Form with flash/error feedback | Partial swap for form errors; full page for success |

### 3.3 Vineyard Block Harvest Lock Routes

| URL | Method | HTMX Attributes | Go Handler | Response Type | Swap Decision |
|-----|--------|-----------------|------------|---------------|---------------|
| `/vineyard/{id}/blocks/{blockId}/harvest/lock` | POST | `hx-post="/vineyard/{id}/blocks/{blockId}/harvest/lock"`<br>`hx-swap="none"` | `handleHarvestLockPOST` | `HX-Redirect: /vineyard/{id}/harvest/new?block_id={blockId}` | Redirect header |
| `/vineyard/{id}/blocks/{blockId}/harvest/lock` | DELETE | `hx-delete="/vineyard/{id}/blocks/{blockId}/harvest/lock"`<br>`hx-swap="none"` | `handleHarvestUnlockPOST` | `HX-Redirect: /vineyard/{id}` | Redirect header |
| `/vineyard/{id}/blocks/{blockId}/harvest/lock` | POST (extend) | `hx-post="/vineyard/{id}/blocks/{blockId}/harvest/lock/extend"`<br>`hx-swap="none"` | `handleHarvestExtendPOST` | `HX-Trigger: lock-extended` + partial reload | Trigger header |

### 3.4 Static Pages

| URL | Method | HTMX Attributes | Go Handler | Response Type |
|-----|--------|-----------------|------------|---------------|
| `/onboard` | GET | — | `handleOnboardGET` | Full page HTML |
| `/onboard` | POST | `hx-post="/onboard"`<br>`hx-swap="none"` | `handleOnboardPOST` | `HX-Redirect: /vineyard/{id}` |
| `/privacy` | GET | — | `handlePrivacyGET` | Full page HTML |
| `/terms` | GET | — | `handleTermsGET` | Full page HTML |

### 3.5 API Routes (HTMX partial swaps)

These are JSON endpoints from the existing `core-api` that become HTMX partial HTML endpoints:

| URL | Method | HTMX Attributes | Go Handler | Response Type | Swap Decision |
|-----|--------|-----------------|------------|---------------|---------------|
| `/api/varieties/search?q=xxx` | GET | `hx-get="/api/varieties/search?q={q}"`<br>`hx-trigger="input changed delay:300ms from:#variety-search"`<br>`hx-swap="innerHTML"`<br>`hx-target="#variety-results"` | `handleVarietySearchGET` | HTML fragment (dropdown list) | Partial swap — variety dropdown |
| `/api/geo/reverse` | POST | `hx-post="/api/geo/reverse"`<br>`hx-swap="none"` | `handleGeoReversePOST` | `HX-Trigger: geo-reverse-success` with lat/lon values | Trigger header |
| `/api/account/export` | GET | `hx-get="/api/account/export"`<br>`hx-swap="none"` | `handleAccountExportGET` | JSON (download trigger) | No swap — download |
| `/api/account/delete` | POST | `hx-post="/api/account/delete"`<br>`hx-swap="none"` | `handleAccountDeletePOST` | `HX-Redirect: /login` | Redirect header |
| `/health` | GET | — | `handleHealthGET` | JSON health check | Not HTMX — diagnostic |

---

## 4. Alpine.js Reactivity Hooks

### 4.1 Variety Search Autocomplete (Block New/Edit)

```html
<div x-data="varietySearch()" x-init="init()" class="relative">
    <input 
        id="variety-search"
        type="text" 
        placeholder="Sök sort..."
        x-model="searchQuery"
        @input.debounce.300ms="search()"
        class="w-full p-2 border border-gray-300 rounded"
    />
    
    <!-- Loading state -->
    <div x-show="searching" x-cloak class="text-gray-500 text-sm py-1">
        Söker...
    </div>
    
    <!-- Error state -->
    <div x-show="searchError" x-cloak class="text-red-700 text-sm py-1">
        {{.searchError}}
    </div>
    
    <!-- High-confidence match -->
    <div x-show="selectedVarietyId !== null" x-cloak class="text-green-700 text-sm py-1">
        ✓ <span x-text="searchQuery"></span>
    </div>
    
    <!-- Results dropdown -->
    <ul x-show="searchResults.length > 0 && selectedVarietyId === null" 
        x-cloak
        class="border border-gray-200 rounded mb-2 list-none p-0">
        <template x-for="result in searchResults" :key="result.id">
            <li>
                <button type="button" 
                        @click="select(result.id, result.name)"
                        class="w-full text-left p-2 border-b border-gray-100 hover:bg-gray-50 cursor-pointer">
                    <span x-text="result.name"></span>
                    <span class="text-gray-400 text-sm"> 
                        (<span x-text="result.color"></span>
                        <span x-show="result.piwi"> · PIWI</span>)
                    </span>
                </button>
            </li>
        </template>
    </ul>
    
    <!-- No results fallback -->
    <button type="button" 
            x-show="searchResults.length > 0 && selectedVarietyId === null"
            @click="useCustom()"
            x-cloak
            class="px-3 py-1 bg-none border border-gray-300 rounded text-sm cursor-pointer">
        Ingen av dessa — använd detta namn
    </button>
    
    <!-- Hidden inputs for form submission -->
    <input type="hidden" name="variety_id" x-model="selectedVarietyId" />
    <input type="hidden" name="variety_name" x-model="customVarietyName" />
</div>

<script>
function varietySearch() {
    return {
        searchQuery: '',
        searchResults: [],
        selectedVarietyId: null,
        customVarietyName: '',
        searching: false,
        searchError: '',
        highConfidence: false,
        
        init() {},
        
        async search() {
            const q = this.searchQuery;
            if (!q || q.length < 2) {
                this.searchResults = [];
                this.highConfidence = false;
                return;
            }
            this.searching = true;
            this.searchError = '';
            
            try {
                const res = await fetch(`/api/varieties/search?q=${encodeURIComponent(q)}`);
                const data = await res.json();
                this.searchResults = data.matches;
                this.highConfidence = data.high_confidence;
                
                if (data.high_confidence && data.matches.length > 0) {
                    this.select(data.matches[0].id, data.matches[0].name);
                }
            } catch (e) {
                this.searchError = 'Sökningen misslyckades.';
                this.searchResults = [];
            } finally {
                this.searching = false;
            }
        },
        
        select(id, name) {
            this.selectedVarietyId = id;
            this.searchQuery = name;
            this.searchResults = [];
            this.highConfidence = false;
            this.customVarietyName = '';
        },
        
        useCustom() {
            this.selectedVarietyId = null;
            this.customVarietyName = this.searchResults[0]?.name ?? '';
            this.searchResults = [];
            this.highConfidence = false;
        }
    };
}
</script>
```

### 4.2 Cookie Consent

```html
<div x-data="{ shown: !localStorage.getItem('cookie_consent') }" 
     x-show="shown"
     x-init="$watch('shown', val => { if (!val) localStorage.setItem('cookie_consent', '1') })"
     class="fixed bottom-0 left-0 right-0 bg-gray-900 text-white p-4 z-50">
    <!-- Cookie notice content -->
    <button @click="shown = false" class="px-4 py-2 bg-[#2d6a2d] text-white rounded">
        Acceptera
    </button>
</div>
```

### 4.3 Password Visibility Toggle

```html
<div class="relative">
    <input type="password" 
           x-bind:type="showPassword ? 'text' : 'password'"
           name="password"
           class="w-full p-2 pr-10 border border-gray-300 rounded" />
    <button type="button"
            @click="showPassword = !showPassword"
            class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-500 text-sm bg-none border-none cursor-pointer">
        <span x-text="showPassword ? 'Dölj' : 'Visa'"></span>
    </button>
</div>
```

### 4.4 Membership Request Toggle (Login Page)

```html
<div x-data="{ showMembership: false }">
    <!-- "Begär medlemskap" button -->
    <button @click="showMembership = !showMembership"
            class="w-full text-center px-4 py-3 bg-gray-100 border border-gray-300 rounded">
        Begär medlemskap
    </button>
    
    <!-- Membership form (shown/hidden) -->
    <div x-show="showMembership" x-cloak x-transition class="mt-4 pt-4 border-t border-gray-300">
        <!-- Membership form fields -->
    </div>
</div>
```

### 4.5 Block Lock Timer (Harvest Page)

```html
<div x-data="{
    lock: {{.Lock | toJson}},
    minutesLeft() {
        if (!this.lock) return null;
        const diff = new Date(this.lock.expiresAt).getTime() - Date.now();
        return Math.max(0, Math.ceil(diff / 60000));
    }
}">
    <template x-if="lock">
        <div class="bg-blue-50 p-2 rounded mb-3 flex justify-between items-center">
            <span class="text-sm text-blue-700">🔒 Blocket är låst · <span x-text="minutesLeft()"></span> min kvar</span>
            <div class="flex gap-2">
                <button @click="fetch('/vineyard/{{.VineyardID}}/blocks/{{.BlockID}}/harvest/lock/extend', {method:'POST'})"
                        class="px-3 py-1 bg-white text-blue-700 border border-blue-700 rounded text-sm cursor-pointer">
                    Förläng lås
                </button>
                <button @click="fetch('/vineyard/{{.VineyardID}}/blocks/{{.BlockID}}/harvest/lock', {method:'DELETE'})"
                        class="px-3 py-1 bg-white text-red-700 border border-red-700 rounded text-sm cursor-pointer">
                    Lås upp
                </button>
            </div>
        </div>
    </template>
</div>
```

### 4.6 GPS Location Request (Onboard)

```html
<div x-data="{ geolocationError: null }">
    <button type="button" 
            @click="navigator.geolocation.getCurrentPosition(pos => {
                document.getElementById('lat').value = pos.coords.latitude.toFixed(6);
                document.getElementById('lon').value = pos.coords.longitude.toFixed(6);
            }, () => { geolocationError = 'Kunde inte hämta plats.'; })"
            class="mb-3 px-4 py-2 bg-green-50 border border-[#2d6a2d] text-[#2d6a2d] rounded cursor-pointer">
        📍 Hämta GPS-position
    </button>
    <div x-show="geolocationError" x-text="geolocationError" x-cloak class="text-yellow-800 bg-yellow-100 p-3 rounded mb-4"></div>
    <input id="lat" type="hidden" name="lat" />
    <input id="lon" type="hidden" name="lon" />
</div>
```

---

## 5. Component/Partial Map

### 5.1 Go Template Inheritance Hierarchy

```
base.html
├── nav.html           (include)
├── flash.html         (define)
├── form-errors.html   (define)
└── footer.html        (define, minimal)

auth/login.html
├── {{template "base.html" .}}
└── {{define "title"}}Logga in — Svenskt Vin{{end}}
└── {{define "content"}}
    <!-- login form -->
    {{template "form-errors.html" .}}
{{end}}

vineyard/dashboard.html
├── {{template "base.html" .}}
├── {{template "nav.html" .}}  (inline or include)
└── {{define "title"}}{vineyard.name} — Svenskt Vin{{end}}
└── {{define "content"}}
    <!-- vineyard dashboard -->
{{end}}
```

### 5.2 Concrete Template: Login Page

```html
{{template "base.html" .}}

{{define "title"}}Logga in — Svenskt Vin{{end}}

{{define "content"}}
<main class="max-w-md mx-auto mt-24 px-4">
    <h1 class="text-2xl mb-2">Svenskt Vin</h1>
    
    {{if .Sent}}
    <!-- Success: magic link sent -->
    <div class="bg-green-100 p-4 rounded mb-4">
        <p class="m-0">Om ett konto finns för den adressen har du fått en inloggningslänk via e-post.</p>
    </div>
    <a href="/" class="text-[#2d6a2d]">← Tillbaka</a>
    
    {{else if .MembershipSent}}
    <!-- Success: membership request sent -->
    <div class="bg-green-100 p-4 rounded mb-4">
        <p class="m-0">Tack! Vi skickar en inbjudningslänk så snart vi har godkänt din förfrågan.</p>
    </div>
    <a href="/" class="text-[#2d6a2d]">← Tillbaka</a>
    
    {{else}}
    <!-- Error state -->
    {{template "form-errors.html" .}}
    
    <!-- Vineyard context -->
    {{if .Vineyard}}
    <div class="bg-blue-50 p-4 rounded mb-4 border-l-4 border-blue-600">
        <p class="text-sm text-gray-500 mb-1">Du har blivit inbjuden att gå med i</p>
        <p class="text-xl font-semibold">{{.Vineyard.Name}}</p>
    </div>
    {{end}}
    
    <!-- Password login form -->
    <form method="POST" hx-post="/login" hx-swap="outerHTML" hx-target="#login-form">
        <input type="hidden" name="action" value="login_password" />
        <input type="hidden" name="invite_token" value="{{.InviteToken}}" />
        
        <label for="email" class="block mb-1 text-sm">E-postadress</label>
        <input id="email" type="email" name="email" required value="{{.Email}}"
               class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
        
        <div class="mt-3 relative">
            <label for="password" class="block mb-1 text-sm">Lösenord</label>
            <input id="password" type="password" name="password"
                   class="w-full p-2 pr-10 border border-gray-300 rounded text-lg box-border" />
            <button type="button" 
                    onclick="togglePassword('password', this)"
                    class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-500 text-sm bg-none border-none cursor-pointer">
                Visa
            </button>
        </div>
        
        <button type="submit"
                class="w-full mt-3 p-3 bg-[#2d6a2d] text-white border-none rounded text-lg cursor-pointer">
            Logga in
        </button>
    </form>
    
    <p class="my-4 text-center">
        <a href="/auth/forgot-password" class="text-[#2d6a2d] text-sm">Glömt lösenord?</a>
    </p>
    
    <div class="border-t border-gray-300 mt-6 pt-6">
        <p class="text-gray-500 text-sm text-center mb-2">Har du inget konto?</p>
        
        {{if .InviteToken}}
        <a href="/register?invite={{.InviteToken}}"
           class="block text-center p-3 bg-white border-2 border-[#2d6a2d] text-[#2d6a2d] rounded text-lg no-underline cursor-pointer">
            Skapa konto med inbjudan
        </a>
        {{else}}
        <button onclick="document.querySelector('#register-link').style.display='block'; this.style.display='none'"
                class="block text-center p-3 w-full bg-white border-2 border-[#2d6a2d] text-[#2d6a2d] rounded text-lg cursor-pointer mb-2">
            Skapa konto
        </button>
        
        <button type="button"
                onclick="document.querySelector('#membership-form').classList.toggle('hidden')"
                class="block text-center p-2 w-full bg-gray-100 border border-gray-300 rounded text-sm cursor-pointer">
            Begär medlemskap
        </button>
        {{end}}
    </div>
    
    <!-- Membership request form -->
    <div id="membership-form" class="hidden mt-4 pt-4 border-t border-gray-300">
        <p class="text-sm text-gray-500 mb-3">
            Skicka en förfrågan till oss. Vi godkänner och skickar en inbjudningslänk via e-post.
        </p>
        <form method="POST" hx-post="/login" hx-swap="outerHTML">
            <input type="hidden" name="action" value="request_membership" />
            <label for="mem-email" class="block mb-1 text-sm">E-postadress</label>
            <input id="mem-email" type="email" name="email" required
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            <label for="mem-name" class="block mt-3 mb-1 text-sm">Namn</label>
            <input id="mem-name" type="text" name="name" required
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            <button type="submit"
                    class="w-full mt-3 p-3 bg-gray-200 border border-gray-300 rounded text-lg cursor-pointer">
                Skicka förfrågan
            </button>
        </form>
    </div>
    {{end}}
</main>
{{end}}

{{define "scripts"}}
<script>
function togglePassword(inputId, btn) {
    const input = document.getElementById(inputId);
    const isPassword = input.type === 'password';
    input.type = isPassword ? 'text' : 'password';
    btn.textContent = isPassword ? 'Dölj' : 'Visa';
}
</script>
{{end}}
```

### 5.3 Concrete Template: Vineyard Dashboard

```html
{{template "base.html" .}}
{{template "nav.html" .}}

{{define "title"}}{{.Vineyard.Name}} — Svenskt Vin{{end}}

{{define "content"}}
<main class="max-w-7xl mx-auto px-4">
    <div class="mb-8">
        <h1 class="mt-0 mb-1">{{.Vineyard.Name}}</h1>
        <p class="text-gray-500 m-0">
            {{.Vineyard.County}} · {{.Vineyard.Municipality}}
            {{if .Vineyard.EstablishedYear}} · Startad {{.Vineyard.EstablishedYear}}{{end}}
            {{if .Vineyard.TotalAreaHA}} · {{.Vineyard.TotalAreaHA}} ha{{end}}
        </p>
        <p class="text-gray-500 m-0 mt-1">
            {{if .Vineyard.Organic}}🌿 Ekologisk{{end}}
            {{if and .Vineyard.Organic .Vineyard.Biodynamic}} · {{end}}
            {{if .Vineyard.Biodynamic}}🌀 Biodynamisk{{end}}
        </p>
    </div>
    
    <!-- Benchmark teaser -->
    {{if .BenchmarkTeaser}}
    <div class="bg-green-50 p-4 rounded mb-8">
        <h3 class="mt-0 mb-1 text-base">Benchmark — {{.BenchmarkTeaser.VarietyName}}</h3>
        <p class="m-0 text-sm">
            Din skörd: <strong>{{.BenchmarkTeaser.UserYieldKgHa}}</strong> kg/ha
            <span class="text-gray-400"> · {{.BenchmarkTeaser.VineyardCount}} vingårdar i {{.Vineyard.County}}</span>
        </p>
    </div>
    {{end}}
    
    <!-- Blocks section -->
    <div class="mb-6">
        <div class="flex justify-between items-center mb-4">
            <h2 class="mt-0">Block</h2>
            <a href="/vineyard/{{.Vineyard.ID}}/blocks/new"
               class="px-4 py-2 bg-[#2d6a2d] text-white rounded no-underline text-sm">
                + Nytt block
            </a>
        </div>
        
        {{if eq (len .Blocks) 0}}
        <p class="text-gray-400 text-center p-8">Inga block ännu. Skapa ditt första block!</p>
        {{else}}
        <table class="w-full border-collapse">
            <thead>
                <tr class="border-b-2 border-gray-100 text-left">
                    <th class="p-2 text-sm text-gray-500">Namn</th>
                    <th class="p-2 text-sm text-gray-500">Sort</th>
                    <th class="p-2 text-sm text-gray-500">Area</th>
                    <th class="p-2 text-sm text-gray-500">Senaste skörden</th>
                    <th class="p-2 text-sm text-gray-500"></th>
                </tr>
            </thead>
            <tbody>
                {{range .Blocks}}
                <tr class="border-b border-gray-100">
                    <td class="p-3">
                        <strong>{{.BlockName}}</strong>
                        {{if not .IsActive}}<span class="text-gray-400 text-sm"> (inaktiv)</span>{{end}}
                    </td>
                    <td class="p-3">
                        <span style="color: {{if eq .VarietyStatus "approved"}}#2d6a2d{{else}}#856404{{end}}">
                            {{.VarietyName}}
                        </span>
                        {{if eq .VarietyStatus "review_needed"}}
                        <span class="text-xs text-[#856404]"> (granskas)</span>
                        {{end}}
                    </td>
                    <td class="p-3 text-gray-500">{{.AreaHA}} ha</td>
                    <td class="p-3 text-gray-500">
                        {{if .LatestHarvest}}
                            {{.LatestHarvest.HarvestYear}}: {{.LatestHarvest.YieldKg}} kg
                        {{else}}
                            <span class="text-gray-300">—</span>
                        {{end}}
                    </td>
                    <td class="p-3 text-right">
                        <button 
                            hx-post="/vineyard/{{$.Vineyard.ID}}/blocks/{{.ID}}/harvest/lock"
                            hx-swap="none"
                            hx-headers='{"Accept": "application/json"}'
                            hx-trigger="click"
                            hx-confirm="Vill du låsa blocket för skörd?"
                            @htmx:after-request="if(event.detail.successful) window.location.href='/vineyard/{{$.Vineyard.ID}}/harvest/new?block_id={{.ID}}'"
                            class="px-3 py-1 bg-[#2d6a2d] text-white border-none rounded text-sm cursor-pointer">
                            🌾 Skörd
                        </button>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{end}}
    </div>
</main>
{{end}}
```

### 5.4 Concrete Template: Block New (with Alpine variety search)

```html
{{template "base.html" .}}

{{define "title"}}Nytt block — Svenskt Vin{{end}}

{{define "content"}}
<main class="max-w-2xl mx-auto mt-20 px-4">
    <a href="/vineyard/{{.VineyardID}}" class="text-gray-500 text-sm">← Tillbaka</a>
    <h1 class="mt-2 mb-2">Nytt block</h1>
    
    <form method="POST" hx-post="/vineyard/{{.VineyardID}}/blocks/new" hx-swap="none">
        {{template "form-errors.html" .}}
        
        <label for="block_name" class="block mb-1 text-sm">Blocknamn <span class="text-red-700">*</span></label>
        <input id="block_name" type="text" name="block_name" required
               class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
        
        <!-- Alpine.js variety search -->
        <div x-data="varietySearch()" class="mt-4 mb-2">
            <label for="variety-search" class="block mb-1 text-sm">Sort <span class="text-red-700">*</span></label>
            <input id="variety-search" type="text" placeholder="Sök sort..."
                   x-model="searchQuery"
                   @input.debounce.300ms="search()"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <div x-show="searching" x-cloak class="text-gray-400 text-sm py-1">Söker...</div>
            <div x-show="searchError" x-cloak class="text-red-700 text-sm py-1" x-text="searchError"></div>
            
            <div x-show="selectedVarietyId !== null" x-cloak class="text-green-700 text-sm py-1">
                ✓ <span x-text="searchQuery"></span>
            </div>
            
            <ul x-show="searchResults.length > 0 && selectedVarietyId === null"
                x-cloak
                class="border border-gray-200 rounded mb-2 p-0 list-none">
                <template x-for="result in searchResults" :key="result.id">
                    <li>
                        <button type="button"
                                @click="select(result.id, result.name)"
                                class="w-full text-left p-2 border-b border-gray-100 hover:bg-gray-50 cursor-pointer">
                            <span x-text="result.name"></span>
                            <span class="text-gray-400 text-sm"> 
                                (<span x-text="result.color"></span>
                                <span x-show="result.piwi"> · PIWI</span>)
                            </span>
                        </button>
                    </li>
                </template>
            </ul>
            
            <button type="button"
                    x-show="searchResults.length > 0 && selectedVarietyId === null"
                    @click="useCustom()"
                    x-cloak
                    class="px-3 py-1 bg-none border border-gray-300 rounded text-sm cursor-pointer">
                Ingen av dessa — använd detta namn
            </button>
            
            <input type="hidden" name="variety_id" x-model="selectedVarietyId" />
            <input type="hidden" name="variety_name" x-model="customVarietyName" />
        </div>
        
        <fieldset class="border border-gray-300 p-4 rounded mb-4">
            <legend class="font-semibold px-2">Blockinformation</legend>
            
            <label for="area_ha" class="block mb-1 text-sm">Area (ha) <span class="text-red-700">*</span></label>
            <input id="area_ha" type="number" name="area_ha" step="0.001" min="0.01" required
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <label for="vine_count" class="block mt-3 mb-1 text-sm">Vinstockar</label>
            <input id="vine_count" type="number" name="vine_count" min="0"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <label for="planting_year" class="block mt-3 mb-1 text-sm">Planteringsår</label>
            <input id="planting_year" type="number" name="planting_year" min="1800" max="2030"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <label for="training_system" class="block mt-3 mb-1 text-sm">Uppbindningssystem</label>
            <input id="training_system" type="text" name="training_system" placeholder="t.ex. VSP, GDC"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <label for="aspect" class="block mt-3 mb-1 text-sm">Exposition</label>
            <select id="aspect" name="aspect"
                    class="w-full p-2 border border-gray-300 rounded text-lg box-border">
                <option value="">Välj</option>
                <option>N</option><option>NE</option><option>E</option>
                <option>SE</option><option>S</option><option>SW</option>
                <option>W</option><option>NW</option>
            </select>
            
            <label for="slope_degrees" class="block mt-3 mb-1 text-sm">Sluttning (grader)</label>
            <input id="slope_degrees" type="number" name="slope_degrees" step="0.1" min="0" max="90"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <label for="elevation_m" class="block mt-3 mb-1 text-sm">Höjmö (m)</label>
            <input id="elevation_m" type="number" name="elevation_m" min="0"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
        </fieldset>
        
        <button type="submit"
                class="w-full p-3 bg-[#2d6a2d] text-white border-none rounded text-lg cursor-pointer font-semibold">
            Skapa block
        </button>
    </form>
</main>
{{end}}

{{define "scripts"}}
<script>
function varietySearch() {
    return {
        searchQuery: '',
        searchResults: [],
        selectedVarietyId: null,
        customVarietyName: '',
        searching: false,
        searchError: '',
        highConfidence: false,
        async search() {
            const q = this.searchQuery;
            if (!q || q.length < 2) { this.searchResults = []; this.highConfidence = false; return; }
            this.searching = true; this.searchError = '';
            try {
                const res = await fetch(`/api/varieties/search?q=${encodeURIComponent(q)}`);
                const data = await res.json();
                this.searchResults = data.matches;
                this.highConfidence = data.high_confidence;
                if (data.high_confidence && data.matches.length > 0) {
                    this.select(data.matches[0].id, data.matches[0].name);
                }
            } catch (e) { this.searchError = 'Sökningen misslyckades.'; this.searchResults = []; }
            finally { this.searching = false; }
        },
        select(id, name) { this.selectedVarietyId = id; this.searchQuery = name; this.searchResults = []; this.highConfidence = false; },
        useCustom() { this.selectedVarietyId = null; this.customVarietyName = this.searchResults[0]?.name ?? ''; this.searchResults = []; this.highConfidence = false; }
    };
}
</script>
{{end}}
```

### 5.5 Concrete Template: Vineyard Settings

```html
{{template "base.html" .}}

{{define "title"}}Inställningar: {{.Vineyard.Name}} — Svenskt Vin{{end}}

{{define "content"}}
<main class="max-w-2xl mx-auto mt-20 px-4">
    <a href="/vineyard/{{.VineyardID}}" class="text-gray-500 text-sm">← Tillbaka</a>
    <h1 class="mt-2 mb-4">Inställningar</h1>
    
    <!-- Vineyard Settings Form -->
    <form id="settings-form" method="POST" hx-post="/vineyard/{{.VineyardID}}/settings" hx-swap="outerHTML" hx-target="#settings-form">
        <input type="hidden" name="action" value="update_vineyard" />
        
        <fieldset class="border border-gray-300 p-4 rounded mb-6">
            <legend class="font-semibold px-2">Vingårdsuppgifter</legend>
            
            <label for="name" class="block mb-1 text-sm">Namn <span class="text-red-700">*</span></label>
            <input id="name" type="text" name="name" required value="{{.Vineyard.Name}}"
                   class="w-full p-2 border border-gray-300 rounded text-lg box-border" />
            
            <label for="county" class="block mt-3 mb-1 text-sm">Län <span class="text-red-700">*</span></label>
            <select id="county" name="county" required
                    class="w-full p-2 border border-gray-300 rounded text-lg box-border">
                <option value="">Välj län</option>
                {{range .Counties}}<option {{if eq . $.Vineyard.County}}selected{{end}}>{{.}}</option>{{end}}
            </select>
            
            <!-- ... other fields ... -->
            
            <label for="organic" class="flex items-center mt-4 cursor-pointer">
                <input type="checkbox" name="organic" value="on" {{if .Vineyard.Organic}}checked{{end}} class="mr-2 text-lg" />
                Ekologisk
            </label>
            <label class="flex items-center cursor-pointer">
                <input type="checkbox" name="biodynamic" value="on" {{if .Vineyard.Biodynamic}}checked{{end}} class="mr-2 text-lg" />
                Biodynamisk
            </label>
        </fieldset>
        
        <button type="submit"
                class="w-full p-3 bg-[#2d6a2d] text-white border-none rounded text-lg cursor-pointer font-semibold">
            Spara ändringar
        </button>
    </form>
    
    <!-- Flash messages after HTMX swap -->
    {{if .PasswordSuccess}}
    <div class="bg-green-100 p-3 rounded mb-4">
        <p class="m-0 text-green-700 text-sm">✅ Lösenordet har ändrats.</p>
    </div>
    {{end}}
    {{if .PasswordError}}
    <div class="bg-red-100 p-3 rounded mb-4">
        <p class="m-0 text-red-700 text-sm">❌ {{.PasswordError}}</p>
    </div>
    {{end}}
    
    <!-- Password Change Form -->
    <fieldset class="border border-gray-300 p-4 rounded mb-6">
        <legend class="font-semibold px-2">Ändra lösenord</legend>
        <form method="POST" hx-post="/vineyard/{{.VineyardID}}/settings" hx-swap="none">
            <input type="hidden" name="action" value="change_password" />
            <!-- ... password fields ... -->
            <button type="submit" class="w-full mt-3 p-3 bg-[#2d6a2d] text-white border-none rounded text-lg cursor-pointer">
                Uppdatera lösenord
            </button>
        </form>
    </fieldset>
    
    <!-- Member Management -->
    <fieldset class="border border-gray-300 p-4 rounded mb-6">
        <legend class="font-semibold px-2">Medlemmar</legend>
        <!-- Members table -->
        <!-- Invite form -->
        <form method="POST" hx-post="/vineyard/{{.VineyardID}}/settings" hx-swap="none">
            <input type="hidden" name="action" value="invite_member" />
            <!-- ... invite fields ... -->
        </form>
    </fieldset>
</main>
{{end}}
```

---

## 6. HTMX + Alpine Integration Examples

### 6.1 Form Submission with HTMX Loading State

```html
<form hx-post="/vineyard/{{.VineyardID}}/settings" 
      hx-swap="outerHTML" 
      hx-target="#settings-form"
      hx-indicator="#settings-form .submit-btn">
    
    <!-- ... form fields ... -->
    
    <button type="submit" class="submit-btn">
        <span class="hx-loading">Sparar...</span>
        <span class="hx-loading-hidden">Spara ändringar</span>
    </button>
</form>

<!-- Or with Alpine loading indicator -->
<button type="submit" 
        x-data x-show="!$el.closest('form').getAttribute('hx-target')"
        class="relative px-4 py-2">
    Sparar...
    <span class="animate-spin inline-block w-4 h-4 border-2 border-white border-t-transparent rounded-full"></span>
</button>
```

### 6.2 Modal with HTMX Content Load

```html
<!-- Modal trigger -->
<button @click="modalOpen = true" class="px-4 py-2 bg-red-500 text-white rounded">
    Ta bort konto
</button>

<!-- Modal with Alpine + HTMX -->
<div x-data="{ modalOpen: false }" 
     x-show="modalOpen"
     @keydown.escape.window="modalOpen = false"
     class="fixed inset-0 z-50 flex items-center justify-center">
    <!-- Overlay -->
    <div @click="modalOpen = false" 
         class="fixed inset-0 bg-black bg-opacity-50"></div>
    <!-- Content -->
    <div class="relative bg-white p-6 rounded-lg max-w-md z-10">
        <h2 class="text-xl mb-4">Ta bort konto?</h2>
        <p class="mb-4 text-gray-600">Detta raderar permanent alla dina data.</p>
        <form hx-post="/api/account/delete" hx-swap="none">
            <div class="mb-4">
                <label class="flex items-center">
                    <input type="checkbox" name="confirm" value="true" required class="mr-2" />
                    Jag bekräftar radering
                </label>
            </div>
            <button type="submit"
                    hx-on::after-request="window.location.href='/login'"
                    class="px-4 py-2 bg-red-500 text-white rounded">
                Radera konto
            </button>
        </form>
        <button @click="modalOpen = false" class="ml-2 px-4 py-2 border border-gray-300 rounded">
            Avbryt
        </button>
    </div>
</div>
```

### 6.3 Tab Navigation with HTMX + Alpine Active State

```html
<!-- Not currently needed in SvensktVin, but useful pattern for future -->
<div x-data="{ activeTab: 'vineyard' }">
    <div class="flex border-b">
        <button @click="activeTab = 'vineyard'"
                :class="activeTab === 'vineyard' ? 'border-b-2 border-[#2d6a2d]' : ''"
                class="px-4 py-2 cursor-pointer">
            Vingård
        </button>
        <button @click="activeTab = 'members'"
                :class="activeTab === 'members' ? 'border-b-2 border-[#2d6a2d]' : ''"
                class="px-4 py-2 cursor-pointer">
            Medlemmar
        </button>
    </div>
    
    <div x-show="activeTab === 'vineyard'">
        <div hx-get="/vineyard/{{.VineyardID}}/tab/vineyard"
             hx-trigger="load"
             hx-swap="innerHTML"
             hx-target="#tab-content-vineyard">
        </div>
    </div>
    <div x-show="activeTab === 'members'">
        <div hx-get="/vineyard/{{.VineyardID}}/tab/members"
             hx-trigger="load"
             hx-swap="innerHTML"
             hx-target="#tab-content-members">
        </div>
    </div>
</div>
```

### 6.4 Search with Alpine Debounce + HTMX Fetch

The variety search already shown in section 4.1 is the primary example. Key points:
- `x-model` for two-way binding
- `@input.debounce.300ms` for throttling
- Direct `fetch()` call to JSON API (not HTMX) because we need JSON for high-confidence detection
- HTMX not used for search because the response is JSON, not HTML

### 6.5 Confirm + Delete Flow

```html
<!-- Block delete -->
<button 
    hx-post="/vineyard/{{.VineyardID}}/blocks/{{.BlockID}}"
    hx-swap="none"
    hx-method="DELETE"
    hx-confirm="Vill du ta bort blocket '{{.BlockName}}'? Detta kan inte ångras."
    hx-headers='{"Accept": "application/json"}'
    @htmx:after-request="if(event.detail.successful) window.location.reload()"
    class="px-3 py-1 bg-red-500 text-white border-none rounded text-sm cursor-pointer">
    Ta bort
</button>

<!-- Harvest delete -->
<button 
    hx-delete="/vineyard/{{.VineyardID}}/harvest/{{.RecordID}}"
    hx-swap="none"
    hx-confirm="Vill du ta bort skörden från {{.RecordID}}?"
    @htmx:after-request="if(event.detail.successful) window.location.reload()"
    class="px-3 py-1 bg-red-500 text-white border-none rounded text-sm cursor-pointer">
    Ta bort
</button>
```

---

## 7. Migration Mapping Table

| Svelte Page | Current Svelte Feature | New Go Template + HTMX/Alpine | Example (Svelte → New) |
|-------------|----------------------|-------------------------------|------------------------|
| `+page.svelte` (landing) | Svelte `{#each}` loop over `data.vineyards` | Go `{{range .Vineyards}}` in template | `{#each data.vineyards as v}<a href="/vineyard/{v.id}">...</a>{/each}` → `{{range .Vineyards}}<a href="/vineyard/{{.ID}}">...</a>{{end}}` |
| `+layout.svelte` | Svelte `onMount` + `document.cookie` check | Alpine.js `x-data` with `localStorage` | `onMount(()=>{showCookieNotice=!document.cookie.includes('cookie_consent=')})` → `x-data="{shown:!localStorage.getItem('cookie_consent')}"` |
| `vineyard/[id]/+layout.svelte` | Svelte `$: vineyardId = $page.params.id` reactive variables | Go template variables + CSS conditional classes | `$: isHome = /^\/vineyard\/\d+$/.test(path)` → `class="{{if .IsHome}}active{{end}}"` |
| `vineyard/[id]/+page.svelte` | Svelte `{#each}` block table, `fetch()` for lock | Go template table, HTMX `hx-post` for harvest lock | `<button onclick={() => harvestBlock(id)}>🌾 Skörd</button>` → `<button hx-post="/vineyard/{{.ID}}/blocks/{{.BlockID}}/harvest/lock" hx-swap="none">🌾 Skörd</button>` |
| `vineyard/[id]/benchmark/+page.svelte` | Svelte `{#each}` over 3 tables | Go `{{range}}` over 3 slices | Same pattern with Go `{{range}}` |
| `vineyard/[id]/blocks/new/+page.svelte` | Svelte `$state`, `fetch()` to `/api/varieties/search` | Alpine.js `x-data` with `@input.debounce.300ms="search()"` | `$: searchResults = []` → `searchResults: []` in Alpine `x-data` |
| `vineyard/[id]/blocks/[blockId]/edit/+page.svelte` | Svelte `$state` + pre-filled form values | Alpine.js same pattern + Go pre-filled values | Same as new block, with Go values pre-filled |
| `vineyard/[id]/harvest/new/+page.svelte` | Svelte `onMount` for lock acquisition, `fetch()` for lock/unlock | Alpine.js `x-data` with lock state + HTMX `hx-post`/`hx-delete` | `await fetch(.../harvest/lock, {method:'POST'})` → `<button hx-post="/.../harvest/lock" hx-swap="none">Lås</button>` |
| `vineyard/[id]/harvest/[recordId]/edit/+page.svelte` | Svelte pre-filled form | Go template with pre-filled values | `<input value={record.yield_kg}>` → `<input value="{{.Record.YieldKG}}" />` |
| `vineyard/[id]/settings/+page.svelte` | Multiple Svelte `<form use:enhance>`, `{#if}` states | Multiple Go `<form hx-post>`, `{{if .Flash}}` | `<form use:enhance>` → `<form hx-post="/vineyard/{{.VineyardID}}/settings" hx-swap="outerHTML">` |
| `login/+page.svelte` | Svelte `use:enhance`, `{#if form?.sent}` states | Go template with `{{if .Sent}}` states, HTMX for form | `<form use:enhance>` → `<form hx-post="/login" hx-swap="outerHTML">` |
| `register/+page.svelte` | Svelte `{#if form?.error}` error display | Go template `{{if .Error}}` + `{{template "form-errors.html" .}}` | Same pattern |
| `invite/+server.ts` | SvelteKit `RequestHandler` with `throw redirect()` | Go handler with `http.Redirect()` | Same redirect pattern |
| `invite/confirm/+page.svelte` | Svelte form with `use:enhance` | Go template with HTMX form | `<form use:enhance>` → `<form hx-post="/invite/confirm" hx-swap="none">` |
| `auth/forgot-password/+page.svelte` | Svelte `{#if form?.sent}` | Go template `{{if .Sent}}` | Same pattern |
| `auth/set-password/+page.svelte` | Svelte `validatePasswords()` client-side | Go template + HTML5 `minlength` + Alpine.js | `<script>function validatePasswords()</script>` → HTML5 `minlength="8"` + Go server-side validation |
| `onboard/+page.svelte` | Svelte `navigator.geolocation` | Alpine.js `@click` with `navigator.geolocation` | Same pattern with Alpine |
| `privacy/+page.svelte` | Svelte `{new Date().toISOString()}` | Go `{{.Now}}` (passed from handler) | `{new Date().toISOString().split('T')[0]}` → `{{.Now.Format "2006-01-02"}}` |
| `terms/+page.svelte` | Svelte `{new Date().toISOString()}` | Go `{{.Now}}` | Same as privacy |
| `api/account/delete/+server.ts` | Svelte `fetch()` POST with JSON body | Alpine.js modal + HTMX `hx-post` | `fetch('/api/account/delete', {method:'POST', body:JSON.stringify({confirm:true})})` → `<form hx-post="/api/account/delete" hx-swap="none"><input type="checkbox" name="confirm" value="true"><button type="submit">Radera</button></form>` |
| `api/account/export/+server.ts` | Svelte `fetch()` GET, JSON response | HTMX `hx-get` with `hx-on::after-request` download trigger | `fetch('/api/account/export').then(r=>r.blob())` → `<a hx-get="/api/account/export" hx-on::after-request="downloadBlob(event.detail.xhr)">Exportera data</a>` |
| `api/geo/reverse/+server.ts` | Svelte `fetch()` POST with lat/lon | Go handler returns JSON for Alpine to use | Same JSON endpoint, consumed by Alpine in onboard |
| `vineyard/+page.svelte` | Simple Svelte page | Go redirect to first vineyard | Full page redirect in handler |
| `logout/+page.server.ts` | Svelte `cookies.delete()` | Go `http.Redirect` with cookie cleanup | Same pattern in Go |

---

## 8. Error Handling & Validation Strategy

### 8.1 Server-Side Validation Flow

```
User submits form via HTMX hx-post
    ↓
Go handler receives FormData
    ↓
Server-side validation in Go
    ↓
Validation FAILS →
    • Return HTTP 400
    • Render form template with error data injected
    • HTMX swaps the form outerHTML (re-renders entire form with errors)
    ↓
Validation SUCCEEDS →
    • Perform DB operation
    • Success → HX-Redirect header → browser navigates to new page
    • Error → render form with error message → HTMX swap
```

### 8.2 Go Error Response Pattern

```go
func (h *handler) handleBlockNew(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    
    blockName := strings.TrimSpace(r.FormValue("block_name"))
    areaHA := r.FormValue("area_ha")
    
    if blockName == "" {
        // Re-render form with error
        h.templates.ExecuteTemplate(w, "blocks/new.html", map[string]any{
            "VineyardID": vineyardID,
            "Error":      "Blocknamn krävs.",
            // Pass back form values for re-display
            "BlockName":  blockName,
            "AreaHA":     areaHA,
        })
        return
    }
    
    // ... success → redirect
}
```

### 8.3 HTMX Response Headers

| Header | Purpose | Example |
|--------|---------|---------|
| `HX-Redirect` | Navigate browser to new URL | `HX-Redirect: /vineyard/42` |
| `HX-Replace-URL` | Update browser URL without reload | `HX-Replace-URL: /vineyard/42/blocks/new` |
| `HX-Retarget` | Swap element from another page | `HX-Retarget: #settings-form` |
| `HX-Trigger` | Dispatch custom JS event | `HX-Trigger: form-success` |
| `HX-Reswap` | Override swap method | `HX-Reswap: outerHTML` |

### 8.4 Client-Side Validation

- HTML5 `required`, `minlength`, `type="email"`, `type="number"`, `min`, `max` for immediate feedback
- Alpine.js for complex validation (password match, form state)
- No client-side validation replaces server-side — Go always validates

### 8.5 Flash Messages via HTMX

For POST/Redirect/Get pattern with HTMX, we use a session-based flash store:

```go
// After successful form submission
session.Flash("success", "Vingården har uppdaterats.")
w.Header().Set("HX-Redirect", "/vineyard/42/settings")
```

The base template renders flashes on every page load.

---

## 9. Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Copy go module files first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /svensktvin ./cmd/web

# Runtime stage
FROM debian:bookworm-slim AS runtime

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary and templates
COPY --from=builder /svensktvin /app/svensktvin
COPY templates/ /app/templates/
COPY static/ /app/static/

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["/app/svensktvin"]
```

### 9.1 Docker Compose

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://sv_app:sv_dev_pass@db:5432/svensktvin
      - SESSION_SECRET=${SESSION_SECRET}
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_USER=${SMTP_USER}
      - SMTP_PASS=${SMTP_PASS}
      - SMTP_FROM=${SMTP_FROM}
      - APP_HOST=${APP_HOST}
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped
  
  db:
    image: postgres:16
    environment:
      POSTGRES_DB: svensktvin
      POSTGRES_USER: sv_app
      POSTGRES_PASSWORD: sv_dev_pass
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sv_app"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  pgdata:
```

---

## 10. Risk Assessment

### 10.1 Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| HTMX partial swap breaks CSS classes | Medium | Medium | Use Tailwind utility classes exclusively; no external CSS dependencies |
| Go template security (XSS) | Low | High | Use Go's `html/template` (auto-escapes by default); never use `text/template` for user data |
| HTMX `hx-target` misidentification | Medium | Medium | Use explicit `id` attributes on all `hx-target` elements |
| Alpine.js loading state race conditions | Low | Low | Use `x-cloak` for flash; debounce search inputs |
| Session cookie migration from SvelteKit to Go | Low | High | Same cookie name (`session_id`), same httpOnly/secure/SameSite attributes |
| Tailwind CDN performance in prod | Medium | Medium | Use pre-built Tailwind CSS in production (`npx tailwindcss -o static/css/app.css`) |
| HTMX event conflict with existing JS | Low | Low | HTMX uses native DOM events; no jQuery dependency |

### 10.2 Migration Effort Estimates

| Category | Days | Notes |
|----------|------|-------|
| Auth pages (6 pages) | 2-3 | Straightforward SSR migration |
| Vineyard layout + dashboard | 2-3 | Blocks table needs careful mapping |
| Block CRUD (new + edit) | 2 | Alpine.js variety search is the complex part |
| Harvest CRUD (new + edit + lock) | 2 | Lock state adds complexity |
| Benchmark | 1 | Simple table rendering |
| Settings | 2 | Multiple forms, role-gated UI |
| Static pages | 1 | No interactivity |
| Docker + deploy | 1 | Build pipeline |
| **Total** | **~13-15** | |

### 10.3 Rollback Strategy

1. **Dual-stack deployment** — Keep SvelteKit running on `:5173` alongside Go on `:8080` during transition
2. **Nginx reverse proxy** — Route `/` to Go, fallback to SvelteKit on 502
3. **Feature flags** — Use `?migrate=0` query parameter to force SvelteKit even if Go is deployed
4. **Database compatibility** — Schema unchanged, zero data risk

### 10.4 What Might Break

1. **Variety search** — The current API (`/api/varieties/search`) returns JSON; we keep this as a JSON endpoint consumed by Alpine.js (not HTMX). If the JSON contract changes, search breaks.
2. **Geo reverse lookup** — `/api/geo/reverse` depends on Nominatim; if Nominatim is slow/unavailable, GPS location fails gracefully with manual fallback.
3. **Block lock mechanism** — The lock check is in the harvest new page server handler; if the lock TTL is shorter than the user's form-filling time, the lock expires and the save fails. Current behavior (error + "try again") is acceptable.
4. **Rate limiting** — The in-memory rate limiter in the Go handler needs to be tested; if running behind multiple replicas, a shared rate limiter (Redis) would be needed.
5. **CSRF protection** — HTMX forms don't automatically include CSRF tokens. Need to implement CSRF token generation + validation in Go middleware.

### 10.5 CSRF Protection (New Requirement)

HTMX forms need CSRF tokens since the SvelteKit `enhance()` provides them automatically:

```go
// In middleware, generate token
token := generateCSRFToken(r)
w.Header().Set("Set-Cookie", "csrf_token="+token+"; path=/; SameSite=Lax")

// In form template
<input type="hidden" name="_csrf" value="{{.CSRFToken}}" />

// In handler, validate
csrfToken := r.FormValue("_csrf")
if csrfToken != sessionCSRFToken {
    http.Error(w, "CSRF validation failed", http.StatusForbidden)
    return
}
```

---

## Appendix A: Complete Svelte Route → Go Handler Mapping

| # | Svelte Route | Go Handler | HTMX Pattern | Notes |
|---|-------------|------------|-------------|-------|
| 1 | `GET /` | `handleLandingGET` | None (full page) | Vineyard list or redirect |
| 2 | `GET /login` | `handleLoginGET` | None (full page) | Auth required check |
| 3 | `POST /login` | `handleLoginPOST` | `hx-post` | Password login + membership request |
| 4 | `GET /register` | `handleRegisterGET` | None (full page) | Invite token validation |
| 5 | `POST /register` | `handleRegisterPOST` | `hx-post` | User creation |
| 6 | `GET /invite?token=xxx` | `handleInviteGET` | None (full page) | Token validation + redirect |
| 7 | `GET /invite/confirm` | `handleInviteConfirmGET` | None (full page) | Cross-account invite confirmation |
| 8 | `POST /invite/confirm` | `handleInviteConfirmPOST` | `hx-post` | Accept invite |
| 9 | `GET /auth/forgot-password` | `handleForgotPasswordGET` | None (full page) | Email form |
| 10 | `POST /auth/forgot-password` | `handleForgotPasswordPOST` | `hx-post` | Send magic link |
| 11 | `GET /auth/set-password` | `handleSetPasswordGET` | None (full page) | Password setup form |
| 12 | `POST /auth/set-password` | `handleSetPasswordPOST` | `hx-post` | Set password |
| 13 | `POST /auth/verify` | `handleVerifyPOST` | `hx-post` | Magic link verification |
| 14 | `POST /logout` | `handleLogoutPOST` | `hx-post` | Session cleanup |
| 15 | `GET /vineyard` | `handleVineyardListGET` | None (full page) | User's vineyards |
| 16 | `GET /vineyard/{id}` | `handleVineyardGET` | None (full page) | Dashboard with blocks |
| 17 | `GET /vineyard/{id}/blocks/new` | `handleBlockNewGET` | None (full page) | Block creation form |
| 18 | `POST /vineyard/{id}/blocks/new` | `handleBlockNewPOST` | `hx-post` | Create block |
| 19 | `GET /vineyard/{id}/blocks/{blockId}/edit` | `handleBlockEditGET` | None (full page) | Block edit form |
| 20 | `POST /vineyard/{id}/blocks/{blockId}/edit` | `handleBlockEditPOST` | `hx-post` | Update block |
| 21 | `POST /vineyard/{id}/blocks/{blockId}/harvest/lock` | `handleHarvestLockPOST` | `hx-post` | Acquire lock |
| 22 | `DELETE /vineyard/{id}/blocks/{blockId}/harvest/lock` | `handleHarvestUnlockPOST` | `hx-delete` | Release lock |
| 23 | `POST /vineyard/{id}/blocks/{blockId}/harvest/lock/extend` | `handleHarvestExtendPOST` | `hx-post` | Extend lock TTL |
| 24 | `GET /vineyard/{id}/harvest/new` | `handleHarvestNewGET` | None (full page) | Harvest creation form |
| 25 | `POST /vineyard/{id}/harvest/new` | `handleHarvestNewPOST` | `hx-post` | Create harvest record |
| 26 | `GET /vineyard/{id}/harvest/{recordId}/edit` | `handleHarvestEditGET` | None (full page) | Harvest edit form |
| 27 | `POST /vineyard/{id}/harvest/{recordId}/edit` | `handleHarvestEditPOST` | `hx-post` | Update harvest record |
| 28 | `GET /vineyard/{id}/benchmark` | `handleBenchmarkGET` | None (full page) | Benchmark tables |
| 29 | `GET /vineyard/{id}/settings` | `handleSettingsGET` | None (full page) | Settings (owner only) |
| 30 | `POST /vineyard/{id}/settings` | `handleSettingsPOST` | `hx-post` | Vineyard update / invite / remove member / password change |
| 31 | `GET /onboard` | `handleOnboardGET` | None (full page) | Vineyard registration |
| 32 | `POST /onboard` | `handleOnboardPOST` | `hx-post` | Create vineyard |
| 33 | `GET /privacy` | `handlePrivacyGET` | None (full page) | Static page |
| 34 | `GET /terms` | `handleTermsGET` | None (full page) | Static page |
| 35 | `GET /api/varieties/search` | `handleVarietySearchGET` | `hx-get` | JSON, consumed by Alpine.js |
| 36 | `GET /api/account/export` | `handleAccountExportGET` | `hx-get` | JSON download |
| 37 | `POST /api/account/delete` | `handleAccountDeletePOST` | `hx-post` | Delete account |
| 38 | `POST /api/geo/reverse` | `handleGeoReversePOST` | `hx-post` | JSON, consumed by Alpine.js |
| 39 | `GET /health` | `handleHealthGET` | None | Diagnostic endpoint |

---

## Appendix B: Tailwind Color Mapping (from SvelteKit inline styles)

| Svelte Style | Tailwind Class |
|-------------|---------------|
| `color: #2d6a2d` (green brand) | `text-[#2d6a2d]` / `bg-[#2d6a2d]` |
| `background: #e8f5e9` (green light) | `bg-green-50` |
| `background: #ffebee` (red light) | `bg-red-50` |
| `background: #fff3e0` (amber light) | `bg-amber-50` |
| `background: #e3f2fd` (blue light) | `bg-blue-50` |
| `background: #f5f5f5` (gray light) | `bg-gray-100` |
| `background: #f9f9f9` (gray lighter) | `bg-gray-50` |
| `border: 1px solid #ddd` | `border border-gray-300` |
| `border-radius: 4px` | `rounded` |
| `font-family: sans-serif` | `font-sans` |
| `padding: 0.6rem` | `p-2` |
| `padding: 0.75rem` | `p-3` |
| `padding: 1rem` | `p-4` |
| `max-width: 400px` | `max-w-md` |
| `max-width: 420px` | `max-w-md` |
| `max-width: 600px` | `max-w-2xl` |
| `max-width: 700px` | `max-w-3xl` |
| `max-width: 900px` | `max-w-7xl` |
| `margin: 5vh auto` | `mt-24 mx-auto` |
| `margin: 10vh auto` | `mt-40 mx-auto` |
| `margin: 15vh auto` | `mt-60 mx-auto` |

---

## Appendix C: Go Middleware Stack

```go
func (s *Server) setupMiddleware() {
    // 1. Logging
    s.router.Use(middleware.Logger)
    
    // 2. Recovery
    s.router.Use(middleware.Recover)
    
    // 3. Session (cookie-based)
    s.router.Use(middleware.Session(s.sessionStore))
    
    // 4. Auth required (skip for public routes)
    s.router.Use(middleware.AuthRequired)
    
    // 5. CSRF protection
    s.router.Use(middleware.CSRF(s.csrfStore))
    
    // 6. Vineyard membership check (for /vineyard/* routes)
    s.router.Use(middleware.VineyardAccess)
    
    // 7. Serve static files
    s.router.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
    
    // 8. Template handlers
    s.router.HandleFunc("GET /", s.handleLanding)
    // ... all routes
}
```
