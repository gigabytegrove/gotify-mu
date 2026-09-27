#!/bin/sh
set -eu

RUNTIME_BINARY="${MONITA_RUNTIME_BINARY:-/app/data/.monita-runtime/monita-app}"

if [ -x "$RUNTIME_BINARY" ]; then
    exec "$RUNTIME_BINARY" "$@"
fi

exec /app/monita-app "$@"
