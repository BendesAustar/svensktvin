# Svenskt Vin — Template Architecture & UX Review

**Date:** 2026-06-15
**Author:** DesignBrain (Cinerarium)
**Scope:** Template loading, routing, inheritance, UX flows, accessibility

---

## Executive Summary

The Svenskt Vin template system has **critical structural defects** that prevent any vineyard page from rendering. The system suffers from two conflicting template paradigms (standalone vs. inheritance) applied inconsistently across the file tree, combined with name mismatches between `{{define}}` declarations and `{{template}}` calls. Additionally, the root route `/` is unregistered, and the custom CSS file referenced by all templates is missing.

**Severity Assessment:**
| Issue | Severity | Impact |
|-------|----------|--------|
| Template inheritance name mismatch (`base` vs `base.html`) | **CRITICAL** | No vineyard/dashboard/benchmark page renders |
| No root route `/` handler | **HIGH** | `/` returns 404 |
| Missing `static/css/app.css` | **HIGH** | Custom styles never load |
| Nested `{{define}}` blocks in vineyard templates | **CRITICAL** | Content blocks never execute |
| Duplicate HTML boilerplate across all templates | **MEDIUM** | Massive maintainability issue |
| Nav template requires vineyard-scoped context | **MEDIUM** | Landing page nav breaks for unauthenticated users |

---

## 1. Current State Analysis

### 1.1 Template Loading Architecture

**File:** `cmd/web/main.go` — `loadTemplatesFromDir()`

```go
func loadTemplatesFromDir(dir string, funcMap template.FuncMap) (*template.Template, error) {
    var paths []string
    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() && strings.HasSuffix(path, ".html") {
            paths = append(paths, path)
        }
        return nil
    })
    // ...
    tmpl := template.New("")
    tmpl = tmpl.Funcs(funcMap)
    _, err = tmpl.ParseFiles(paths...)
    // ...
}
```

**How Go templates work here:**
1. `filepath.Walk` discovers all `.html` files under `internal/templates/`
2. `ParseFiles` parses ALL files into a **single** `*template.Template` value (one template set)
3. Each file's `{{define "name"}}` registers a named template in the set
4. The file's path (minus extension) also becomes an implicit template name

### 1.2 Template Name Resolution

When Go parses a file, it creates **two** template names:
- The **implicit** name from the file path (e.g., `vineyard/index.html` → `vineyard/index.html`)
- The **explicit** name from `{{define "name"}}` (if present)

**Critical detail:** If a file has a `{{define}}` block, Go names the *outermost* template as the filename (without extension), and the `{{define}}` content as the named template. If there's no `{{define}}`, the entire file content becomes the template.

### 1.3 Define Name Audit

| File | `{{define "..."}}` name | Handler calls as | Match? |
|------|------------------------|------------------|--------|
| `base.html` | `"base"` | — (never called directly) | — |
| `nav.html` | `"nav"` | `{{template "nav.html" .}}` | ❌ MISMATCH |
| `flash.html` | `"flash"` | `{{template "flash.html" .}}` | ❌ MISMATCH |
| `form-errors.html` | `"form-errors"` | — (never included) | — |
| `invite/success.html` | `"invite/success.html"` | — (never called) | — |
| `auth/login.html` | `"auth/login.html"` | `"auth/login.html"` | ✅ OK |
| `auth/register.html` | `"auth/register.html"` | `"auth/register.html"` | ✅ OK |
| `auth/forgot-password.html` | `"auth/forgot-password.html"` | `"auth/forgot-password.html"` | ✅ OK |
| `auth/set-password.html` | `"auth/set-password.html"` | `"auth/set-password.html"` | ✅ OK |
| `invite/confirm.html` | `"invite/confirm.html"` | `"invite/confirm.html"` | ✅ OK |
| `vineyard/index.html` | `"vineyard/index.html"` | `"vineyard/index.html"` | ✅ OK |
| `vineyard/dashboard.html` | `"vineyard/dashboard.html"` | `"vineyard/dashboard.html"` | ✅ OK |
| `vineyard/benchmark.html` | `"vineyard/benchmark.html"` | `"vineyard/benchmark.html"` | ✅ OK |
| `vineyard/blocks/new.html` | `"vineyard/blocks/new.html"` | — (likely OK) | ✅ OK |
| `vineyard/blocks/edit.html` | `"vineyard/blocks/edit.html"` | — (likely OK) | ✅ OK |
| `vineyard/harvest/new.html` | `"vineyard/harvest/new.html"` | — (likely OK) | ✅ OK |
| `vineyard/harvest/edit.html` | `"vineyard/harvest/edit.html"` | — (likely OK) | ✅ OK |

