#!/bin/sh
# Usage: scripts/movie_sheets.sh
# Writes contact sheets (one frame per 2 s, 8x8 tiles) of every PMF movie to research/movies.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$root/research/movies"
for g in kfaI kfaII; do
	for f in "$root/research/extract/$g/PSP_GAME/USRDIR/movie/"*.pmf; do
		b=$(basename "$f" .pmf)
		# PSMF has a 0x800-byte header; the rest is MPEG-PS with H.264 video.
		tail -c +2049 "$f" | ffmpeg -v error -f mpeg -i - -an \
			-vf "fps=1/2,scale=240:136,tile=8x8" -frames:v 4 -y "$root/research/movies/${g}_${b}_%d.png"
	done
done
