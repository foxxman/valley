#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=./_dev.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_dev.sh"

ensure_env
"${COMPOSE[@]}" up --build -d
"${COMPOSE[@]}" ps
"${COMPOSE[@]}" logs -f backend
