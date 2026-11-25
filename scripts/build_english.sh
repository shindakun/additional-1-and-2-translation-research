#!/bin/sh
# Usage: scripts/build_english.sh [GAME]
# Builds research/builds/<GAME>-english.iso (default GAME kfaI) with every
# current translation: main.dat text and images (docs/kfa1/0004, docs/kfa1/0012), FMG
# messages (docs/kfa1/0013), movie subtitles with the boot font (docs/kfa1/0010) and
# fonts with padded ASCII glyphs (docs/kfa1/0005).
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
"$kfa" text import "$gamedir" "$root/research/text/$game-test.json" "$tmp/main.dat" "$tmp/img.dat" 2>"$tmp/text.log" >/dev/null
grep -c '^not written' "$tmp/text.log" >/dev/null && sed -n '/^not written/p' "$tmp/text.log" >&2
"$kfa" fmg import "$gamedir" "$root/research/text/$game-fmg-test.json" "$tmp/AdvSys.bnd"
subs=""
for t in "$root"/research/subs/"$game"-movie*.tsv; do
	[ -e "$t" ] || continue
	m=$(basename "$t" .tsv)
	subs="$subs ${m#"$game"-}.pmf=$t"
done
# shellcheck disable=SC2086 # subs is a list of arguments
"$kfa" patch subtitles --boot-font "$gamedir/SYSDIR/BOOT.BIN" "$tmp/BOOT-sub.BIN" $subs
# Relocated chunks keep their pointers; the loader takes the base from the
# chunk table (docs/kfa1/0004).
"$kfa" patch chunkbase "$tmp/BOOT-sub.BIN" "$tmp/BOOT.BIN"

for f in "$gamedir"/USRDIR/Font/BitFont_Game*.bnd "$gamedir/USRDIR/Bnd/AdvMenu.bnd"; do
	"$kfa" font pad "$f" "$tmp/$(basename "$f")" >/dev/null
done

cp "$root/research/iso/$game.iso" "$iso.part"
"$kfa" iso put "$iso.part" PSP_GAME/USRDIR/GameData/01_UmdTitle/main.dat "$tmp/main.dat"
"$kfa" iso put "$iso.part" PSP_GAME/USRDIR/Bnd/AdvSys.bnd "$tmp/AdvSys.bnd"
"$kfa" iso put "$iso.part" PSP_GAME/SYSDIR/EBOOT.BIN "$tmp/BOOT.BIN"
for f in "$tmp"/BitFont_Game*.bnd; do
	"$kfa" iso put "$iso.part" "PSP_GAME/USRDIR/Font/$(basename "$f")" "$f" >/dev/null
done
"$kfa" iso put "$iso.part" PSP_GAME/USRDIR/Bnd/AdvMenu.bnd "$tmp/AdvMenu.bnd"
"$kfa" dat verify "$gamedir/SYSDIR/BOOT.BIN" "$tmp/main.dat"
mv "$iso.part" "$iso"
echo "built $iso"
