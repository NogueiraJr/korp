#!/usr/bin/env bash
# Resets the databases to a pristine demo state (schema preserved, data wiped).
set -euo pipefail

run() {
  PGPASSWORD=korp_dev_2026 psql -h localhost -U korp -d "$1" -c "$2" >/dev/null
}

echo ">> Cleaning korp_estoque..."
run korp_estoque "TRUNCATE TABLE products RESTART IDENTITY CASCADE;"

echo ">> Cleaning korp_faturamento..."
run korp_faturamento "TRUNCATE TABLE invoices, invoice_items, idempotency_keys CASCADE;"
run korp_faturamento "UPDATE invoice_counters SET next_number = 0 WHERE id = 1;"

echo ">> Databases ready for a fresh demo."