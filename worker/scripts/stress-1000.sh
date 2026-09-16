#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
	echo "usage: bash scripts/stress-1000.sh <level1-or-level4-entry> [extra astrosim stress flags]" >&2
	exit 2
fi

worker_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$worker_directory"

go build -o astrosim ./cmd/astrosim
./astrosim stress "$1" -ships 1000 -out out/stress-1000 "${@:2}"