**Two mismatches found:** `nav.html` and `flash.html` — both use simple `{{define}}` names but are called with `.html` suffix via `{{template}}`.

### 1.4 Two Competing Template Patterns

The codebase uses **two fundamentally incompatible** patterns:

**Pattern A — Standalone (auth templates):**
Each file is a complete, self-contained HTML document with its own `<!DOCTYPE html>`, `<head>`, scripts, body. The handler executes the template directly:
```go
renderTemplate(w, tmpl, "auth/login.html", data)
```
→ `ExecuteTemplate("auth/login.html")` finds the named template and renders the full document. ✅ Works.

**Pattern B — Inheritance (vineyard templates):**
Each file attempts to define BOTH a complete layout wrapper AND content blocks, expecting Go to compose them:
```html
{{define "vineyard/index.html"}}
<!DOCTYPE html>
...
{{template "nav.html" .}}       ← tries to call "nav.html" but define is "nav"
{{template "flash.html" .}}     ← tries to call "flash.html" but define is "flash"
{{block "content" .}}{{end}}
...
{{define "content"}}             ← registers "content" at parse time
  ... actual content ...
{{end}}
{{end}}
```

**The inheritance pattern is fundamentally broken** because:

1. **Name mismatches** — `nav.html`/`flash.html` define simple names, called with `.html` suffix
2. **Nested `{{define}}` blocks don't work as expected** — the vineyard templates nest `{{define "title"}}` and `{{define "content"}}` inside `{{define "vineyard/index.html"}}`. In Go templates, `{{define}}` is a **parse-time registration**, not a runtime include. All `{{define}}` blocks in a file are registered when the file is parsed, regardless of nesting context. The blocks registered at parse-time are never automatically executed — only `{{template}}` or `{{block}}` calls can invoke them.
3. **The content between the first `{{end}}` (closing `</html>`) and the second `{{define "content"}}` is dead code** — it appears inside the outer `{{define}}` but outside any `{{template}}`/`{{block}}` invocation. Go will register the `{{define}}` blocks but won't execute the intermediate HTML.

### 1.5 What Actually Gets Rendered

When `ExecuteTemplate("vineyard/index.html")` is called:
1. Go finds the named template `"vineyard/index.html"`
2. It renders the outermost template content (everything in the `{{define "vineyard/index.html"}}` block)
3. `{{template "nav.html" .}}` → **fails silently** (no template named `"nav.html"`; the defined name is `"nav"`)
4. `{{template "flash.html" .}}` → **fails silently** (same issue)
5. `{{block "content" .}}{{end}}` → renders empty (no `content` key populated in the block)
6. The HTML between `</html>` and `{{define "content"}}` is part of the template definition but is **dead text** — it's never rendered by `{{template}}`/`{{block}}`
7. The `{{define "content"}}` block is **registered at parse time** but **never invoked**
8. Result: User sees a page with the HTML wrapper but NO nav, NO flash messages, NO page content

### 1.6 Route Handler Audit

