# Svenskt Vin — Go-Live Plan

**Date:** 2026-06-12
**Author:** Implementer
**Status:** Draft — awaiting review

---

## Executive Summary

This plan covers the production deployment of **Svenskt Vin**, a SvelteKit-based vineyard data collection platform for Swedish viticulture. The application uses a single PostgreSQL instance (PostGIS 16), Node.js runtime (SvelteKit adapter-node), and SMTP for magic-link authentication.

---

## 0. Pre-Flight Checklist

| # | Item | Status | Notes |
|---|------|--------|-------|
| 0.1 | Port alignment | ✅ | Dev uses 5434; prod uses 5434. See §1.2 for rationale. |
| 0.2 | SMTP provider selected & credentials available | 🔲 | **Blocker for magic-link login** |
| 0.3 | Production database credentials generated | 🔲 | Separate from `sv_dev_pass` |
| 0.4 | TLS certificate ready for domain | 🔲 | Required for `secure: true` cookies |
| 0.5 | DNS record configured | 🔲 | Points to production host |
| 0.6 | Docker Compose hardened for production | 🔲 | Passwords via env var, not defaults |

---

## 1. Infrastructure

### 1.1 Hosting Options

| Option | Pros | Cons | Recommendation |
|--------|------|------|----------------|
| **Cinerarium VPS** (shared infra) | Shared networking, existing monitoring | Resource contention, mixed tenants | Acceptable for MVP |
| **Dedicated VPS** (e.g., Hetzner, DigitalOcean) | Isolated resources, simple Docker Compose deployment | Cost (~$5-10/mo) | Recommended for production |
| **PaaS** (Fly.io, Railway, Render) | Zero ops, built-in TLS, auto-deploy | Vendor lock-in, less control | Consider if ops overhead is a concern |

### 1.2 Docker Compose — Production Configuration

**Port rationale:** Svenskt Vin uses host port **5434** (not 5433). Cinerarium's PostgreSQL instance already owns 5433 on CORE. Both services run on the same host during development, so Svenskt Vin must use a separate port to avoid a bind conflict. On a dedicated production VPS, port 5434 is used for consistency — no collision exists there, but keeping the same port simplifies the config surface.

**Current state:** Single `db` service, port 5434 mapped. No app service defined.

**Required additions:**

```yaml
services:
  db:
    image: postgis/postgis:16-3.4
    environment:
      POSTGRES_DB: svensktvin
      POSTGRES_USER: sv_app
      POSTGRES_PASSWORD: ${PG_PASSWORD}   # MUST NOT use default
    ports:
      - "5434:5432"
    volumes:
      - sv_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sv_app -d svensktvin"]
      interval: 10s
      timeout: 5s
      retries: 10
    restart: unless-stopped

  app:
    build: .
    ports:
      - "3000:3000"    # Node adapter default
    environment:
      DATABASE_URL: postgres://sv_app:${PG_PASSWORD}@db:5432/svensktvin
      APP_HOST: https://svensktvin.se    # or your domain
      NODE_ENV: production
      SESSION_COOKIE_NAME: sv_session
      SMTP_HOST: ${SMTP_HOST}
      SMTP_PORT: ${SMTP_PORT}
      SMTP_USER: ${SMTP_USER}
      SMTP_PASS: ${SMTP_PASS}
      SMTP_FROM: ${SMTP_FROM}
    depends_on:
      db:
        condition: service_healthy
    restart: unless-stopped

volumes:
  sv_db_data:
```

### 1.3 Nginx Reverse Proxy (Optional but Recommended)

```nginx
server {
    listen 80;
    server_name svensktvin.se www.svensktvin.se;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name svensktvin.se www.svensktvin.se;

    ssl_certificate     /etc/letsencrypt/live/svensktvin.se/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/svensktvin.se/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 1.4 SSL with Let's Encrypt

```bash
apt install certbot python3-certbot-nginx
certbot --nginx -d svensktvin.se -d www.svensktvin.se
# Auto-renewal is configured by certbot
```

---

## 2. Database

### 2.1 Production Database Setup

```bash
# On production host
docker compose up -d db
sleep 15  # wait for healthcheck

# Apply migrations
DATABASE_URL=postgres://sv_app:<prod_password>@localhost:5434/svensktvin \
  ./scripts/migrate.sh

# Run verification
DATABASE_URL=postgres://sv_app:<prod_password>@localhost:5434/svensktvin \
  ./scripts/test.sh
