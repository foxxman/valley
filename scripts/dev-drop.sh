#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=./_dev.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_dev.sh"

require_env
"${COMPOSE[@]}" down --volumes --remove-orphans
echo "containers, networks and volumes removed"