| Route | Handler | Auth Required | Template Called | Notes |
|-------|---------|---------------|-----------------|-------|
| `GET /health` | inline | No | — | Works |
| `GET /login` | `authHandler.HandleLoginGET` | No | `auth/login.html` | ✅ Works (standalone) |
| `POST /login` | `authHandler.HandleLoginPOST` | No | `auth/login.html` | ✅ Works (standalone) |
| `POST /logout` | `authHandler.HandleLogoutPOST` | Yes | `auth/login.html` via HX-Redirect | Works |
| `GET /register` | `authHandler.HandleRegisterGET` | No | `auth/register.html` | ✅ Works (standalone) |
| `POST /register` | `authHandler.HandleRegisterPOST` | No | `auth/register.html` | ✅ Works (standalone) |
| `GET /auth/forgot-password` | `authHandler.HandleForgotPasswordGET` | No | `auth/forgot-password.html` | ✅ Works (standalone) |
| `POST /auth/forgot-password` | `authHandler.HandleForgotPasswordPOST` | No | `auth/forgot-password.html` | ✅ Works (standalone) |
| `GET /auth/set-password` | `authHandler.HandleSetPasswordGET` | No | `auth/set-password.html` | ✅ Works (standalone) |
| `POST /auth/set-password` | `authHandler.HandleSetPasswordPOST` | No | `auth/set-password.html` | ✅ Works (standalone) |
| `GET /invite/confirm` | `authHandler.HandleInviteConfirmGET` | No | `invite/confirm.html` | ✅ Works (standalone) |
| `POST /invite/confirm` | `authHandler.HandleInviteConfirmPOST` | No | `invite/confirm.html` | Works |
| **`GET /`** | **NOT REGISTERED** | — | — | **🔴 404** |
| `GET /vineyard` | `vineyardHandler.HandleLandingGET` | Optional | `vineyard/index.html` | 🔴 Broken (inheritance) |
| `GET /vineyard/` | `vineyardHandler.HandleLandingGET` | Optional | `vineyard/index.html` | 🔴 Broken (inheritance) |
| `GET /vineyard/{id}` | `vineyardHandler.HandleVineyardGET` | Yes | `vineyard/dashboard.html` | 🔴 Broken (inheritance) |
| `GET /vineyard/{id}/benchmark` | `vineyardHandler.HandleBenchmarkGET` | Yes | `vineyard/benchmark.html` | 🔴 Broken (inheritance) |
| `POST /vineyard/{id}/` | `vineyardHandler.HandleVineyardPOST` | Yes | — (delegates) | Delegates to harvest/block handlers |
| `GET /static/*` | FileServer | No | — | Works (but app.css missing) |
| `GET /api/varieties/search` | `varietySearchHandler` | No | — | JSON API |
| `POST /api/geo/reverse` | `geoReverseHandler` | No | — | JSON API |

---

## 2. Template Architecture Recommendation

### 2.1 Recommended Pattern: Base Template + Content Blocks

Go templates support a **clean base template pattern**: one file defines the layout, and other templates define content blocks that the base template includes.

**File structure:**
```
internal/templates/
├── base.html              # Layout wrapper (no {{define}} — entire file IS the template)
├── nav.html               # {{define "nav"}} — called from base.html
├── flash.html             # {{define "flash"}} — called from base.html
├── form-errors.html       # {{define "form-errors"}} — standalone utility
├── auth/
│   ├── login.html         # Standalone (complete HTML) — login is special (no nav needed)
│   ├── register.html      # Standalone
│   ├── forgot-password.html  # Standalone
│   └── set-password.html      # Standalone
└── vineyard/
    ├── index.html         # {{define "content"}} + {{define "title"}} + {{define "head"}} + {{define "scripts"}}
    ├── dashboard.html     # Same block pattern
    ├── benchmark.html     # Same block pattern
    ├── blocks/
    │   ├── new.html       # Same block pattern
    │   └── edit.html      # Same block pattern
    └── harvest/
        ├── new.html       # Same block pattern
        └── edit.html      # Same block pattern
```

### 2.2 `base.html` (The Layout)

```html
<!DOCTYPE html>
<html lang="sv" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}{{.Title}}{{end}}</title>

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-..." crossorigin="anonymous"></script>
    <!-- Alpine.js -->
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.14.8/dist/cdn.min.js"></script>
    <!-- Tailwind CSS -->
    <script src="https://cdn.tailwindcss.com"></script>
    <!-- Custom styles -->
    <link rel="stylesheet" href="/static/css/app.css">

    {{block "head" .}}{{end}}
</head>
<body class="min-h-full bg-gray-50 font-sans text-gray-900">

    {{template "nav" .}}
    <!-- or conditional nav -->
    {{if .User}}{{template "nav" .}}{{else}}{{include "logged-out-header"}}{{end}}

    <main class="max-w-7xl mx-auto px-4">
        {{template "flash" .}}
        {{block "content" .}}{{end}}
    </main>

    {{block "scripts" .}}{{end}}
</body>
</html>
```

