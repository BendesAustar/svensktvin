# Svenskt Vin — Operator Go-Live Checklist

**Repository:** `/home/neurograft/Techstack/svensktvin` (or cloned from git)
**Build:** ✅ Clean production build (`npm run build`) — all routes compile
**DB:** ✅ 12 migrations applied, all integrity tests pass, 20 varieties seeded

---

## Pre-Launch Checklist

### 1. SMTP Configuration ⚠️ BLOCKER
Magic-link login requires SMTP. Configure before launch:

| Variable | Example | Description |
|----------|---------|-------------|
| `SMTP_HOST` | `smtp.gmail.com` | Your SMTP server |
| `SMTP_PORT` | `587` | Usually 587 (STARTTLS) |
| `SMTP_USER` | `noreply@svensktvin.se` | Sender address |
| `SMTP_PASS` | `<app-password>` | Auth credentials |
| `SMTP_FROM` | `noreply@svensktvin.se` | Display name |

Set in `.env` on the production host.

### 2. Production Database
- Generate a new `PG_PASSWORD` (different from `sv_dev_pass`)
- Run migrations: `./scripts/migrate.sh`
- Verify: `./scripts/test.sh`
- Create admin user (magic-link only, no password):
  ```sql
  INSERT INTO users (email, name, is_admin, active)
  VALUES ('admin@svensktvin.se', 'Admin', true, true);
  ```

### 3. Hosting & Infrastructure
Recommended: Dedicated VPS (Hetzner, DigitalOcean) for MVP.

```bash
# 1. Install Docker + Docker Compose (Ubuntu 24.04 LTS)
apt install docker.io docker-compose-plugin

# 2. Firewall
ufw allow 22/tcp   # SSH
ufw allow 80/tcp   # HTTP
ufw allow 443/tcp  # HTTPS (once TLS is configured)
ufw enable

# 3. Clone repo & configure
git clone <repo-url> /opt/svensktvin
cd /opt/svensktvin
cp .env.example .env
# Edit .env with production values (see §1 above + §4 below)
```

### 4. Docker Compose — App Service (Missing)
The current `docker-compose.yml` only defines the `db` service. Add:

```yaml
services:
  db:
    # ... existing config ...

  app:
    build: .
    ports:
      - "3000:3000"
    environment:
      DATABASE_URL: postgres://sv_app:${PG_PASSWORD}@db:5432/svensktvin
      APP_HOST: https://svensktvin.se
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
```

Add to `.env.example`:
```bash
PG_PASSWORD=CHANGE_ME
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@svensktvin.se
```

### 5. Nginx + TLS (Production)
- Obtain TLS certificate (Let's Encrypt / certbot recommended)
- Configure Nginx to proxy `/` → `http://localhost:3000`
- Redirect HTTP → HTTPS
- Set `APP_HOST=https://svensktvin.se` in `.env`
- With TLS, cookies will use `secure: true`

### 6. DNS
- Point domain `svensktvin.se` (and `www.`) to production host IP
- Verify: `dig svensktvin.se`

### 7. Backup Strategy
```bash
# Cron: daily backup at 3 AM
0 3 * * * docker exec sv_db pg_dump -U sv_app -d svensktvin | gzip > /backups/svensktvin_$(date +\%Y\%m\%d).sql.gz
# Retain 30 days
find /backups -name 'svensktvin_*.sql.gz' -mtime +30 -delete
```

### 8. Monitoring
- Health check endpoint: `GET /health` → `{status, checks{app, db}, uptime}`
- Configure external monitor (UptimeRobot, UptimeKuma, etc.) to hit `/health` every 5 min
- Consider Sentry for error tracking (medium priority)

### 9. Startup on Boot
```bash
# PM2 systemd integration
pm2 startup systemd
pm2 start server.js --name svensktvin
pm2 save
```

### 10. First Admin Account
The admin user is created via SQL (see §2 above). After creation, the operator:
1. Visits `/login` and requests a magic link
2. Logs in via the email link
3. Onboards their first vineyard
4. Invites additional users from Settings

---

## What's Already Built (No Code Needed)

| Feature | Route | Notes |
|---------|-------|-------|
| Authentication | `/login` → `/auth/verify` | Magic link, 15-min expiry |
| Onboarding | `/onboard` | Creates first vineyard |
| Vineyard dashboard | `/vineyard/[id]` | Home page per vineyard |
| Blocks management | `/vineyard/[id]/blocks` | Create/edit blocks |
| Harvest records | `/vineyard/[id]/harvest/new` | Record harvest data |
| Settings | `/vineyard/[id]/settings` | Update vineyard, manage members |
| Benchmarks | `/benchmarks` | Aggregate statistics |
| Privacy policy | `/privacy` | GDPR-compliant |
| Terms of service | `/terms` | |
| Account deletion | `POST /api/account/delete` | Checks vineyard ownership first |
| Data export | `GET /api/account/export` | Full JSON export |
| Cookie notice | Layout-level | Dismissible banner |
| Health check | `GET /health` | `{status, checks{app, db}}` |

---

## Invite Flow Note (Post-Golive)
Current invite only works with existing registered users. Real-world use case:
- Owner wants to invite a vineyard neighbor who doesn't have an account
- **Future:** Send email invite with signup link (requires email template + route)
- **Defer to post-golive** — current flow works for early adopter group

---

## Launch Command (Production)
```bash
cd /opt/svensktvin
docker compose up -d db
sleep 15
./scripts/migrate.sh
./scripts/test.sh
# Once DB is ready, start the app
docker compose up -d app
```

---

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Login doesn't send email | Verify `SMTP_*` env vars in `.env` |
| 502 Bad Gateway | `docker compose logs app` — check DB_URL format |
| Cookie not persisting | Ensure `APP_HOST` matches actual domain (not localhost) |
| DB connection fails | `docker exec sv_db pg_isready -U sv_app` |
| Health check degraded | Check `docker compose logs db` for DB issues |

---

**Last updated:** 2026-06-12
**Build status:** ✅ Clean production build
**DB status:** ✅ All tests pass