```

**⚠️ Do NOT run `seed.sh` on production** — seeds are for development only. Real vineyard data is entered through the app by users.

### 2.2 Seed Data for Admin Testing

If admin accounts need pre-provisioning, use raw SQL instead of the seed script:

```sql
-- Create initial admin user (magic-link auth — no password column)
INSERT INTO users (email, name, is_admin, active)
VALUES ('admin@svensktvin.se', 'Admin', true, true);
```

### 2.3 Backup Strategy

Add to `docker-compose.yml` or cron:

```bash
# crontab -e — daily at 2am
0 2 * * * docker exec sv_db pg_dump -U sv_app svensktvin | gzip > /backups/svensktvin_$(date +\%Y\%m\%d).sql.gz
```

Retain 30 days of backups. Test restoration monthly.

---

## 3. Environment Variables

### 3.1 Required Variables

| Variable | Required | Production Value | Notes |
|----------|----------|------------------|-------|
| `DATABASE_URL` | Yes | `postgres://sv_app:<password>@db:5432/svensktvin` | Internal Docker network hostname |
| `NODE_ENV` | Yes | `production` | Enables production mode for cookies, error handling |
| `APP_HOST` | Yes | `https://svensktvin.se` | Used in magic-link URLs |
| `SESSION_COOKIE_NAME` | Yes | `sv_session` | Application-specific name |

### 3.2 SMTP Variables

| Variable | Required | Example |
|----------|----------|---------|
| `SMTP_HOST` | **Yes** | `smtp.smtp2go.com` |
| `SMTP_PORT` | Yes | `587` |
| `SMTP_USER` | **Yes** | `your_api_key` |
| `SMTP_PASS` | **Yes** | `your_api_secret` |
| `SMTP_FROM` | Yes | `noreply@svensktvin.se` |

**Recommended providers:**
- **SMTP2Go** — Swedish-friendly, free tier (500 emails/mo), simple API-key auth
- **Mailgun** — $0.008/email, robust API, GDPR compliant
- **Resend** — Developer-friendly, free tier (3000 emails/mo)

**⚠️ Magic-link login is non-functional without SMTP configured.** This is the critical pre-launch blocker.

### 3.3 .env.example Update

The current `.env.example` needs updating to reflect production conventions:

```bash
DATABASE_URL=postgres://sv_app:CHANGE_ME@localhost:5434/svensktvin
APP_HOST=http://localhost:5173
SESSION_COOKIE_NAME=sv_session
NODE_ENV=development
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@svensktvin.se
```

---

## 4. Application Build & Deployment

### 4.1 Pre-deployment Fixes (Before Go-Live)

| # | Fix | Priority | Status |
|---|-----|----------|--------|
| 4.1.1 | Port alignment — scripts, `.env.example`, README use 5434 | **High** | ✅ Done |
| 4.1.2 | Add `.svelte-kit` to `.gitignore` | **Low** | ✅ Done |
| 4.1.3 | Handle `err.code` type in edit routes | **Medium** | ✅ Done (`blocks/[blockId]/edit`, `blocks/new`) |
| 4.1.4 | Privacy/Terms pages | **High** | ✅ Done (`/privacy`, `/terms`) |
| 4.1.5 | GDPR endpoints (delete, export) | **High** | ✅ Done (`/api/account/delete`, `/api/account/export`) |
| 4.1.6 | Cookie notice | **Medium** | ✅ Done (layout-level banner) |
| 4.1.7 | Settings page nested form fix | **High** | ✅ Done (three-form structure) |

### 4.2 Build Process

```bash
# Ensure dependencies are installed
npm ci --production

# Build (NODE_ENV=production enables Node adapter)
NODE_ENV=production DATABASE_URL=<prod_db_url> npm run build

# Verify output
ls .svelte-kit/output/server/
```

### 4.3 Run Production Server

```bash
NODE_ENV=production \
DATABASE_URL=postgres://sv_app:<prod_password>@db:5432/svensktvin \
APP_HOST=https://svensktvin.se \
NODE_PORT=3000 \
node .svelte-kit/output/server/index.js
```

### 4.4 Process Manager (PM2)

```bash
npm i -g pm2
pm2 start .svelte-kit/output/server/index.js --name svensktvin
pm2 save
pm2 startup  # auto-start on boot
pm2 monit    # monitoring
```

---

## 5. Security Hardening

### 5.1 Session Security

Current `hooks.server.ts` sets `secure: true` in production (good). Verify:
- `sameSite: 'strict'` — correct for auth-only cookies
- `httpOnly: true` — correct, prevents JS access
- Session expiry: 30 days — acceptable

### 5.2 Rate Limiting

No rate limiter exists in the current SvelteKit app. Consider adding:

```typescript
// Middleware approach or express-rate-limit via @sveltejs/adapter-node
import rateLimit from 'express-rate-limit';
```

