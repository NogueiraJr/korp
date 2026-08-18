#!/usr/bin/env bash
# Demonstrates failure handling: with the Estoque service DOWN, a print request
# returns 503 after retries and the invoice stays OPEN. After recovery the
# same print succeeds.
# Prerequisites: faturamento up. EstoQUE will be stopped and restarted here.
set -euo pipefail

ESTOQUE_PID="$(fuser 8081/tcp 2>/dev/null || true)"
FAT="http://localhost:8082"
ESTOQUE="http://localhost:8081"
INTERNAL="korp-internal-token-dev"

STAMP=$(date +%s)
CODE="FAIL$STAMP"

# ensure estoque is up to seed data
if ! curl -sf "$ESTOQUE/health" >/dev/null 2>&1; then
  echo "!! Estoque is down. Start it first (scripts/run-all.sh)." >&2
  exit 1
fi

echo ">> Creating product $CODE with stock = 5..."
curl -s -X POST $ESTOQUE/api/products -H "X-Internal-Token: $INTERNAL" -H 'Content-Type: application/json' \
  -d "{\"code\":\"$CODE\",\"description\":\"Produto p/ teste de falha\",\"stock_quantity\":5}" >/dev/null

TOKEN=$(curl -s -X POST $FAT/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

INV=$(curl -s -X POST $FAT/api/invoices -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"product_code\":\"$CODE\",\"quantity\":2}]}" | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")

echo ">> Stopping Estoque service..."
fuser -k 8081/tcp 2>/dev/null || true
sleep 0.5

echo ">> Printing with Estoque DOWN (expect 503 after retries)..."
HTTP=$(curl -s -X POST $FAT/api/invoices/$INV/print -H "Authorization: Bearer $TOKEN" -w '%{http_code}' -o /tmp/fail1.json)
echo "  status: $HTTP"
echo "  body: $(cat /tmp/fail1.json)"

STATE=$(curl -s $FAT/api/invoices/$INV -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json;print(json.load(sys.stdin)['status'])")
echo ">> Invoice status: $STATE (must still be OPEN)"

echo ">> Restarting Estoque service..."
(cd "$(dirname "${BASH_SOURCE[0]}")/../backend/estoque" && setsid ./bin/estoque >/tmp/estoque.log 2>&1 </dev/null &)
sleep 1.5

echo ">> Printing again (recovery, expect 200)..."
HTTP=$(curl -s -X POST $FAT/api/invoices/$INV/print -H "Authorization: Bearer $TOKEN" -w '%{http_code}' -o /tmp/fail2.json)
echo "  status: $HTTP"
python3 -c "import json;d=json.load(open('/tmp/fail2.json'));print('  invoice status:', d['status'])"

FINAL=$(curl -s $ESTOQUE/api/products/$CODE -H "X-Internal-Token: $INTERNAL" | python3 -c "import sys,json;print(json.load(sys.stdin)['stock_quantity'])")
echo ">> Final stock of $CODE: $FINAL (expected 3)"
echo ">> Recovery demonstrated: service returns, print succeeds, stock consumed once."

[ "$HTTP" = "200" ] && [ "$FINAL" = "3" ]