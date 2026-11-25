#!/bin/sh
# Usage: scripts/extract_iso.sh
# Unzips the discs into research/iso and extracts their file trees into research/extract.
# Adventure Player (ap) is optional; it is extracted when its archive is present.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$root/research/iso" "$root/research/listings"
cd "$root/research/iso"

unpack() { # name, archive, iso inside the archive
	if [ ! -f "$1.iso" ]; then
		unzip -d . "$root/$2"
		mv "$3" "$1.iso"
	fi
	7z x -y -bso0 -bsp0 -o"$root/research/extract/$1" "$1.iso"
	7z l -slt "$1.iso" >"$root/research/listings/$1.slt.txt"
	shasum -a 1 "$1.iso"
}

unpack kfaI "King's Field - Additional I (Japan, Asia).zip" "King's Field - Additional I (Japan, Asia).iso"
unpack kfaII "King's Field - Additional II (Asia).zip" "King's Field - Additional II (Asia).iso"
if [ -f "$root/Adventure Player (Japan).zip" ]; then
	unpack ap "Adventure Player (Japan).zip" "Adventure Player (Japan).iso"
fi
