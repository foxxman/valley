#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=./_dev.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_dev.sh"

require_env
"${COMPOSE[@]}" down --remove-orphans