**Priority for Go-Live:** Low (low traffic expected for MVP)
**Priority for Scale:** High (magic-link endpoint is vulnerable to abuse without rate limiting)

### 5.3 Input Validation

Current validation:
- ✅ Block name required, variety required, area > 0
- ✅ Harvest yield > 0, harvest_year required, block_id required
- ✅ Harvest records have CHECK constraints (non-negative yield, brix, acid, health)
- ✅ Anonymity floor (3 vineyards) enforced in benchmark queries
- ❌ No CSRF tokens on form actions (SvelteKit enhances forms natively — mitigated by `enhance`)
- ❌ No email format validation before DB lookup (minor — `getUserByEmail` handles it)

### 5.4 Database Credentials

```
❌ DO NOT commit .env
✅ Use Docker secrets or environment injection
✅ Separate dev/prod credentials
✅ Rotate passwords periodically
```

---

## 6. Data & Content

### 6.1 Seed Data Policy

| Data Type | Production Strategy |
|-----------|---------------------|
| Varieties (20) | **Keep seeds** — these are reference catalog entries, not user data |
| Vineyards (3) | **Do NOT seed** — created by users during onboarding |
| Users (3) | **Manual SQL** — create admin account only |
| Blocks/Harvests | **No seeds** — all user-created |

### 6.2 Post-Golive Content

Before launching, consider:
- Adding 5-10 more common Swedish-appropriate varieties (Pinot Noir, Riesling, Solaris, etc.)
- Writing onboarding help text
- Pre-populating county/municipality lists if needed

---

## 7. Monitoring & Observability

### 7.1 Health Check

Add a health endpoint to the SvelteKit app:

```typescript
// src/routes/health/+server.ts
import { json } from '@sveltejs/kit';
import { sql } from '$lib/server/db.js';

export async function GET() {
  try {
    await sql`SELECT 1`;
    return json({ status: 'ok', db: 'connected' });
  } catch (err) {
    return json({ status: 'degraded', db: 'disconnected', error: err.message }, { status: 500 });
  }
}
```

### 7.2 Log Level

SvelteKit + Node produce structured output in production. Configure:

```bash
NODE_ENV=production   # Suppresses dev warnings
```

Consider adding a logging library:
- `pino` — fast, structured JSON logs
- `@sentry/node` — error tracking (free tier)

### 7.3 Uptime Monitoring

| Tool | Cost | Notes |
|------|------|-------|
| **UptimeRobot** | Free (5 monitors) | Simple, email/SMS alerts |
| **Better Stack** | Free tier | Structured logs + uptime |
| **Cronitor** | Free tier | API health + cron monitoring |

Monitor: `https://svensktvin.se/health` every 5 minutes.

---

## 8. GDPR & Legal

### 8.1 Data Collected

| Data | Purpose | Legal Basis | Retention |
|------|---------|-------------|-----------|
| Email address | Account creation | Consent | Until account deletion |
| Name | Profile display | Consent | Until account deletion |
| Vineyard name, location | Data collection | Consent | Until vineyard deletion |
| Harvest records (yield, Brix, acid) | Benchmarking | Consent | Until vineyard deletion |
| Session ID | Authentication | Legitimate interest | Until expiry (30 days) |
| Magic-link token hash | Login verification | Consent | 15 minutes |

### 8.2 Required Before Launch

- [x] Privacy policy page (`/privacy`)
- [x] Terms of service page (`/terms`)
- [x] Data export endpoint (`/api/account/export`) — GET, returns full user + vineyard data
- [x] Account deletion endpoint (`/api/account/delete`) — POST, checks vineyard ownership first
- [x] Cookie notice on first visit — dismissible banner, sets cookie_consent

### 8.3 Data Retention

The schema does not use soft deletes — there is no `deleted_at` column on any table. Deletes are hard. Implement the following before launch:

- **Account deletion** (`/api/account/delete`) — hard-deletes the user row; cascade rules handle sessions and magic-link tokens. Vineyard ownership must be transferred or the vineyard deleted first.
- **Harvest records** — keep indefinitely (benchmarking value); excluded from account deletion cascade.
- **Sessions** — auto-expire after 30 days (already implemented via `expires_at`).

---

## 9. Go-Live Procedure

### Phase 1: Infrastructure Setup (Day 1)

```bash
# 1. Provision VPS (Ubuntu 24.04 LTS recommended)
# 2. Install Docker + Docker Compose
# 3. Configure firewall
ufw allow 22/tcp   # SSH
ufw allow 80/tcp   # HTTP
ufw allow 443/tcp  # HTTPS
ufw enable

# 4. Clone repository
git clone <repo-url> /opt/svensktvin
cd /opt/svensktvin

# 5. Create .env with production values
cp .env.example .env
# Edit with production credentials
```

