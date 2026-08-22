#!/bin/sh
# Build distributable packages for one platform. Invoked by `make dist-*`.
#
#   packaging/package.sh macos      -> dist/Kite-<ver>-macos-<arch>.dmg
#   packaging/package.sh linux      -> dist/go-kite-<ver>-linux-<arch>.tar.gz
#                                      dist/go-kite_<ver>_<arch>.deb
#   packaging/package.sh windows    -> dist/go-kite-<ver>-windows-<arch>.zip
#
# POSIX sh on purpose, matching assets/icon/generate.sh: make runs recipes
# with /bin/sh and this is dev-only tooling.
#
# Linux and Windows cross-build from any host because go-gui's gl backend is
# CGo-free on both -- it dlopens libEGL/opengl32 through purego, and go-glyph
# supplies a pure-Go text pipeline when cgo is off (go-gui
# docs/specs/cgo-free-backend-feasibility.md). macOS is the exception: its
# backend is 5.9k lines of ObjC, so a .dmg can only be built on macOS.
set -eu

cd "$(dirname "$0")/.."

DIST=dist
BIN=go-kite
APP=Kite

# Release builds pin to go.mod, never to a go.work pointing at sibling
# go-gui/go-glyph checkouts. A shipped artifact must be built from the
# versions go.sum records, not from whatever happens to be checked out.
export GOWORK=off

# VERSION defaults to the git description. Override for a real release:
#   make dist VERSION=1.2.0
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 0.0.0)}"
VERSION="${VERSION#v}"

# Architectures for the cross-compiled platforms. macOS ignores this and
# builds host-native only.
ARCHES="${ARCHES:-amd64 arm64}"

# Debian policy requires a version starting with a digit. A bare commit
# description ("ad81dac") does not qualify, so untagged builds get a 0.0.0+git
# prefix that sorts below every real release.
case "$VERSION" in
	[0-9]*) DEB_VERSION="$VERSION" ;;
	*) DEB_VERSION="0.0.0+git$VERSION" ;;
esac
# ':' and '-' carry meaning in a Debian version (epoch, revision); '+' does not.
DEB_VERSION=$(printf '%s' "$DEB_VERSION" | tr ':-' '++')

MAINTAINER="${MAINTAINER:-$(git config user.name 2>/dev/null || echo unknown) <$(git config user.email 2>/dev/null || echo unknown)>}"

mkdir -p "$DIST"

# --------------------------------------------------------------------------
# helpers
# --------------------------------------------------------------------------

# linuxbin ARCH -- build (once per run) and echo the path to the linux binary
# for ARCH. The tarball and the .deb ship the identical file; building it
# twice would double the wall clock for no benefit and, worse, let the two
# artifacts diverge if the tree changed mid-run.
linuxbin() {
	out="$DIST/stage/bin-linux-$1"
	if [ ! -f "$out" ]; then
		mkdir -p "$DIST/stage"
		gobuild linux "$1" "$out" >&2
	fi
	echo "$out"
}

# gobuild GOOS GOARCH OUT -- same flags as the Makefile's $(KITE_BIN) rule.
# CGO_ENABLED=0 is explicit rather than inherited from the cross-compile
# default, because it is also what we want for a *native* Linux build: no
# glibc link means one binary runs on every distro.
gobuild() {
	echo "  build $1/$2"
	CGO_ENABLED=0 GOOS="$1" GOARCH="$2" \
		go build -tags=prod -trimpath -ldflags="-s -w" -o "$3" .
}

# tarball OUT CDIR MEMBER -- tar.gz of MEMBER relative to CDIR, owned by
# root:root. bsdtar (macOS) and GNU tar (Linux) spell every one of those flags
# differently, so the two cases cannot be collapsed. gnutar format because
# bsdtar defaults to pax extended headers, which older dpkg refuses.
tarball() {
	if tar --version 2>&1 | head -1 | grep -q bsdtar; then
		COPYFILE_DISABLE=1 tar --format gnutar \
			--uid 0 --gid 0 --uname root --gname root \
			-czf "$1" -C "$2" "$3"
	else
		tar --format=gnu --owner=0 --group=0 \
			-czf "$1" -C "$2" "$3"
	fi
}

# icontree DESTDIR -- copy the committed hicolor tree and .desktop file into
# a freedesktop layout rooted at DESTDIR (which is /usr for a package, or the
# tarball's share/ parent).
icontree() {
	mkdir -p "$1/share/applications"
	cp assets/go-kite.desktop "$1/share/applications/go-kite.desktop"
	mkdir -p "$1/share/icons"
	cp -R assets/icons/hicolor "$1/share/icons/hicolor"
}

# --------------------------------------------------------------------------
# macos
# --------------------------------------------------------------------------