**Key change:** `base.html` should NOT have a `{{define "base"}}` wrapper. It should be a bare template — the entire file IS the template. Then `ExecuteTemplate` is called as:
```go
tmpl.ExecuteTemplate(w, "base", data)  // or just tmpl.Execute(w, data) if "base" is the name
```

But wait — Go's `ParseFiles` names the template by the file path. So `base.html` would be named `"base"`. The named template call `{{template "base" .}}` from inside `vineyard/index.html` would need to match.

Actually, the **cleanest approach** for Go templates is:

**Approach 1: Single base template, execute by name**
- `base.html` has NO `{{define}}` — it's the entire template named `"base"`
- Content templates only contain `{{define "title"}}`, `{{define "content"}}`, `{{define "scripts"}}`, `{{define "head"}}` blocks
- Handler calls:
  ```go
  // First parse the base
  baseTmpl := template.Must(template.New("base").ParseFiles("internal/templates/base.html"))
  // Then parse content templates (which register blocks)
  // Finally execute base with blocks populated
  baseTmpl.ExecuteTemplate(w, "base", data)
  ```

**Approach 2: All-in-one ParseFiles call (simpler, what the current code does)**
- `base.html` has NO `{{define}}` — the entire file is the `"base"` template
- Content files register `{{define "content"}}` etc. at parse time
- Handler calls `tmpl.ExecuteTemplate(w, "base", data)` — Go resolves blocks at execution time
- Content files themselves are NEVER directly executed; they only register blocks

**Approach 3: Standalone per-page (simplest for maintenance)**
- Every page is a complete, self-contained HTML document
- Use Go's `{{template "nav.html" .}}` to include partials
- Each file defines itself as `{{define "path/to/file.html"}}`
- Handler calls `renderTemplate(w, tmpl, "path/to/file.html", data)`
- This is what auth templates already do — extend this pattern to ALL pages

### 2.3 Recommended Decision: Approach 3 (Standalone + Partials)

**Why:**
1. **Simplest to understand** — each file is a complete, viewable page
2. **No parse-time complexity** — no block resolution confusion
3. **Easy to debug** — open the file in a browser, see the page
4. **Consistent** — auth templates already work this way
5. **HTMX-friendly** — HTMX replaces `outerHTML` of elements; standalone pages integrate naturally
6. **Partial inclusion** — `nav.html`, `flash.html` are included via `{{template}}` calls inside the standalone page

**Changes needed to partial files:**

Fix `nav.html`:
```html
{{define "nav"}}
<nav class="...">{{/* ... */}}</nav>
{{end}}
```

Fix `flash.html`:
```html
{{define "flash"}}
{{range .Flashes}}{{/* ... */}}{{end}}
{{end}}
```

These are fine as-is — the issue is only in how they're CALLED. The handler uses `{{template "nav.html" .}}` but the define name is `"nav"`. Fix: change calls to `{{template "nav" .}}` AND `{{template "flash" .}}`.

### 2.4 Fix Summary Table

| File | Issue | Fix |
|------|-------|-----|
| `base.html` | Has `{{define "base"}}` wrapper — not needed for standalone approach | **Remove** `{{define "base"}}`/`{{end}}`, OR **remove** file entirely and move content inline to each page |
| `nav.html` | Defined as `"nav"`, called as `"nav.html"` | Change all `{{template "nav.html"}}` → `{{template "nav"}}` |
| `flash.html` | Defined as `"flash"`, called as `"flash.html"` | Change all `{{template "flash.html"}}` → `{{template "flash"}}` |
| `vineyard/index.html` | Uses broken inheritance + standalone hybrid | Convert to standalone: remove block nesting, put full HTML + content inline |
| `vineyard/dashboard.html` | Same inheritance issue | Convert to standalone |
| `vineyard/benchmark.html` | Same inheritance issue | Convert to standalone |
| `vineyard/blocks/new.html` | Same inheritance issue | Convert to standalone |
| `vineyard/blocks/edit.html` | Same inheritance issue | Convert to standalone |
| `vineyard/harvest/new.html` | Same inheritance issue | Convert to standalone |
| `vineyard/harvest/edit.html` | Same inheritance issue | Convert to standalone |
| `route "/"` | No handler registered | Add `GET /` handler pointing to `vineyardHandler.HandleLandingGET` |
| `static/css/app.css` | File doesn't exist | Create minimal CSS file or remove reference |

