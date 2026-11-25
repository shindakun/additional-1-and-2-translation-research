#!/bin/sh
# Usage: scripts/build_text_test.sh GAME TRANSLATION.json
# Builds research/builds/<GAME>-text.iso: the original image with the
# translations written into main.dat in place (docs/0004). Needs kfa built
# at the repo root and the disc extracted by scripts/extract_iso.sh.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=$1 json=$2
kfa="$root/kfa"
out="$root/research/builds"
mkdir -p "$out"
"$kfa" text import "$root/research/extract/$game/PSP_GAME" "$json" "$out/$game-text-main.dat"
cp "$root/research/iso/$game.iso" "$out/$game-text.iso"
"$kfa" iso put "$out/$game-text.iso" PSP_GAME/USRDIR/GameData/01_UmdTitle/main.dat "$out/$game-text-main.dat"
"$kfa" dat verify "$root/research/extract/$game/PSP_GAME/SYSDIR/BOOT.BIN" "$out/$game-text-main.dat"
