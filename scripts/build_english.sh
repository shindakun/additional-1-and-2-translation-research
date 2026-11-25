#!/bin/sh
# Usage: scripts/build_english.sh [GAME]
# Builds research/builds/<GAME>-english.iso (default GAME kfaI) with every
# current translation: main.dat text and images (docs/kfa1/0004, docs/kfa1/0012), FMG
# messages (docs/kfa1/0013), movie subtitles with the boot font (docs/kfa1/0010) and
# fonts with padded ASCII glyphs (docs/kfa1/0005).
# Inputs: translations/GAME (text.json, fmg.json, subs/, img/), which hold no
# Japanese, and the disc extracted by scripts/extract_iso.sh. The script
# builds bin/kfa, exports the Japanese from the disc and fills it in
# (scripts/translation.py), and redraws the image labels from the specs in
# translations/GAME/img (scripts/relabel_jpeg.py; needs Pillow and the
# fonts named in the specs).
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=${1:-kfaI}
kfa="$root/bin/kfa"
gamedir="$root/research/extract/$game/PSP_GAME"
out="$root/research/builds"
tmp="$out/.tmp-$game"
iso="$out/$game-english.iso"
tr="$root/translations/$game"
mkdir -p "$tmp/img" "$tmp/edit"
trap 'rm -rf "$tmp"' EXIT

(cd "$root" && go build -o "$kfa" ./cmd/kfa)
"$kfa" text export "$gamedir" "$tmp/export.json" >/dev/null
python3 -I "$root/scripts/translation.py" fill "$tmp/export.json" "$tr/text.json" "$tmp/text.json" >/dev/null
"$kfa" img export "$gamedir" "$tmp/img" >/dev/null
while IFS="$(printf '\t')" read -r spec image; do
	# -P: Pillow comes from the user site; run from the temporary directory.
	(cd "$tmp" && python3 -P "$root/scripts/relabel_jpeg.py" "$tr/img/$spec" "$tmp/img/$image" "$tmp/edit/$image" >/dev/null)
done <"$tr/img/images.tsv"

"$kfa" img import "$gamedir" "$tmp/edit" "$tmp/img.dat"
"$kfa" text import "$gamedir" "$tmp/text.json" "$tmp/main.dat" "$tmp/img.dat" 2>"$tmp/text.log" >/dev/null
grep -c '^not written' "$tmp/text.log" >/dev/null && sed -n '/^not written/p' "$tmp/text.log" >&2
"$kfa" fmg import "$gamedir" "$tr/fmg.json" "$tmp/AdvSys.bnd"
subs=""
for t in "$tr"/subs/movie*.tsv; do
	[ -e "$t" ] || continue
	subs="$subs $(basename "$t" .tsv).pmf=$t"
done
# shellcheck disable=SC2086 # subs is a list of arguments
if [ -n "$subs" ]; then
	"$kfa" patch subtitles --boot-font "$gamedir/SYSDIR/BOOT.BIN" "$tmp/BOOT-sub.BIN" $subs
else
	cp "$gamedir/SYSDIR/BOOT.BIN" "$tmp/BOOT-sub.BIN"
fi
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
