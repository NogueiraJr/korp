#!/usr/bin/env bash
# Demonstrates concurrency handling: a product with stock 1 is used by two
# invoices printed at the same time. Only one may succeed.
# Prerequisites: services up (scripts/run-all.sh).
set -euo pipefail

ESTOQUE="http://localhost:8081"
FAT="http://localhost:8082"
INTERNAL="korp-internal-token-dev"

STAMP=$(date +%s)
CODE="RACE$STAMP"

echo ">> Creating product $CODE with stock = 1..."
curl -s -X POST $ESTOQUE/api/products -H "X-Internal-Token: $INTERNAL" -H 'Content-Type: application/json' \
  -d "{\"code\":\"$CODE\",\"description\":\"Produto para teste de concorrencia\",\"stock_quantity\":1}" >/dev/null

TOKEN=$(curl -s -X POST $FAT/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

echo ">> Creating two invoices that each use 1 unit of $CODE..."
I1=$(curl -s -X POST $FAT/api/invoices -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"product_code\":\"$CODE\",\"quantity\":1}]}" | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")
I2=$(curl -s -X POST $FAT/api/invoices -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"product_code\":\"$CODE\",\"quantity\":1}]}" | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")

echo ">> Printing both invoices CONCURRENTLY..."
C1=$(curl -s -X POST $FAT/api/invoices/$I1/print -H "Authorization: Bearer $TOKEN" -o /tmp/race1.json -w '%{http_code}')
echo "  invoice 1 -> $C1"
C2=$(curl -s -X POST $FAT/api/invoices/$I2/print -H "Authorization: Bearer $TOKEN" -o /tmp/race2.json -w '%{http_code}')
echo "  invoice 2 -> $C2"
wait 2>/dev/null || true

FINAL=$(curl -s $ESTOQUE/api/products/$CODE -H "X-Internal-Token: $INTERNAL" | python3 -c "import sys,json;print(json.load(sys.stdin)['stock_quantity'])")
echo ">> Final stock of $CODE: $FINAL (expected 0 — no overselling)"

echo ">> Invoice 1 HTTP: $C1 | Invoice 2 HTTP: $C2"
echo ">> Exactly one 200, one 409, and stock not negative = concurrency handled correctly."

[ "$FINAL" = "0" ] && [ "$C1$C2" = "200409" -o "$C1$C2" = "409200" ]