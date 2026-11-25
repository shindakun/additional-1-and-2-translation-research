#!/bin/sh
# Usage: scripts/build_vwf_test.sh [kfaI|kfaII]
# Builds research/builds/<game>-vwf-test.iso: the original image with
# proportional ASCII metrics in every font (docs/0005-fonts.md). File sizes do
# not change, so files are replaced in place. Needs kfa built at the repo root.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=${1:-kfaI}
kfa="$root/kfa"
src="$root/research/iso/$game.iso"
out="$root/research/builds/$game-vwf-test.iso"
tmp="$root/research/builds/$game-vwf-test.d"
mkdir -p "$tmp"
cp "$src" "$out"
"$kfa" iso extract "$src" "$tmp/orig" >/dev/null
for p in $("$kfa" iso ls "$src" | awk '{print $3}' | grep -E '^PSP_GAME/USRDIR/(Font/BitFont_Game[0-9]+|Bnd/AdvMenu)\.bnd$'); do
	"$kfa" font vwf "$tmp/orig/$p" "$tmp/$(basename "$p")"
	"$kfa" iso put "$out" "$p" "$tmp/$(basename "$p")"
done
shasum -a 1 "$out"
