#!/usr/bin/env bash
# Builds the control supervisor plus both microservices and starts them all.
# The control server (:8080) is the single owner of the estoque (:8081) and
# faturamento (:8082) processes, so it can start/stop them on demand.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo ">> Building services..."
mkdir -p "$ROOT/backend/estoque/bin" "$ROOT/backend/faturamento/bin" "$ROOT/backend/control/bin"
(cd "$ROOT/backend/estoque" && go build -o bin/estoque ./cmd/server)
(cd "$ROOT/backend/faturamento" && go build -o bin/faturamento ./cmd/server)
(cd "$ROOT/backend/control" && go build -o bin/control ./cmd/server)

echo ">> Stopping previous instances..."
fuser -k 8080/tcp 2>/dev/null && echo "  control stopped" || true
fuser -k 8081/tcp 2>/dev/null && echo "  estoque stopped" || true
fuser -k 8082/tcp 2>/dev/null && echo "  faturamento stopped" || true
sleep 1

echo ">> Starting control supervisor on :8080..."
(cd "$ROOT/backend/control" && setsid ./bin/control >/tmp/control.log 2>&1 </dev/null &)

sleep 2

if [ "${1:-}" = "--frontend" ]; then
  echo ">> Starting Angular dev server on :4200..."
  (cd "$ROOT/frontend/invoice-app" && setsid npx ng serve --port 4200 >/tmp/ngserve.log 2>&1 </dev/null &)
  sleep 8
fi

echo ">> Health checks:"
curl -s -m 3 http://localhost:8080/health || echo "control DOWN"
echo
curl -s -m 3 http://localhost:8081/health || echo "estoque DOWN"
echo
curl -s -m 3 http://localhost:8082/health || echo "faturamento DOWN"
echo
echo ">> All services up."