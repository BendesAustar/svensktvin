#!/usr/bin/env bash
# Techstack/svensktvin/scripts/seed.sh
# Loads seed data from db/seeds/ in alphabetical order.

set -euo pipefail

DB_URL="${DATABASE_URL:-postgres://sv_app:${PG_PASSWORD:-sv_dev_pass}@localhost:5434/svensktvin}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SEEDS_DIR="$SCRIPT_DIR/../db/seeds"

for f in "$SEEDS_DIR"/*.sql; do
  filename="$(basename "$f")"
  echo "Seeding: $filename"
  if ! psql "$DB_URL" -f "$f"; then
    echo "FAILED: $filename"
    exit 1
  fi
done

echo "Seeding complete."
