#!/bin/sh
# Usage: scripts/build_text_test.sh GAME TRANSLATION.json [IMAGEDIR [FMG.json]]
# Builds research/builds/<GAME>-text.iso: the original image with the
# translations written into main.dat in place (docs/0004), the edited images
# in IMAGEDIR (docs/0012) and the FMG translations in AdvSys.bnd (docs/0013).
# An empty IMAGEDIR argument skips the images. Needs kfa built at the repo
# root and the disc extracted by scripts/extract_iso.sh.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=$1 json=$2 imgdir=${3:-} fmgjson=${4:-}
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
if [ -n "$fmgjson" ]; then
	"$kfa" fmg import "$gamedir" "$fmgjson" "$out/$game-AdvSys.bnd"
	"$kfa" iso put "$out/$game-text.iso" PSP_GAME/USRDIR/Bnd/AdvSys.bnd "$out/$game-AdvSys.bnd"
fi
"$kfa" dat verify "$gamedir/SYSDIR/BOOT.BIN" "$dat"
