#!/bin/sh
# Usage: scripts/headless_shots.sh IMAGE OUTDIR SECONDS...
# Runs IMAGE in PPSSPPHeadless (software renderer, no window) and saves a PNG
# at each emulated second given. Uses its own memory stick under
# research/ppsspp/headless-ms. Build PPSSPPHeadless first (docs/0009).
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
headless="$root/research/tools/ppsspp-src/build-headless/PPSSPPHeadless"
image=$1 out=$2
shift 2
mkdir -p "$out" "$root/research/ppsspp/headless-ms"
list="" last=0
for t in "$@"; do
	list="$list$t:$out/t$t.png,"
	[ "$t" -gt "$last" ] && last=$t
done
"$headless" --memstick="$root/research/ppsspp/headless-ms" --graphics=software \
	--timeout-emulated=$((last + 2)) --timeout-wall=900 \
	"--screenshot-at=${list%,}" "$image" >"$out/headless.log" 2>&1 || true
grep -v 'Screenshot saved' "$out/headless.log" | tail -3
ls "$out"/t*.png
