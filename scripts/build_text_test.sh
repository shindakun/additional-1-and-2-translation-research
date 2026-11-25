#!/bin/sh
# Usage: scripts/build_text_test.sh GAME TRANSLATION.json [IMAGEDIR]
# Builds research/builds/<GAME>-text.iso: the original image with the
# translations written into main.dat in place (docs/0004) and, if IMAGEDIR
# is given, the edited images in it (docs/0012). Needs kfa built at the repo
# root and the disc extracted by scripts/extract_iso.sh.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=$1 json=$2 imgdir=${3:-}
kfa="$root/kfa"
gamedir="$root/research/extract/$game/PSP_GAME"
out="$root/research/builds"
dat="$out/$game-text-main.dat"
mkdir -p "$out"
"$kfa" text import "$gamedir" "$json" "$dat"
if [ -n "$imgdir" ]; then
	"$kfa" img import "$gamedir" "$imgdir" "$dat.img" "$dat"
	mv "$dat.img" "$dat"
fi
cp "$root/research/iso/$game.iso" "$out/$game-text.iso"
"$kfa" iso put "$out/$game-text.iso" PSP_GAME/USRDIR/GameData/01_UmdTitle/main.dat "$dat"
"$kfa" dat verify "$gamedir/SYSDIR/BOOT.BIN" "$dat"
