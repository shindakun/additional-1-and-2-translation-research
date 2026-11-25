#!/bin/sh
# Usage: scripts/make_patch.sh GAME VERSION
# Builds research/builds/GAME-english.iso (scripts/build_english.sh) and
# writes dist/GAME-english-VERSION/ with an xdelta3 (VCDIFF) patch from the
# original ISO, a readme and SHA-256 sums. The script applies the patch to a
# copy of the original and checks that the result is the built ISO byte for
# byte. The 1 GB source window (-B) lets xdelta3 find the files that move
# to the end of the image. Needs xdelta3. See docs/kfa1/0009-test-builds.md.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=$1 version=$2
orig="$root/research/iso/$game.iso"
built="$root/research/builds/$game-english.iso"
name="$game-english-$version"
out="$root/dist/$name"

"$root/scripts/build_english.sh" "$game"
rm -rf "$out"
mkdir -p "$out"
xdelta3 -e -9 -f -B 1073741824 -s "$orig" "$built" "$out/$name.xdelta"
xdelta3 -d -f -B 1073741824 -s "$orig" "$out/$name.xdelta" "$out/check.iso"
cmp "$out/check.iso" "$built"
rm "$out/check.iso"

sum() { shasum -a 256 "$1" | cut -d' ' -f1; }
case $game in
kfaI) title="King's Field Additional I (ULJS-00070)" ;;
kfaII) title="King's Field Additional II (ULJS-00076)" ;;
*) title=$game ;;
esac
cat >"$out/README.txt" <<EOF
$title English patch, version $version
Commit: $(git -C "$root" rev-parse --short HEAD)

Apply to the original ISO with xdelta3 (or Delta Patcher):

  xdelta3 -d -B 1073741824 -s ORIGINAL.iso $name.xdelta $name.iso

Original ISO SHA-256: $(sum "$orig")
Patched ISO SHA-256:  $(sum "$built")
Patch SHA-256:        $(sum "$out/$name.xdelta")

The English is a first draft. The patch changes main.dat (text, images),
AdvSys.bnd (system messages), the fonts and EBOOT.BIN (movie subtitles and
the loader change for longer text). It runs on PPSSPP; on PSP hardware it
needs custom firmware to run the unencrypted EBOOT.BIN.
EOF
echo "wrote $out"
ls -l "$out"
