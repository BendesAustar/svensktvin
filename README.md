# Svenskt Vin — Core Database

PostgreSQL database layer for the Svenskt Vin vineyard data collection platform.

## Quick Start

### Database

```bash
# 1. Start the database
docker compose up -d

# 2. Apply migrations (idempotent — safe to run multiple times)
./scripts/migrate.sh

# 3. Load seed data
./scripts/seed.sh

# 4. Run verification tests
./scripts/test.sh
```

### Application

```bash
# 1. Install dependencies
npm install

# 2. Copy and configure environment
cp .env.example .env
# Edit .env with your DATABASE_URL and SMTP settings

# 3. Build (requires DATABASE_URL in env)
DATABASE_URL=postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin npm run build

# 4. Run dev server
DATABASE_URL=postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin npm run dev

# 5. Run tests
DATABASE_URL=postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin npm test
```

## Architecture

```
db/
├── migrations/   # 12 numbered SQL migrations (idempotent)
├── seeds/        # 5 seed scripts (varieties, vineyards, users, blocks, harvests)
└── tests/        # 4 SQL test scripts (schema, data integrity, constraints, benchmark)
scripts/
├── migrate.sh    # Idempotent migration runner with _migrations_applied tracking
├── seed.sh       # Sequential seed loader
└── test.sh       # Test suite runner
```

## Migrations

| # | File | Description |
|---|------|-------------|
| 1 | 001_extensions.sql | pg_trgm + postgis |
| 2 | 002_users.sql | User accounts with is_admin |
| 3 | 003_varieties.sql | Variety catalog with trigram index |
| 4 | 004_vineyards.sql | Vineyard entities with soft deletes |
| 5 | 005_blocks.sql | Vineyard blocks with soft deletes |
| 6 | 006_harvest_records.sql | Harvest records with CHECK constraints |
| 7 | 007_vineyard_members.sql | User-vineyard membership |
| 8 | 008_sessions.sql | Server-side sessions |
| 9 | 009_magic_link_tokens.sql | Magic-link login tokens |
| 10 | 010_varieties_submitted_by_fk.sql | Varieties → Vineyards FK |
| 11 | 011_updated_at_triggers.sql | Auto-update triggers on 3 tables |
| 12 | 012_indexes_and_views.sql | Performance indexes + soft-delete views |

## Seed Data

- 20 curated varieties (PIWI + classic vinifera)
- 3 vineyard fixtures (Skåne + Blekinge)
- 3 users (admin, owner, editor) with member assignments
- 3 blocks for Vingård A
- 3 harvest records for Vingård A

## Testing

```bash
# Run all tests
./scripts/test.sh

# Run individual test
psql postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin -f db/tests/001_verify_schema.sql
```

## Environment

Copy `.env.example` to `.env` and adjust as needed:

```bash
cp .env.example .env
```

## Notes

- Port 5434 (not 5433) to avoid collision with Cinerarium PostgreSQL
- Soft deletes on vineyards/blocks/harvest_records preserve historical benchmark data
- Active views (`active_vineyards`, `active_blocks`, `active_harvest_records`) filter deleted rows transparently
- Trigram fuzzy matching on `varieties.name` for variety submission flow
