#!/usr/bin/env bash
# Creates the PostgreSQL role and databases used by the microservices.
# Idempotent: safe to run multiple times.
set -euo pipefail

DB_USER="${DB_USER:-korp}"
DB_PASSWORD="${DB_PASSWORD:-korp_dev_2026}"

run_as_postgres() {
  su postgres -c "cd /tmp && $*"
}

echo ">> Creating role ${DB_USER} (if missing)..."
run_as_postgres "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'\" | grep -q 1" \
  || run_as_postgres "psql -c \"CREATE ROLE ${DB_USER} WITH LOGIN PASSWORD '${DB_PASSWORD}';\""

for db in korp_estoque korp_faturamento; do
  echo ">> Ensuring database ${db}..."
  run_as_postgres "psql -tc \"SELECT 1 FROM pg_database WHERE datname='${db}'\" | grep -q 1" \
    || run_as_postgres "psql -c \"CREATE DATABASE ${db} OWNER ${DB_USER};\""
done

echo ">> Done. Databases korp_estoque and korp_faturamento are ready."