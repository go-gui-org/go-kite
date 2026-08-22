#!/bin/sh
# Regenerate every icon artifact from kite.svg and kite-small.svg.
# Invoked by `make icons`. See README.md for the why behind each rule.
#
# POSIX sh on purpose: make runs recipes with /bin/sh, and this script is
# dev-only tooling that already depends on macOS' iconutil, so there is no
# reason to also require fish here.
set -eu

cd "$(dirname "$0")"

FULL=kite.svg        # full artwork, used at >=128px
SMALL=kite-small.svg # simplified mark, used at <=64px
HICOLOR=../icons/hicolor
BUILD="${TMPDIR:-/tmp}/go-kite-icons.$$"
trap 'rm -rf "$BUILD"' EXIT

for tool in rsvg-convert magick; do
	command -v "$tool" >/dev/null 2>&1 || {
		echo "$0: $tool not found (brew install librsvg imagemagick)" >&2
		exit 1
	}
done

mkdir -p "$BUILD"

# src_for SIZE -- which master a given pixel size renders from. The split is
# the whole point of shipping two drawings; see README.md "Two marks".
src_for() {
	if [ "$1" -le 64 ]; then echo "$SMALL"; else echo "$FULL"; fi
}

# png8 FILE -- normalise to 8-bit, strip metadata, max compression. A 16-bit
# master inflates the .icns several-fold for no visible gain.
png8() {
	magick "$1" -depth 8 -strip -define png:compression-level=9 "$1"
}

# bleed SIZE OUT -- full-bleed render, artwork edge to edge. What Windows and
# Linux want; they have no equivalent of Apple's icon grid.
bleed() {
	rsvg-convert -w "$1" -h "$1" -o "$2" "$(src_for "$1")"
	png8 "$2"
}

# inset SIZE OUT -- Big Sur grid: artwork at 824/1024 of the slot, centred on
# a transparent canvas of the full slot size. Rendering the vector at the art
# size and padding out beats downscaling the 1024 master; it is visibly
# sharper at 16-48px.
inset() {
	art=$(awk -v s="$1" 'BEGIN { printf "%d", int(s * 824 / 1024 + 0.5) }')
	rsvg-convert -w "$art" -h "$art" -o "$BUILD/inset.png" "$(src_for "$1")"
	magick "$BUILD/inset.png" -background none -gravity center \
		-extent "$1x$1" -depth 8 -strip \
		-define png:compression-level=9 "$2"
}

echo "==> masters"
bleed 1024 kite.png

# kite-dock.png is a separate file from kite.png, so skipping this step
# silently keeps the old artwork embedded in the binary. Inset to match the
# .icns slots, so the Dock tile lines up with its neighbours.
inset 1024 "$BUILD/dock1024.png"
magick "$BUILD/dock1024.png" -resize 512x512 -depth 8 -strip \
	-define png:compression-level=9 kite-dock.png

echo "==> kite.icns"
# The ten slots macOS reads. @2x names carry twice the pixels of their base.
ICONSET="$BUILD/kite.iconset"
mkdir -p "$ICONSET"
inset 16 "$ICONSET/icon_16x16.png"
inset 32 "$ICONSET/icon_16x16@2x.png"
inset 32 "$ICONSET/icon_32x32.png"
inset 64 "$ICONSET/icon_32x32@2x.png"
inset 128 "$ICONSET/icon_128x128.png"
inset 256 "$ICONSET/icon_128x128@2x.png"
inset 256 "$ICONSET/icon_256x256.png"
inset 512 "$ICONSET/icon_256x256@2x.png"
inset 512 "$ICONSET/icon_512x512.png"
inset 1024 "$ICONSET/icon_512x512@2x.png"
iconutil -c icns -o kite.icns "$ICONSET"

echo "==> kite.ico"
ICO_SIZES="16 24 32 48 64 128 256"
ICO_PNGS=""
for size in $ICO_SIZES; do
	bleed "$size" "$BUILD/ico-$size.png"
	ICO_PNGS="$ICO_PNGS $BUILD/ico-$size.png"
done
# shellcheck disable=SC2086 -- word splitting is the argument list here.
magick $ICO_PNGS kite.ico

echo "==> hicolor tree"
for size in 16 32 48 64 128 256 512; do
	mkdir -p "$HICOLOR/${size}x${size}/apps"
	bleed "$size" "$HICOLOR/${size}x${size}/apps/go-kite.png"
done
mkdir -p "$HICOLOR/scalable/apps"
cp "$FULL" "$HICOLOR/scalable/apps/go-kite.svg"

echo "==> windows resources"
# Linked automatically by `go build` under GOOS=windows and ignored on every
# other platform, so the bare .exe carries its own icon with no installer.
for arch in amd64 arm64; do
	go run github.com/akavel/rsrc@v0.10.2 \
		-ico kite.ico -arch "$arch" -o "../../rsrc_windows_$arch.syso"
done

echo "done"
