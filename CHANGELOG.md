# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Every entry below was reconstructed from git history on 2026-08-24, so they
record what shipped rather than what was written down at the time.

## [Unreleased]

## [0.3.0] — 2026-09-01

### Added

- `antenne keys` command group for API key management (`list`, `create`, `revoke`).
  Mint secret backend keys and public browser keys, list keys with optional
  `--app` filtering, and revoke keys by ID.

## [0.2.0] — 2026-08-24

### Changed

- Breaking: `antenne login` takes an address as well as a password, with
  `--email` as the flag for it. The instance runs on porte now, which serves
  `/api/auth/login` and returns a token in the response body, so the CLI sends
  `Authorization: Bearer` instead of a session cookie. porte reads the cookie
  before the header and refuses a cookie-authenticated mutating request with no
  `X-Facile-CSRF`, so a cookie client would have read fine and failed every
  write.
- `login` asks `GET /api/auth/config` first, and reports whether a password
  login exists, whether single sign-on is mandatory, and which address to
  prefill. An instance that only federates now fails with an explanation
  instead of storing an empty session and reporting success.

### Added

- Releases publish a Homebrew formula to `FacileStudio/homebrew-tap`.

## [0.1.2] — 2026-08-22

### Fixed

- `antenne logout` clears a credential it cannot parse. `Clear()` returned the
  read error, so a `config.json` that failed to parse aborted the command and
  left the token on disk. The URL overrides moved so `logout` still honours
  `--url` and `ANTENNE_URL`.
- The config file's mode is re-asserted on every write. `os.WriteFile` applies
  its perm argument only when creating the file, so rewriting a config that
  predated the 0600 rule left it group- and world-readable. A file found with
  any group or other bit set is tightened through the already open handle.

### Added

- `integrations/SKILL.md`, so an AI agent can drive the CLI.

### Changed

- `install.sh` bootstraps `facile` from `get.facile.studio` and falls back to
  the raw GitHub URL, so one unreachable host cannot break the install command.
- The README installs via `facile` and drops the paths that write somewhere
  other than `~/.local/bin`.

## [0.1.1] — 2026-08-10

### Changed

- Breaking: `nook-cli` became `antenne-cli`. The binary is `antenne`, the
  module is `github.com/FacileStudio/antenne-cli`, the config directory is
  `~/.config/antenne`, the `pool` subcommand is `bus`, and the default instance
  is `antenne.facile.studio`.
- `install.sh` is a shim that bootstraps `facile` and calls
  `facile install antenne`, replacing a 273-line template that had already
  drifted across the repos meant to share it byte for byte.

### Fixed

- Two implicit links to the old server name survived the rename and broke
  login: the routes are `/api/antenne/*`, and the session cookie is
  `antenne_session`.

## [0.1.0] — 2026-08-06

### Added

- First release. Reads an instance's alert log, follows it live with `tail`,
  and exercises delivery targets without opening the dashboard. Go and cobra.
- The client reads configuration and never writes it. Settings are one document
  replaced wholesale on save with secrets redacted on the way out, so a partial
  write would overwrite live credentials with the redaction marker.
- `tail` treats a status that means "not there right now" (404, 502, 503, 504,
  or a transport error) as a restart: one warning, then quiet retries until the
  stream returns. Anything else is reported verbatim, and a rejected session
  still aborts rather than hammering the login rate limiter.

[Unreleased]: https://github.com/FacileStudio/antenne-cli/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/FacileStudio/antenne-cli/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/FacileStudio/antenne-cli/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/FacileStudio/antenne-cli/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/FacileStudio/antenne-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/FacileStudio/antenne-cli/releases/tag/v0.1.0
