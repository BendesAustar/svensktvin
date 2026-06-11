#!/usr/bin/env bash
# Techstack/svensktvin/scripts/migrate.sh
# Applies migrations in order, tracking applied files in _migrations_applied.
# Idempotent: skips already-applied migrations.

set -euo pipefail

DB_URL="${DATABASE_URL:-postgres://sv_app:${PG_PASSWORD:-sv_dev_pass}@localhost:5434/svensktvin}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR/../db/migrations"

psql "$DB_URL" -c "
CREATE TABLE IF NOT EXISTS _migrations_applied (
  filename    text    PRIMARY KEY,
  applied_at  timestamptz NOT NULL DEFAULT now()
);
"

for f in "$MIGRATIONS_DIR"/*.sql; do
  filename="$(basename "$f")"
  if psql "$DB_URL" -tAc "SELECT 1 FROM _migrations_applied WHERE filename = '$filename'" | grep -q 1; then
    echo "SKIP (already applied): $filename"
    continue
  fi
  echo "Applying: $filename"
  if ! psql "$DB_URL" -f "$f"; then
    echo "FAILED: $filename"
    exit 1
  fi
  psql "$DB_URL" -c "INSERT INTO _migrations_applied (filename) VALUES ('$filename') ON CONFLICT (filename) DO NOTHING"
  echo "DONE: $filename"
done

echo "Migration complete."