### Phase 2: Database Setup (Day 1)

```bash
# 1. Start database
docker compose up -d db
sleep 15

# 2. Run migrations
./scripts/migrate.sh

# 3. Verify
./scripts/test.sh

# 4. Create admin user (SQL)
psql postgres://sv_app:<password>@localhost:5434/svensktvin \
  -f scripts/create-admin.sql
```

### Phase 3: Application Deploy (Day 1)

```bash
# 1. Install dependencies
npm ci --production

# 2. Build
NODE_ENV=production DATABASE_URL=<prod_url> npm run build

# 3. Start with PM2
pm2 start .svelte-kit/output/server/index.js --name svensktvin -- \
  --port 3000

# 4. Configure Nginx + SSL
certbot --nginx -d svensktvin.se
# Reload nginx
```

### Phase 4: Verification (Day 1-2)

1. **DNS propagates** — verify domain resolves
2. **HTTP → HTTPS redirect** — port 80 returns 301
3. **TLS certificate** — valid, not self-signed
4. **Login flow** — magic link arrives in email
5. **Onboarding** — create a vineyard successfully
6. **Block creation** — add a block, verify variety search
7. **Harvest recording** — create a harvest record, verify constraints
8. **Benchmark page** — `/benchmarks` shows aggregated data
9. **Settings page** — member invite/remove works
10. **Logout** — session destroyed, redirect to login

### Phase 5: Monitoring & Handoff (Day 2)

1. Configure uptime monitor
2. Set up daily backup cron
3. Document admin access
4. Handoff to operations

---

## 10. Rollback Plan

### 10.1 Application Rollback

```bash
# Revert to previous commit
git checkout <previous-commit>
cd /opt/svensktvin
npm ci --production
NODE_ENV=production DATABASE_URL=<url> npm run build
pm2 restart svensktvin
```

Migrations are **idempotent** and **never roll back** — this is by design. Database state is forward-only.

### 10.2 Database Rollback

```bash
# Full restore from backup
docker exec sv_db pg_restore -U sv_app -d svensktvin /backups/svensktvin_YYYYMMDD.sql.gz
# OR for plain SQL:
zcat /backups/svensktvin_YYYYMMDD.sql.gz | psql postgres://sv_app@localhost:5434/svensktvin
```

### 10.3 Emergency Stop

```bash
pm2 stop svensktvin
# Or kill all app processes:
pkill -f 'svelte-kit/output/server'
```

---

## 11. Outstanding Items

| Item | Priority | Owner | Notes |
|------|----------|-------|-------|
| SMTP provider selection | **Critical** | Operator | Blocks magic-link login |
| Privacy policy page | **High** | Operator | GDPR requirement |
| Account deletion endpoint | **High** | Implement | GDPR right-to-erasure |
| Rate limiting | Medium | Implement | Protect auth endpoints |
| Error tracking (Sentry) | Medium | Implement | Production error visibility |
| Admin email provisioning | **High** | Operator | Need first admin user |
| Port alignment (5434 throughout) | ✅ Done | — | docker-compose, scripts, .env.example, README all consistent |
| PM2 systemd integration | Low | Operator | Auto-start on boot |

---

## Appendix A: Port Reference

| Service | Host Port | Container Port | Notes |
|---------|-----------|---------------|-------|
| PostgreSQL (Svenskt Vin) | 5434 | 5432 | PostGIS 16-3.4; 5434 avoids collision with Cinerarium :5433 |
| SvelteKit app | 3000 | 3000 | Node adapter |
| Nginx proxy | 80, 443 | — | TLS termination |

## Appendix B: Docker Compose Network

```
Host ── Nginx (80/443) ── SvelteKit App (:3000) ── Docker network ── PostgreSQL (:5432)
```

## Appendix C: Directory Structure (Production-Ready)

```
/opt/svensktvin/
├── .env                    # Production credentials (not in git)
├── docker-compose.yml      # Or separate services
├── db/
│   ├── migrations/         # 12 migrations (idempotent)
│   ├── seeds/              # Dev only
│   └── tests/              # Verification
├── scripts/
│   ├── migrate.sh
│   ├── seed.sh
│   └── test.sh
├── src/
│   ├── lib/server/         # db.ts, auth.ts, email.ts
│   └── routes/             # All SvelteKit routes
├── .svelte-kit/            # Build output (gitignored)
├── package.json
├── node_modules/           # (gitignored)
└── pm2.config.js           # Optional PM2 config
```
