#!/usr/bin/env bash
# Stops the microservices and the Angular dev server.
set -uo pipefail

echo ">> Stopping services..."
fuser -k 8080/tcp 2>/dev/null && echo "  control stopped" || echo "  control not running"
fuser -k 8081/tcp 2>/dev/null && echo "  estoque stopped" || echo "  estoque not running"
fuser -k 8082/tcp 2>/dev/null && echo "  faturamento stopped" || echo "  faturamento not running"
fuser -k 4200/tcp 2>/dev/null && echo "  frontend stopped" || echo "  frontend not running"

echo ">> Done."