#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> API"
(cd apps/api && go build -o /tmp/harc-api ./cmd/server)
if ! curl -sf http://127.0.0.1:8000/health >/dev/null 2>&1; then
  /tmp/harc-api >/tmp/harc-api.log 2>&1 &
  echo $! >/tmp/harc-api.pid
  sleep 1
fi

echo "==> speaker dry-run"
(cd apps/speaker && go build -o /tmp/harc-speaker ./cmd/harc-speaker)
/tmp/harc-speaker -api http://127.0.0.1:8000 -mode sleep -dry-run -duration 1

echo "==> speaker render"
/tmp/harc-speaker -api http://127.0.0.1:8000 -mode focus -render /tmp/harc-focus.ogg -render-sec 5 -duration 1
ls -la /tmp/harc-focus.ogg

echo "OK"
