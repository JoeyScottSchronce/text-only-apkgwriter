# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `LICENSE` (MIT, SPDX `MIT`).
- This changelog.
- GitHub Actions CI (`.github/workflows/ci.yml`: `go vet`, `go test`).

### Changed

- README: “Resource use and large decks”, “Tests”; `WriteApkg` godoc describes temp SQLite, full read into memory, and lack of streaming.
- Expanded tests: `SanitizeDeckTitle`, `GuidFor` (incl. zero-hash branch), `WriteApkg` (zip entries, zero-card error).

## [0.1.0] - 2026-03-21

### Added

- Initial release: `WriteApkg` (text-only `.apkg` with `collection.anki2` + `media`), `SanitizeDeckTitle`, `GuidFor`, and SQLite schema helpers.
