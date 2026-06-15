# SvensktVin — Template System Fix Plan

**Source:** EngineeringBrain analysis, validated against DesignBrain's review (`.pi/reviews/design-review-template-system.md`)
**Date:** 2026-06-15
**Approach:** DesignBrain Recommendation — Approach 3 (Standalone + Partials)

## Problem Summary

The vineyard page templates use a broken inheritance pattern (`{{template "base.html"}}` + nested `{{define "content"}}`) that Go's template engine doesn't support. This results in:
- No content rendering on vineyard pages
- Template name mismatches: `nav.html`/`flash.html` called with `.html` suffix but defined as simple names
- No root route `/` handler
- Missing `static/css/app.css`

## Root Cause Analysis

### Go Template Name Resolution

When `filepath.Walk` discovers `internal/templates/vineyard/index.html`, Go creates template names from file paths. A file with `{{define "vineyard/index.html"}}` registers that exact name. A file without `{{define}}` has an implicit name from its path.

### The Broken Pattern

```
vineyard/index.html:
  {{template "base.html" .}}   ← tries to include "base.html" (defined as "base")
  {{define "title"}}...{{end}} ← registered at parse time, never invoked
  {{define "content"}}...{{end}} ← registered at parse time, never invoked
```

`ExecuteTemplate("vineyard/index.html")` finds the named template, but `base.html` is named `"base"` not `"base.html"`, and the content blocks are never called.

### DesignBrain's Verdict

The template system needs to be unified. DesignBrain recommends **Approach 3: Standalone + Partials** — every page is a complete HTML document with Go partial includes (`{{template "nav"}}`). This is consistent with how auth templates already work.

## Implementation Plan

### Phase 1: Structural Fixes (cmd/web/main.go)

**T1: Add root route handler**

```go
// After the health route in main.go:
mux.HandleFunc("GET /", vineyardHandler.HandleLandingGET(templates))
```

**T2: Add `static/css/app.css`**

Create empty file: `touch static/css/app.css`

### Phase 2: Template Name Mismatches

**T3: Fix `nav.html` and `flash.html` calls**

Change ALL occurrences of `{{template "nav.html"}}` → `{{template "nav"}}`
Change ALL occurrences of `{{template "flash.html"}}` → `{{template "flash"}}`

Files to update:
- `vineyard/index.html`
- `vineyard/dashboard.html`
- `vineyard/benchmark.html`
- `vineyard/blocks/new.html`
- `vineyard/blocks/edit.html`
- `vineyard/harvest/new.html`
- `vineyard/harvest/edit.html`

### Phase 3: Convert Vineyard Templates to Standalone Pattern

**T4: Convert ALL 7 vineyard templates to standalone pattern**

For each file, the transformation is:
1. Add `{{define "vineyard/..."}}` at the very top
2. Replace `{{template "base.html" .}}` with the full HTML boilerplate (DOCTYPE, head, scripts, body start)
3. Change `{{define "title"}}{{.Title}}{{end}}` to `{{block "title" .}}{{.Title}}{{end}}`
4. Move content OUT of `{{define "content"}}...{{end}}` and into the main body flow
5. Change `{{template "nav.html"}}` → `{{template "nav"}}`
6. Change `{{template "flash.html"}}` → `{{template "flash"}}`
7. Add logged-out header fallback (instead of bare `{{template "nav"}}`)
8. Add cookie consent bar (currently only in `base.html`)
9. Close with `{{end}}` for the outer define

The boilerplate template to use for each file:

```html
{{define "TEMPLATE_NAME"}}
<!DOCTYPE html>
<html lang="sv" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}{{.Title}}{{end}}</title>
    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfMzorYmKGrM1lGbI0i6YcfnBDWTRfz94B0nTDnO2UcBZcDR40XyXGNq" crossorigin="anonymous"></script>
    <!-- Alpine.js -->
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.14.8/dist/cdn.min.js"></script>
    <!-- Tailwind CSS -->
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="/static/css/app.css">
    {{block "head" .}}{{end}}
</head>
<body class="min-h-full bg-gray-50 font-sans text-gray-900">
    {{if .User}}
        {{template "nav" .}}
    {{else}}
        <header class="bg-gray-100 border-b border-gray-300">
            <div class="max-w-7xl mx-auto px-4 py-2 flex justify-between items-center">
                <a href="/" class="no-underline font-semibold text-lg text-[#2d6a2d]">🍷 Svenskt Vin</a>
                <div class="flex gap-4">
                    <a href="/login" class="text-sm text-gray-500 hover:text-gray-700 no-underline">Logga in</a>
                    <a href="/register" class="text-sm text-gray-500 hover:text-gray-700 no-underline">Skapa konto</a>
                </div>
            </div>
        </header>
    {{end}}
    <main class="max-w-7xl mx-auto px-4">
        {{template "flash" .}}
        <!-- PAGE CONTENT (moved from {{define "content"}}) -->
    </main>
    {{if not .User}}
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
        <button @click="shown = false"
                class="px-4 py-2 bg-[#2d6a2d] text-white border-none rounded text-sm cursor-pointer font-semibold">
            Acceptera
        </button>
    </div>
    {{end}}
    {{block "scripts" .}}{{end}}
</body>
</html>
{{end}}
```

