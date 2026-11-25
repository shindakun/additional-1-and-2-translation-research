#!/bin/sh
# Usage: scripts/extract_iso.sh
# Unzips both discs into research/iso and extracts their file trees into research/extract.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$root/research/iso" "$root/research/listings"
cd "$root/research/iso"
if [ ! -f kfaI.iso ]; then
	unzip -d . "$root/King's Field - Additional I (Japan, Asia).zip"
	mv "King's Field - Additional I (Japan, Asia).iso" kfaI.iso
fi
if [ ! -f kfaII.iso ]; then
	unzip -d . "$root/King's Field - Additional II (Asia).zip"
	mv "King's Field - Additional II (Asia).iso" kfaII.iso
fi
for g in kfaI kfaII; do
	7z x -y -bso0 -bsp0 -o"$root/research/extract/$g" "$g.iso"
	7z l -slt "$g.iso" >"$root/research/listings/$g.slt.txt"
done
shasum -a 1 kfaI.iso kfaII.iso