package_macos() {
	[ "$(uname -s)" = "Darwin" ] || {
		echo "$0: macOS packages can only be built on macOS (the backend is cgo/ObjC)" >&2
		exit 1
	}
	command -v hdiutil >/dev/null 2>&1 || {
		echo "$0: hdiutil not found" >&2
		exit 1
	}
	[ -d "$APP.app" ] || {
		echo "$0: $APP.app not found -- run 'make all' first" >&2
		exit 1
	}

	arch=$(uname -m)
	case "$arch" in x86_64) arch=amd64 ;; esac
	out="$DIST/$APP-$VERSION-macos-$arch.dmg"

	stage="$DIST/dmg-stage"
	rm -rf "$stage"
	mkdir -p "$stage"
	# Copy with -R to preserve the bundle's symlinks and the code signature.
	cp -R "$APP.app" "$stage/$APP.app"
	# The conventional drag-to-install affordance. Nothing enforces it; a
	# .dmg without it just makes the user find /Applications themselves.
	ln -s /Applications "$stage/Applications"
	cp LICENSE "$stage/LICENSE" 2>/dev/null || true

	rm -f "$out"
	# UDZO = zlib-compressed read-only image, the format every macOS release
	# ships. -ov because a stale image of the same name would otherwise abort.
	hdiutil create -quiet -volname "$APP $VERSION" -srcfolder "$stage" \
		-ov -format UDZO "$out"
	rm -rf "$stage"
	echo "  $out"
}

# --------------------------------------------------------------------------
# linux
# --------------------------------------------------------------------------

# linux_tarball ARCH -- relocatable tree plus an install.sh, for distros with
# no .deb. Everything sits under a single top-level directory so an unpack in
# the wrong place is trivially undone.
linux_tarball() {
	name="$BIN-$VERSION-linux-$1"
	stage="$DIST/stage/$name"
	rm -rf "$stage"
	mkdir -p "$stage"

	cp "$(linuxbin "$1")" "$stage/$BIN"
	chmod 755 "$stage/$BIN"
	icontree "$stage"
	cp README.md "$stage/README.md" 2>/dev/null || true
	cp LICENSE "$stage/LICENSE" 2>/dev/null || true

	# Written here rather than committed so the paths cannot drift from what
	# this script actually stages.
	cat > "$stage/install.sh" <<'INSTALL'
#!/bin/sh
# Install Kite for the current user (default) or system-wide (PREFIX=/usr/local).
set -eu
cd "$(dirname "$0")"
PREFIX="${PREFIX:-$HOME/.local}"

install -Dm755 go-kite "$PREFIX/bin/go-kite"
install -Dm644 share/applications/go-kite.desktop \
	"$PREFIX/share/applications/go-kite.desktop"
# cp -R, not install: the hicolor tree is a directory hierarchy.
mkdir -p "$PREFIX/share/icons"
cp -R share/icons/hicolor "$PREFIX/share/icons/"

# Menus and taskbars read a cached index; a fresh icon is invisible until it
# is rebuilt. Both tools are optional -- absent, the caches refresh on login.
command -v gtk-update-icon-cache >/dev/null 2>&1 &&
	gtk-update-icon-cache -q "$PREFIX/share/icons/hicolor" || true
command -v update-desktop-database >/dev/null 2>&1 &&
	update-desktop-database -q "$PREFIX/share/applications" || true

echo "installed to $PREFIX"
case ":$PATH:" in
	*":$PREFIX/bin:"*) ;;
	*) echo "warning: $PREFIX/bin is not on PATH" >&2 ;;
esac
INSTALL
	chmod +x "$stage/install.sh"

	tarball "$DIST/$name.tar.gz" "$DIST/stage" "$name"
	rm -rf "$stage"
	echo "  $DIST/$name.tar.gz"
}

