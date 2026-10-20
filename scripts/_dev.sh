#!/usr/bin/env bash
# shellcheck shell=bash
# Shared helpers for dev docker compose scripts.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

COMPOSE=(docker compose -f docker-compose.yml -f docker-compose.dev.yml --env-file .env)

ensure_env() {
  if [[ ! -f .env ]]; then
    cp .env.example .env
    echo "created .env from .env.example"
  fi
}

require_env() {
  if [[ ! -f .env ]]; then
    echo "missing .env — run scripts/dev-up.sh first" >&2
    exit 1
  fi
}