---

## 3. UX Flow Completeness

### 3.1 User Journey: Landing → Login → Dashboard

| Step | Route | Status |
|------|-------|--------|
| 1. Visit home `/` | `GET /` | 🔴 No handler — 404 |
| 2. Click "Logga in" | `GET /login` | ✅ Works (standalone) |
| 3. Enter credentials, submit | `POST /login` | ✅ Works |
| 4. Redirect to vineyard | `GET /vineyard` | 🔴 Broken (inheritance + nav mismatch) |
| 5. Single vineyard → dashboard | `GET /vineyard/{id}` | 🔴 Broken (inheritance) |

### 3.2 User Journey: Landing → Register → Vineyard Join

| Step | Route | Status |
|------|-------|--------|
| 1. Visit `/register?invite=xxx` | `GET /register` | ✅ Works |
| 2. Submit registration | `POST /register` | ✅ Works |
| 3. Redirect to vineyard | `GET /vineyard` | 🔴 Broken |

### 3.3 User Journey: Forgot Password

| Step | Route | Status |
|------|-------|--------|
| 1. Click "Glömt lösenord?" | `GET /auth/forgot-password` | ✅ Works |
| 2. Submit email | `POST /auth/forgot-password` | ✅ Works |
| 3. Click magic link in email | `GET /auth/set-password?token=xxx` | ✅ Works |
| 4. Set new password | `POST /auth/set-password` | ✅ Works |
| 5. Redirect to vineyard | `GET /vineyard` | 🔴 Broken |

### 3.4 Dashboard → Block Creation → Harvest Logging

| Step | Route | Status |
|------|-------|--------|
| 1. Click "+ Nytt block" | `GET /vineyard/{id}/blocks/new` | 🔴 Broken (inheritance) |
| 2. Submit block form | `POST /vineyard/{id}/blocks/new` | Likely works (handler returns 302) |
| 3. Click "Skapa skörd" | `GET /vineyard/{id}/harvest/new` | 🔴 Broken (inheritance) |
| 4. Submit harvest | `POST /vineyard/{id}/harvest/new` | Likely works (handler returns 302) |
| 5. Click "Jämförelse" | `GET /vineyard/{id}/benchmark` | 🔴 Broken (inheritance) |

### 3.5 Nav Template Issues

The `nav.html` template has **two critical problems**:

1. **Name mismatch** (see above): `{{template "nav.html"}}` but defined as `{{define "nav"}}`

2. **Context dependency on `.Vineyard.ID`**: The nav template references `.Vineyard.ID` and `.Vineyard.Name` unconditionally. This works on dashboard/benchmark/blocks/harvest pages where `.Vineyard` is populated, but:
   - On the landing page (`/vineyard` with `.NoVineyards`), `.Vineyard` is NOT in the data map → template panic or blank output
   - The landing page is supposed to show a logged-out header (no nav), but the nav template is called unconditionally

3. **Owner-only settings link**: `.Role` is used with `{{if eq .Role "owner"}}` — this works when Role is set, but the landing page data doesn't include `Role` → potential nil pointer

### 3.6 Cookie Consent Bar

The cookie consent bar is included in BOTH `base.html` and all vineyard templates' layout wrapper. With the current duplicate-inheritance structure, it would render **twice** on any page that manages to load. After fixing to standalone pattern, it should exist in every standalone page OR be extracted to a partial.

---

## 4. Accessibility Audit

### 4.1 Missing Elements

| Issue | Impact | Severity |
|-------|--------|----------|
| No `<main>` role or `role="main"` | Screen readers can't quickly navigate | MEDIUM |
| No `aria-label` on nav | Screen readers don't know nav purpose | MEDIUM |
| Form labels use `for` but some inputs in nav have no label | Accessibility violation | LOW |
| Cookie consent bar has no keyboard dismiss alternative | Keyboard users can't close it | HIGH |
| No skip-to-content link | Keyboard navigation must tab through entire nav | MEDIUM |
| Color-only status indicators (emoji) | Colorblind users may miss status | LOW |
| No focus indicators on Alpine.js dynamic elements | Keyboard focus visibility | MEDIUM |
| `<select>` for exposition has no `aria-label` | Screen reader doesn't know purpose | LOW |
| No `aria-live` regions for HTMX swaps | Dynamic content changes are silent to screen readers | HIGH |

