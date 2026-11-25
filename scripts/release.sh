#!/bin/sh
# Usage: scripts/release.sh GAME VERSION [--publish]
# Prepares a GitHub release of the English patch for GAME (kfaI or kfaII).
#
# 1. Checks that the work tree is clean and HEAD is pushed.
# 2. Uses dist/GAME-english-VERSION/ if its readme names HEAD; otherwise
#    builds it with scripts/make_patch.sh.
# 3. Without --publish, prints the tag, title and files and stops.
#    With --publish, creates and pushes the tag GAME-vVERSION and creates
#    the release with gh, with the readme as notes and the patch and readme
#    as files.
# Needs gh (logged in) and xdelta3. See docs/kfa1/0009-test-builds.md.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
game=$1 version=$2 publish=${3:-}
name="$game-english-$version"
dir="$root/dist/$name"
tag="$game-v$version"
cd "$root"

if [ -n "$(git status --porcelain)" ]; then
	echo "release: the work tree has changes; commit them first" >&2
	exit 1
fi
git fetch -q origin
head=$(git rev-parse --short HEAD)
if [ "$(git rev-parse HEAD)" != "$(git rev-parse '@{u}')" ]; then
	echo "release: HEAD is not the same as its upstream; push first" >&2
	exit 1
fi
if git rev-parse -q --verify "refs/tags/$tag" >/dev/null || git ls-remote --exit-code --tags origin "$tag" >/dev/null 2>&1; then
	echo "release: tag $tag exists" >&2
	exit 1
fi

if ! grep -q "^Commit: $head\$" "$dir/README.txt" 2>/dev/null; then
	scripts/make_patch.sh "$game" "$version"
fi

case $game in
kfaI) title="King's Field Additional I English patch $version" ;;
kfaII) title="King's Field Additional II English patch $version" ;;
*) title="$game English patch $version" ;;
esac

echo "tag:   $tag (commit $head)"
echo "title: $title"
echo "files: $dir/$name.xdelta $dir/README.txt"
if [ "$publish" != "--publish" ]; then
	echo "dry run; add --publish to tag and release"
	exit 0
fi

git tag -a "$tag" -m "$title"
git push origin "$tag"
gh release create "$tag" --title "$title" --notes-file "$dir/README.txt" --prerelease \
	"$dir/$name.xdelta" "$dir/README.txt"
