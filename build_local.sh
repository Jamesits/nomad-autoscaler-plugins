#!/usr/bin/env bash
set -Eeuo pipefail
cd -- "$( dirname -- "${BASH_SOURCE[0]}" )"

# for local debugging only

goreleaser build --snapshot --clean --single-target
pushd dist > /dev/null
mkdir -p plugins
cp -r -- ./*/* plugins/
popd > /dev/null

echo "[i] Plugin dir: $(readlink -f dist/plugins)"