# linux_deb ARCH -- a .deb assembled by hand from ar + two tars.
#
# dpkg-deb is not used, and not required: a .deb *is* an ar archive holding
# debian-binary, control.tar.gz and data.tar.gz in that order. Building it
# directly is what lets a macOS host produce Debian packages at all, since
# dpkg is not installable there as a matter of course.
linux_deb() {
	arch=$1
	root="$DIST/stage/deb-$arch"
	rm -rf "$root"
	mkdir -p "$root/data/usr/bin" "$root/control"

	cp "$(linuxbin "$arch")" "$root/data/usr/bin/$BIN"
	chmod 755 "$root/data/usr/bin/$BIN"
	icontree "$root/data/usr"
	mkdir -p "$root/data/usr/share/doc/$BIN"
	cp LICENSE "$root/data/usr/share/doc/$BIN/copyright" 2>/dev/null || true

	# Installed-Size is in KiB and is what apt reports before installing.
	size=$(du -sk "$root/data" | awk '{print $1}')

	# libegl1 is the *only* shared-library dependency: the binary is static
	# (CGO_ENABLED=0) and the backend dlopens libEGL.so.1 by soname at
	# startup. No libc, no X11 client libs -- the X11 transport is pure Go.
	cat > "$root/control/control" <<CONTROL
Package: $BIN
Version: $DEB_VERSION
Section: net
Priority: optional
Architecture: $arch
Depends: libegl1
Installed-Size: $size
Maintainer: $MAINTAINER
Homepage: https://github.com/go-gui-org/go-kite
Description: Desktop Bluesky client
 Kite is a lightweight desktop client for Bluesky, showing a live timeline
 with inline images. Built on go-gui.
CONTROL

	# md5sums is optional but `dpkg -V` and debsums are useless without it.
	# Format is "hash<space><space>path" with no leading "./" -- the sed
	# normalises both GNU md5sum and BSD md5 -r output to exactly that.
	(cd "$root/data" && find . -type f -exec md5sum {} + |
		sed 's| \./| |' > "../control/md5sums") 2>/dev/null ||
		(cd "$root/data" && find . -type f -exec md5 -r {} + |
			sed 's| \./| |' > "../control/md5sums")

	# Members are packed as "./usr/bin/go-kite" etc -- rooted at the archive,
	# not under a top-level directory, which is what dpkg expects.
	tarball "$root/data.tar.gz" "$root/data" .
	tarball "$root/control.tar.gz" "$root/control" .

	echo '2.0' > "$root/debian-binary"

	out="$DIST/${BIN}_${DEB_VERSION}_${arch}.deb"
	rm -f "$out"
	# Member order is load-bearing: dpkg reads debian-binary first and
	# rejects an archive that leads with anything else. `ar` appends in
	# argument order, and none of the three names exceeds ar's 16-character
	# short-name limit, so no BSD/GNU long-name extension is involved.
	#
	# The S is not optional on macOS. Without it BSD ar runs ranlib and
	# prepends a "__.SYMDEF SORTED" member -- in BSD long-name format, no
	# less -- which makes debian-binary the second member and the whole
	# archive unreadable to dpkg. It fails silently at build time; the only
	# symptom is dpkg rejecting the package on the user's machine.
	abs_out="$PWD/$out"
	(cd "$root" && ar rcS "$abs_out" debian-binary control.tar.gz data.tar.gz)
	rm -rf "$root"
	echo "  $out"
}

package_linux() {
	for arch in $ARCHES; do
		linux_tarball "$arch"
		linux_deb "$arch"
	done
	rm -f "$DIST"/stage/bin-linux-*
}

# --------------------------------------------------------------------------
# windows
# --------------------------------------------------------------------------

# No installer: an MSI needs WiX (Windows-only), and the .exe already carries
# its own icon from rsrc_windows_<arch>.syso, so a zip is a complete product.
package_windows() {
	command -v zip >/dev/null 2>&1 || {
		echo "$0: zip not found" >&2
		exit 1
	}
	for arch in $ARCHES; do
		name="$BIN-$VERSION-windows-$arch"
		stage="$DIST/stage/$name"
		rm -rf "$stage"
		mkdir -p "$stage"

		gobuild windows "$arch" "$stage/$BIN.exe"
		cp README.md "$stage/README.md" 2>/dev/null || true
		cp LICENSE "$stage/LICENSE.txt" 2>/dev/null || true

		out="$DIST/$name.zip"
		rm -f "$out"
		# -X drops the .DS_Store-adjacent extra fields macOS' zip otherwise
		# writes, which Windows Explorer surfaces as junk files.
		(cd "$DIST/stage" && zip -q -r -X "../$name.zip" "$name")
		rm -rf "$stage"
		echo "  $out"
	done
}

# --------------------------------------------------------------------------

case "${1:-}" in
	macos) echo "==> macOS package $VERSION"; package_macos ;;
	linux) echo "==> Linux packages $VERSION"; package_linux ;;
	windows) echo "==> Windows packages $VERSION"; package_windows ;;
	*)
		echo "usage: $0 macos|linux|windows" >&2
		exit 2
		;;
esac

rmdir "$DIST/stage" 2>/dev/null || true

# One checksum file covering everything present, regenerated on each run so it
# never describes an artifact that has since been rebuilt.
# Written to a temp file first: the redirect would otherwise create
# SHA256SUMS before find enumerates the directory. `sed` strips find's "./"
# so `shasum -c SHA256SUMS` works from inside dist/.
sums() {
	# The ! -name tests exclude the output file from its own input.
	# shellcheck disable=SC2094
	find . -maxdepth 1 -type f ! -name SHA256SUMS.tmp ! -name SHA256SUMS |
		sort | xargs "$@" | sed 's|  \./|  |' > SHA256SUMS.tmp
	mv SHA256SUMS.tmp SHA256SUMS
}
# shasum on macOS, sha256sum on Linux.
(cd "$DIST" && sums shasum -a 256) 2>/dev/null || (cd "$DIST" && sums sha256sum)
