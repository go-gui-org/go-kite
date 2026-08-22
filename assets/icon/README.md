# Kite app icon

| File             | Use                                                                             |
| ---------------- | ------------------------------------------------------------------------------- |
| `../logo.svg`    | The artistic master. Nothing is rendered from it directly — see below.          |
| `kite.svg`       | **Source of truth** for every artifact at ≥128px. Derived from `logo.svg`.      |
| `kite-small.svg` | Source of truth for every artifact at ≤64px. Derived from `kite.svg`.           |
| `kite.icns`      | macOS bundle icon. `make all` passes it to `buildapp -icon`.                    |
| `kite.ico`       | Windows icon, and the input to the `.syso` resources.                           |
| `kite.png`       | 1024×1024 full-bleed render. Reference master, not consumed by a build.         |
| `kite-dock.png`  | 512×512 padded art, embedded in the binary by `icon.go`. Read the next section. |
| `generate.sh`    | Regenerates everything above except the two SVGs. Run it via `make icons`.      |

`generate.sh` also writes two things outside this directory:
`../icons/hicolor/**` (Linux) and `../../rsrc_windows_*.syso` (the Windows
executable resource, built from `kite.ico`). `../go-kite.desktop` is
hand-written, not generated.

## The bundle icon alone is not enough

Shipping `kite.icns` in `Kite.app` and setting `CFBundleIconFile` does **not**
get the icon shown. The go-gui darwin backend calls
`-[NSApplication setApplicationIconImage:]` during window creation and falls
back to go-gui's own default icon when `gui.WindowCfg.IconPNG` is empty
(`gui/backend/metal/backend.go` as of go-gui v0.64.0). That runtime call takes
precedence over `CFBundleIconFile` for the running process, so an unset
`IconPNG` means Kite installs the _go-gui_ icon over its own on every launch,
from a correctly bundled build.

This is why `icon.go` embeds `kite-dock.png` and `kiteWindowCfg` sets `IconPNG`
from it. It is not redundant with the `.icns`; each covers a case the other
cannot:

- `.icns` / `CFBundleIconFile` — Finder, the Dock's persistent tile, Get Info.
  Applies only to a bundled build.
- `IconPNG` — the running process's application icon on macOS, the
  taskbar/alt-tab window icon on Windows and X11, and the only icon that exists
  at all under `go run .`, where there is no bundle.

Symptom if `IconPNG` regresses: the go-gui icon in `⌘Tab` and the Dock while
Finder still shows the kite. That reads exactly like a stale LaunchServices icon
cache — it isn't, and flushing the cache will not fix it. `TestWindowCfgWiring`
fails if the field is dropped.

## Why `logo.svg` is not rendered directly

`logo.svg` is the artwork as drawn. `kite.svg` is the same drawing with two
changes an icon needs and a logo does not:

1. **No drop shadow.** macOS composites its own Dock shadow; a baked one doubles
   up. Windows and Linux draw the bitmap flat, where the shadow just eats
   contrast at the edge.
2. **Full bleed.** `logo.svg` insets its squircle to `x=32…480` of a 512 canvas,
   leaving a 32px transparent margin. Icons are drawn into a fixed cell, so that
   margin renders Kite visibly smaller than every neighbouring icon. The
   `#bleed` group scales the artwork by 512/448 = 1.142857 about the canvas
   centre, which lands the squircle exactly on `0…512` and takes its corner
   radius to 114.3px (22.3%). It is a uniform scale, so nothing distorts.

Edit `logo.svg` for artwork changes, then re-apply those two edits to
`kite.svg`. They are small and deliberately kept as readable diffs rather than a
build step.

## Two marks

The full artwork carries a neon glow, four 1px accent polygons, three tail bows
and a centre butterfly. Below roughly 64px those stop being detail and become
noise: the glow reads as a halo rather than a highlight, the accent lines and
bows collapse to single grey pixels, and the butterfly merges into the crossbar.
So the icon ships as two marks, and `kite.icns` / `kite.ico` bind them to size
buckets:

- **≤64px** — `kite-small.svg`: four sails, spine, crossbar, one shortened tail.
  Opaque sails and thicker spars, both to hold contrast when the shape is a
  handful of pixels.
- **≥128px** — `kite.svg`: the full artwork.

macOS and Windows both pick the closest slot per context, so the Dock and the
Finder icon get the full art while the menu bar, tab bars and `⌘Tab` get the
legible one.

Unlike some hand-vectorized icon sets, `kite-small.svg` **is** derived from
`kite.svg` — legitimately, because the source is layered vector rather than a
traced raster. It deletes whole elements and scales the survivors; it does not
redraw them. Two numbers in it are computed rather than eyeballed, and should be
recomputed rather than nudged if the artwork moves:

- The tail is `kite.svg`'s tail cubic split at t=0.72 by de Casteljau, so it is
  the exact same arc, just shorter. The full-length tail runs off the tile once
  `#mark` is applied.
- `#mark`'s scale (1.22) and anchor (249.5, 256.5) are fitted to the rendered
  alpha bounding box: 72% fill with equal margins on all four sides. Re-fit by
  rendering with the background `<rect>` deleted and reading
  `magick out.png -trim -format '%wx%h+%X+%Y' info:`.

## Geometry

Both masters are full-bleed 512×512 with a squircle of radius 114.3 (22.3%).
Inside `kite.icns` each slot is padded to the Big Sur convention: artwork at
824/1024 of the slot, centred on a transparent canvas of the full slot size. No
drop shadow is baked in.

`kite.ico` and the hicolor PNGs are **not** padded that way — Windows and the
freedesktop icon spec have no equivalent of Apple's grid, and an inset icon
there just looks small.

## Regenerating

Needs `rsvg-convert` (`brew install librsvg`), ImageMagick 7
(`brew install imagemagick`), and macOS' `iconutil`. ImageMagick's built-in MSVG
renderer does not handle the `feGaussianBlur`/`feMerge` glow correctly, which is
why every SVG→PNG step goes through `rsvg-convert` and `magick` only ever
touches PNGs.

```fish
make icons
```

Each slot is rendered from the _vector_ at its art size and then padded out,
rather than downscaling the 1024 master — noticeably sharper at 16–48px. PNGs
are kept at `-depth 8`; a 16-bit master inflates the `.icns` several-fold for no
visible gain.

`TestAppIconDecodes` bounds `kite-dock.png` at 1024px per side and 512KB, so an
oversized replacement fails the build rather than quietly bloating every binary.

## Installing the Linux tree

```fish
install -Dm644 assets/go-kite.desktop /usr/share/applications/go-kite.desktop
cp -r assets/icons/hicolor /usr/share/icons/
gtk-update-icon-cache /usr/share/icons/hicolor
```

`StartupWMClass` in the `.desktop` file must match `gui.WindowCfg.WMClass` in
`main.go`; X11 taskbars pair a running window with its launcher entry by that
string, and a mismatch shows the generic fallback icon on the running window
while the menu entry looks right. `TestWindowCfgWiring` pins the Go side.
