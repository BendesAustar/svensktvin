#!/usr/bin/env bash
# Techstack/svensktvin/scripts/test.sh
# Verifies schema integrity by running db/tests/*.sql checks.

set -euo pipefail

DB_URL="${DATABASE_URL:-postgres://sv_app:${PG_PASSWORD:-sv_dev_pass}@localhost:5434/svensktvin}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TESTS_DIR="$SCRIPT_DIR/../db/tests"

ERRORS=0

for f in "$TESTS_DIR"/*.sql; do
  filename="$(basename "$f")"
  echo "Testing: $filename"
  if ! psql "$DB_URL" -f "$f"; then
    echo "FAILED: $filename"
    ERRORS=$((ERRORS + 1))
  fi
done

if [ "$ERRORS" -gt 0 ]; then
  echo "$ERRORS test(s) failed."
  exit 1
fi

echo "All tests passed."
