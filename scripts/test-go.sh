#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Archives contain independent one-off Go research programs. Several share
# main symbols within one directory and are not buildable Go packages.
# Keep them intact; cover actual application packages instead.
packages=(.)
for dir in */; do
  case "$dir" in
    strategy_templates/|temp_strategy/|research_specs/|static/|doc/|data/|img/|views/|conf/)
      continue
      ;;
  esac
  if find "$dir" -type f -name '*.go' -print -quit | grep -q .; then
    packages+=("./${dir%/}/...")
  fi
done

printf 'Testing application packages (%s patterns); research archives excluded\n' "${#packages[@]}"
go test -count=1 "${packages[@]}" "$@"