### 4.2 Viewport and Mobile

| Check | Status |
|-------|--------|
| `<meta name="viewport" content="width=device-width, initial-scale=1.0">` | ✅ Present in all templates |
| Tailwind responsive classes (`md:`, `lg:`) | ✅ Used in grid layouts |
| Touch targets (min 44px) | ✅ Buttons are `p-3` (~48px) |
| Mobile nav | ❌ Desktop-only nav; no hamburger menu for mobile |
| Cookie bar mobile | ✅ `flex-wrap gap-3` allows wrapping |

---

## 5. Specific Issues to Fix (Prioritized)

### P0 — Must Fix Before Launch

1. **Add root route handler**
   ```go
   // In main.go, after the /health route:
   mux.HandleFunc("GET /", vineyardHandler.HandleLandingGET(templates))
   ```

2. **Fix template name mismatches**
   - Change all `{{template "nav.html" .}}` → `{{template "nav" .}}`
   - Change all `{{template "flash.html" .}}` → `{{template "flash" .}}`

3. **Convert vineyard templates to standalone pattern**
   - Each vineyard template should be a complete HTML document
   - Content goes directly in the template (no `{{define "content"}}` nesting)
   - Remove the broken block-based inheritance
   - Example for `vineyard/index.html`:
     ```html
     {{define "vineyard/index.html"}}
     <!DOCTYPE html>
     <html lang="sv" class="h-full">
     <head>
         ...
         <title>{{block "title" .}}{{.Title}}{{end}}</title>
         ...
     </head>
     <body>
         {{if .User}}
             {{template "nav" .}}
         {{else}}
             <!-- logged-out header -->
         {{end}}
         <main class="max-w-7xl mx-auto px-4">
             {{template "flash" .}}
             <!-- inline page content here -->
         </main>
         {{block "scripts" .}}{{end}}
     </body>
     </html>
     {{end}}
     ```

4. **Create `static/css/app.css`** — even if minimal (just resets, or move inline styles)

5. **Fix nav template context safety**
   ```html
   {{if .User}}
   {{template "nav" .}}
   {{else}}
   <header class="bg-gray-100 border-b border-gray-300">
       <!-- logged-out header -->
   </header>
   {{end}}
   ```

### P1 — Should Fix Before Beta

6. **Extract duplicate HTML boilerplate** to a helper or base template
   - All vineyard templates have identical `<head>` content (HTMX, Alpine, Tailwind, custom CSS)
   - This is ~35 lines of duplication across 8+ files

7. **Fix cookie consent bar duplication** — currently in both `base.html` and all vineyard templates

8. **Add cookie consent bar to auth templates** — currently missing from `login.html`, `register.html`, etc.

9. **Add `{{block "title" .}}` default to standalone auth templates** — they have hardcoded titles instead of using the block pattern

### P2 — Nice to Have

10. **Add skip-to-content link** for keyboard users
11. **Add `aria-live="polite"` to flash message container** for HTMX updates
12. **Add mobile hamburger nav** for small screens
13. **Add `x-cloak` to Alpine.js dropdowns** — verify all Alpine components have it
14. **Add Open Graph meta tags** for social sharing
15. **Add favicon**
16. **Consistent page title pattern**: `"Page — Svenskt Vin"` (already done, but ensure all templates use it)

---

## 6. Migration Plan

### Phase 1: Quick Fixes (1-2 hours)

1. Add `GET /` route → `HandleLandingGET`
2. Fix `nav.html`/`flash.html` template name references
3. Create empty `static/css/app.css`
4. Add context safety to nav (logged-out header fallback)

### Phase 2: Template Restructure (4-6 hours)

