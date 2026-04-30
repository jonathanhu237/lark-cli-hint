#!/usr/bin/env bash
# Refresh committed lark-cli schema fixtures.
#
# Run from anywhere; the script normalizes its cwd to repo root.
# Requires: lark-cli on PATH.

set -euo pipefail

cd "$(dirname "$0")/.."

services=(im drive task minutes)

for svc in "${services[@]}"; do
  echo "Refreshing fixtures/schema/${svc}.json ..."
  lark-cli schema "$svc" > "fixtures/schema/${svc}.json"
done

lark-cli --version | awk '{print $3}' > fixtures/schema/.lark-cli-version
echo "Captured lark-cli $(cat fixtures/schema/.lark-cli-version)"
