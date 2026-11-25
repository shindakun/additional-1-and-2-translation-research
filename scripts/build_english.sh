#!/bin/sh
# Usage: scripts/build_english.sh [GAME]
# Builds research/builds/<GAME>-english.iso (default GAME kfaI) with every
# current translation: main.dat text and images (docs/0004, docs/0012), FMG
# messages (docs/0013) and movie subtitles with the boot font (docs/0010).
# Inputs are in research/text, research/img/edit and research/subs.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=${1:-kfaI}
kfa="$root/kfa"
gamedir="$root/research/extract/$game/PSP_GAME"
out="$root/research/builds"
tmp="$out/.tmp-$game"
iso="$out/$game-english.iso"
mkdir -p "$tmp"
trap 'rm -rf "$tmp"' EXIT

"$kfa" img import "$gamedir" "$root/research/img/edit/$game" "$tmp/img.dat"
"$kfa" text import "$gamedir" "$root/research/text/$game-test.json" "$tmp/main.dat" "$tmp/img.dat" 2>"$tmp/text.log"
grep -c '^not written' "$tmp/text.log" >/dev/null && sed -n '/^not written/p' "$tmp/text.log" >&2
"$kfa" fmg import "$gamedir" "$root/research/text/$game-fmg-test.json" "$tmp/AdvSys.bnd"
subs=""
for t in "$root"/research/subs/"$game"-movie*.tsv; do
	[ -e "$t" ] || continue
	m=$(basename "$t" .tsv)
	subs="$subs ${m#"$game"-}.pmf=$t"
done
# shellcheck disable=SC2086 # subs is a list of arguments
"$kfa" patch subtitles --boot-font "$gamedir/SYSDIR/BOOT.BIN" "$tmp/BOOT.BIN" $subs

cp "$root/research/iso/$game.iso" "$iso.part"
"$kfa" iso put "$iso.part" PSP_GAME/USRDIR/GameData/01_UmdTitle/main.dat "$tmp/main.dat"
"$kfa" iso put "$iso.part" PSP_GAME/USRDIR/Bnd/AdvSys.bnd "$tmp/AdvSys.bnd"
"$kfa" iso put "$iso.part" PSP_GAME/SYSDIR/EBOOT.BIN "$tmp/BOOT.BIN"
"$kfa" dat verify "$gamedir/SYSDIR/BOOT.BIN" "$tmp/main.dat"
mv "$iso.part" "$iso"
echo "built $iso"
