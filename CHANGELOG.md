# Changelog

## [v0.3.0] - 2026-10-23

### Added

- Show app name, version and GitHub URL in the help view (#87).
- Add a Log out button to the help view. It stops the timeline, removes the
  session tokens from `~/.kite.toml` and ends the session on the server (#88).

### Changed

- Bump go-gui v0.86.0 to v0.92.0 (#80, #81, #82, #83, #84, #85, #86).
- Pin golangci-lint in a `tools/lint` module (#79).

## [v0.2.1] - 2026-10-03

### Fixed

- Fix change font size to use window theme (`w.Theme()`).

### Changed

- Bump go-gui v0.78.0 to v0.84.0 (#72, #73, #74, #75, #76, #77).
- Migrate to Spacing/Radius/Border roles (#75).
- Bump go-glyph to v1.26.1, purego to v0.11.1.
