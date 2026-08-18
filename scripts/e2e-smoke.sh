#!/usr/bin/env bash
# End-to-end smoke test against the running microservices.
# Prerequisites: both services up (scripts/run-all.sh) and PostgreSQL running.
set -euo pipefail

ESTOQUE="http://localhost:8081"
FAT="http://localhost:8082"
INTERNAL="korp-internal-token-dev"
PASS=0
FAIL=0

check() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    echo "  ✓ $desc ($actual)"
    PASS=$((PASS + 1))
  else
    echo "  ✗ $desc — expected $expected, got $actual"
    FAIL=$((FAIL + 1))
  fi
}

code() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}

echo "== Health checks =="
check "estoque health" 200 "$(code $ESTOQUE/health)"
check "faturamento health" 200 "$(code $FAT/health)"

echo "== Auth =="
LOGIN=$(curl -s -X POST $FAT/api/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
check "login ok" 200 "$(code -X POST $FAT/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}')"
check "login wrong password" 401 "$(code -X POST $FAT/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"wrong"}')"
check "no token rejected" 401 "$(code $ESTOQUE/api/products)"

echo "== Products =="
STAMP=$(date +%s)
CODE1="SMOKE$STAMP"
CODE2="SMOKE2$STAMP"
curl -s -X POST $ESTOQUE/api/products -H "X-Internal-Token: $INTERNAL" -H 'Content-Type: application/json' \
  -d "{\"code\":\"$CODE1\",\"description\":\"Smoke A\",\"stock_quantity\":10}" >/dev/null
curl -s -X POST $ESTOQUE/api/products -H "X-Internal-Token: $INTERNAL" -H 'Content-Type: application/json' \
  -d "{\"code\":\"$CODE2\",\"description\":\"Smoke B\",\"stock_quantity\":5}" >/dev/null
check "create product" 201 "$(code -X POST $ESTOQUE/api/products -H "X-Internal-Token: $INTERNAL" -H 'Content-Type: application/json' -d "{\"code\":\"X$STAMP\",\"description\":\"dup\",\"stock_quantity\":1}")"
check "duplicate product" 409 "$(code -X POST $ESTOQUE/api/products -H "X-Internal-Token: $INTERNAL" -H 'Content-Type: application/json' -d "{\"code\":\"$CODE1\",\"description\":\"dup\",\"stock_quantity\":1}")"

echo "== Invoices =="
INV=$(curl -s -X POST $FAT/api/invoices -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"items\":[{\"product_code\":\"$CODE1\",\"quantity\":2},{\"product_code\":\"$CODE2\",\"quantity\":1}]}")
INV_ID=$(echo "$INV" | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")
INV_NUM=$(echo "$INV" | python3 -c "import sys,json;print(json.load(sys.stdin)['number'])")
echo "  invoice #$INV_NUM created (id $INV_ID)"
check "create invoice" 201 "$(code -X POST $FAT/api/invoices -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"items":[{"product_code":"'$CODE1'","quantity":1}]}')"
check "invoice without items" 400 "$(code -X POST $FAT/api/invoices -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"items":[]}')"

echo "== Print (close) =="
KEY="smoke-print-$STAMP"
check "print invoice" 200 "$(code -X POST $FAT/api/invoices/$INV_ID/print -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $KEY")"
check "stock consumed (CODE1 10->8)" "8" "$(curl -s $ESTOQUE/api/products/$CODE1 -H "X-Internal-Token: $INTERNAL" | python3 -c "import sys,json;print(json.load(sys.stdin)['stock_quantity'])")"
check "print closed invoice" 409 "$(code -X POST $FAT/api/invoices/$INV_ID/print -H "Authorization: Bearer $TOKEN")"
check "idempotent replay" 200 "$(code -X POST $FAT/api/invoices/$INV_ID/print -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $KEY")"
check "stock unchanged after replay (8)" "8" "$(curl -s $ESTOQUE/api/products/$CODE1 -H "X-Internal-Token: $INTERNAL" | python3 -c "import sys,json;print(json.load(sys.stdin)['stock_quantity'])")"

echo
echo "== Result: $PASS passed, $FAIL failed =="
[ "$FAIL" -eq 0 ]