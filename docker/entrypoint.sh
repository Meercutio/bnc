#!/bin/sh
set -e
if [ -z "${SERVICE}" ]; then
  echo "SERVICE env is required (gateway/auth/game/matchmaking/stats)" >&2
  exit 1
fi
exec "/app/${SERVICE}"
