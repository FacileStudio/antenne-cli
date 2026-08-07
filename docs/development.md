# antenne-cli — Development

Local setup, the quality gate, and how a release happens.

## Prerequisites

Go 1.25, pinned in `mise.toml`. Nothing else — there is no client to build and no database
to run.

## Setup

```sh
mise run hooks     # enable the tracked git hooks in this clone
mise run build     # ./bin/antenne
```

## Running against an instance

The fastest loop is a locally running Antenne, which the default URL already points at:

```sh
go run . login              # http://localhost:9090
go run . status
```

Against a deployed one, `--url` avoids touching the stored config:

```sh
go run . status --url https://antenne.facile.studio
```

A second instance wants a second config directory rather than repeated `--url`, because the
session is per-instance — see [configuration.md](configuration.md).

## The quality gate

```sh
mise run check     # gofmt, vet, go test
mise run format    # rewrite Go sources in place
```

`scripts/check.sh` is invoked directly rather than through `mise` inside itself: `mise run`
resolves every tool in the merged config before running a task body, so an unrelated broken
tool in a global config would take the gate down with it. The tracked `pre-push` hook runs
the same script.

## Tests

| File | What it pins |
|---|---|
| `internal/client/sse_test.go` | the event-stream grammar: framing, keep-alive comments, multi-line data, a truncated stream |
| `internal/client/client_test.go` | the session cookie, the error envelope, an open instance's empty token, and HTML reported as a wrong URL |

```sh
go test ./...
```

The client tests run against `httptest` servers, so the suite needs no network and no
instance.

## Adding a command

One file per command in `cmd/`, registered from its own `init`. Follow
[`Wiki/CLI-STANDARD.md`](https://github.com/FacileStudio/Antenne) §3 and §4:

- `Short` is capitalized, imperative, no trailing period. So is every flag's help.
- Output goes through `internal/ui` — never `fmt.Println` for status.
- Data on stdout, everything else on stderr.
- A command carrying data supports `--json`, printing one document and nothing else.
- No emoji, anywhere.

## Releasing

Tag and push; GoReleaser builds darwin and linux for amd64 and arm64, publishes the archives
with a `checksums.txt`, and pushes the Homebrew formula to `FacileStudio/homebrew-tap`.

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The tap push needs `HOMEBREW_TAP_GITHUB_TOKEN` in the workflow environment — `GITHUB_TOKEN`
cannot write to another repository, which is the failure `capsule-cli` is recorded as
hitting.

`install.sh` resolves the latest tag by following the redirect from `/releases/latest`
rather than calling the GitHub API, so it needs no token and has no rate limit. Until a tag
exists it falls back to building from source, which works on a clean checkout.
