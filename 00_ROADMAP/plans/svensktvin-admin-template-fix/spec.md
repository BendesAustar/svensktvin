# Admin Template Fix — Standalone → Define Pattern

## Problem

5 admin templates are standalone files (no `{{define}}` wrapper):
- `internal/templates/admin/dashboard.html`
- `internal/templates/admin/users.html`
- `internal/templates/admin/user_detail.html`
- `internal/templates/admin/forbidden.html`
- `internal/templates/admin/invite_result.html`

The project's `renderTemplate()` function expects named templates via `{{define "path/to/file.html"}}`. This causes `template execute err="html/template: "admin/dashboard.html" is undefined"` at runtime.

All other templates in the project follow the `{{define}}` pattern (auth/*, vineyard/*, invite/*).

## Fix

Add matching `{{define}}` wrappers to all 5 templates:
- `dashboard.html` → `{{define "admin/dashboard.html"}}...{{end}}`
- `users.html` → `{{define "admin/users.html"}}...{{end}}`
- `user_detail.html` → `{{define "admin/user_detail.html"}}...{{end}}`
- `forbidden.html` → `{{define "admin/forbidden.html"}}...{{end}}`
- `invite_result.html` → `{{define "admin/invite_result.html"}}...{{end}}`

Note: `login.html` and `layout.html` already have correct `{{define}}` wrappers.

## Verification

After fix:
- `go build ./...` compiles clean
- `curl -s http://localhost:8090/admin` (authenticated) renders dashboard
- `curl -s http://localhost:8090/admin/users` renders user list
- No "template is undefined" errors in server logs
