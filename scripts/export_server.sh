#!/usr/bin/env bash
# Export the FARP benchmark data pack used by the headless evaluation server.
# Requires a Godot 4 editor binary with Linux export templates installed.
#
# Usage:
#   GODOT=/path/to/Godot_v4.3-stable_linux.x86_64 ./web/scripts/export_server.sh
set -euo pipefail

GODOT="${GODOT:-godot}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../game" && pwd)"
OUT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/server_build"

mkdir -p "$OUT_DIR"

echo "Exporting 'Linux Server' pack from $PROJECT_DIR ..."
"$GODOT" --headless --path "$PROJECT_DIR" --export-pack "Linux Server" "$OUT_DIR/astroswarm.pck"

echo "Done -> $OUT_DIR/astroswarm.pck"
echo "Quick local test:"
echo "  $GODOT --headless --main-pack $OUT_DIR/astroswarm.pck -- --bench --trials=5 --nmax=8 --out=/tmp/farp.json"