5. Convert all vineyard templates from inheritance to standalone
6. Consolidate HTML boilerplate into a shared include or use Go's template parsing order
7. Remove `base.html` (or convert it to a true base with no `{{define}}` wrapper)
8. Add cookie consent bar to auth templates
9. Add `{{block "title"}}` pattern to auth templates

### Phase 3: Polish (2-3 hours)

10. Add accessibility improvements
11. Add mobile nav
12. Add Open Graph tags, favicon
13. Test all user journeys end-to-end

---

## 7. Appendix: File-by-File Template Structure Audit

### `base.html`
- **Define name:** `"base"`
- **Structure:** Full HTML layout with `{{define "base"}}` wrapper
- **Problem:** Wrapper means Go names this template `"base"`, but vineyard templates try to call `{{template "base.html"}}` (which doesn't exist)
- **Fix:** Remove `{{define "base"}}`/`{{end}}` wrapper OR remove file entirely and inline boilerplate

### `nav.html`
- **Define name:** `"nav"`
- **Structure:** `<nav>` element only
- **Problem:** Called as `{{template "nav.html"}}` but defined as `"nav"`
- **Fix:** Change calls to `{{template "nav"}}`

### `flash.html`
- **Define name:** `"flash"`
- **Structure:** Flash message loop
- **Problem:** Called as `{{template "flash.html"}}` but defined as `"flash"`
- **Fix:** Change calls to `{{template "flash"}}`

### `form-errors.html`
- **Define name:** `"form-errors"`
- **Structure:** Error display partial
- **Problem:** Never actually included in any template
- **Note:** Content is inlined directly in each form template instead

### `auth/login.html`
- **Define name:** `"auth/login.html"`
- **Structure:** Complete standalone HTML document
- **Status:** ✅ Working correctly
- **Note:** Does NOT use inheritance — correct standalone pattern

### `auth/register.html`
- **Define name:** `"auth/register.html"`
- **Structure:** Complete standalone HTML document
- **Status:** ✅ Working correctly

### `auth/forgot-password.html`
- **Define name:** `"auth/forgot-password.html"`
- **Structure:** Complete standalone HTML document
- **Status:** ✅ Working correctly

### `auth/set-password.html`
- **Define name:** `"auth/set-password.html"`
- **Structure:** Complete standalone HTML document
- **Status:** ✅ Working correctly

### `invite/confirm.html`
- **Define name:** `"invite/confirm.html"`
- **Structure:** Complete standalone HTML document
- **Status:** ✅ Working correctly

### `invite/success.html`
- **Define name:** `"invite/success.html"`
- **Structure:** Complete standalone HTML document
- **Status:** Template exists but never called by any handler
- **Note:** Invite confirmation uses `HX-Redirect` to vineyard instead

### `vineyard/index.html`
- **Define name:** `"vineyard/index.html"`
- **Structure:** **BROKEN** — hybrid of standalone wrapper + inheritance blocks
- **Problem:** `{{define "content"}}` nested inside the wrapper, never executed
- **Fix:** Convert to standalone (see Phase 2 plan)

### `vineyard/dashboard.html`
- **Define name:** `"vineyard/dashboard.html"`
- **Structure:** **BROKEN** — same inheritance issue as index.html
- **Problem:** `{{template "nav.html"}}` fails, content block never executes
- **Fix:** Convert to standalone

### `vineyard/benchmark.html`
- **Define name:** `"vineyard/benchmark.html"`
- **Structure:** **BROKEN** — same inheritance issue
- **Fix:** Convert to standalone

### `vineyard/blocks/new.html`
- **Define name:** `"vineyard/blocks/new.html"`
- **Structure:** **BROKEN** — same inheritance issue
- **Fix:** Convert to standalone

### `vineyard/blocks/edit.html`
- **Define name:** `"vineyard/blocks/edit.html"`
- **Structure:** **BROKEN** — same inheritance issue
- **Fix:** Convert to standalone

### `vineyard/harvest/new.html`
- **Define name:** `"vineyard/harvest/new.html"`
- **Structure:** **BROKEN** — same inheritance issue
- **Fix:** Convert to standalone

### `vineyard/harvest/edit.html`
- **Define name:** `"vineyard/harvest/edit.html"`
- **Structure:** **BROKEN** — same inheritance issue
- **Fix:** Convert to standalone