### Phase 4: Nav Template Context Safety

**T5: Fix nav.html to safely handle nil .Vineyard**

The nav template currently assumes `.Vineyard` is always set. Make it safe:
```html
{{define "nav"}}
<nav class="max-w-7xl mx-auto px-4 py-2 border-b border-gray-300 bg-white flex justify-between items-center">
    <div class="flex gap-1 items-center">
        <a href="/" class="no-underline font-semibold text-lg text-[#2d6a2d]">🍷 Svenskt Vin</a>
        {{if .Vineyard.ID}}
        <a href="/vineyard/{{.Vineyard.ID}}" class="px-3 py-1 text-sm rounded no-underline
            {{if eq .IsHome}}bg-gray-200 font-semibold{{end}} text-gray-600 hover:text-gray-900">
            {{.Vineyard.Name}}
        </a>
        {{end}}
        ...rest of nav...
    </div>
</nav>
{{end}}
```

### Phase 5: Update Config

**T6: Fix config defaults for local development**

Change the hardcoded `5434` port to `5432` in `internal/config/config.go`:
```go
// Before: "postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin"
// After:  "postgres://sv_app:sv_dev_pass@localhost:5432/svensktvin"
```

## Verification Plan

After all changes:

```bash
cd /home/neurograft/Techstack/svensktvin

# 1. Compile
go build ./...

# 2. No template name mismatches remain
grep -r 'template "nav.html"\|template "flash.html"\|template "base.html"' internal/templates/

# 3. All vineyard templates use standalone pattern
grep -l '{{define "vineyard/' internal/templates/vineyard/*.html internal/templates/vineyard/**/*.html

# 4. Start and test
DATABASE_URL="postgres://sv_app:sv_dev_pass@127.0.0.1:5432/svensktvin" \
  SESSION_SECRET="dev-secret-change-me" \
  APP_HOST="http://localhost:8090" \
  PORT="8090" \
  go run ./cmd/web/ &

# 5. Test root route
curl -s http://localhost:8090/ | grep -c "Välkommen till Svenskt Vin"

# 6. Test login page
curl -s http://localhost:8090/login | grep -c "Logga in"

# 7. Kill
kill %1
```

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Go template parse errors | `go build ./...` catches all template syntax errors |
| Content loss during conversion | Each template's `{{define "content"}}` block is extracted and placed inline — use git diff to verify |
| Template name collision | All templates use explicit `{{define "vineyard/..."}}` paths — no implicit name collisions |
| Partial template issues | `nav` and `flash` are simple partials — verified working in auth templates |
| Config change breaks Cinerarium | Only the hardcoded default changes; env vars take precedence |

## Files to Change

| File | Change |
|------|--------|
| `cmd/web/main.go` | Add root route handler |
| `static/css/app.css` | Create new empty file |
| `internal/templates/vineyard/index.html` | Convert to standalone |
| `internal/templates/vineyard/dashboard.html` | Convert to standalone |
| `internal/templates/vineyard/benchmark.html` | Convert to standalone |
| `internal/templates/vineyard/blocks/new.html` | Convert to standalone |
| `internal/templates/vineyard/blocks/edit.html` | Convert to standalone |
| `internal/templates/vineyard/harvest/new.html` | Convert to standalone |
| `internal/templates/vineyard/harvest/edit.html` | Convert to standalone |
| `internal/templates/nav.html` | Add nil-safe .Vineyard check |
| `internal/templates/flash.html` | No changes needed (partial, no calls to fix) |
| `internal/templates/base.html` | No changes needed (deprecated but harmless) |
| `internal/config/config.go` | Fix default PG port 5434 → 5432 |
