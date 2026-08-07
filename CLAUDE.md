# antenne-cli

Terminal client for a [Antenne](https://github.com/FacileStudio/Antenne) instance. Go, cobra, one
binary named `antenne`.

## Commands

| Task | Command |
|---|---|
| Build | `mise run build` |
| Quality gate | `mise run check` |
| Format Go | `mise run format` |
| Enable git hooks | `mise run hooks` |

## Structure

```
main.go            hands off to cmd
cmd/               one file per command; root.go owns flags and exit codes,
                   format.go the rendering shared between them
internal/
  client/          the HTTP surface (client.go) and the SSE reader (sse.go)
  config/          the instance URL and the session token
  ui/              CLI-STANDARD §4 output vocabulary, copied verbatim
install.sh         generated from Wiki/install.sh.template; only the config block differs
```

Dependencies are cobra, `fatih/color` and `golang.org/x/term`. Adding a fourth needs a
reason — a client for one API does not need a framework.

## Conventions

These come from `Wiki/CLI-STANDARD.md`, which is normative. When this repo disagrees with
it, this repo is wrong.

- **`Short` and flag help: capitalized, imperative, no trailing period.** `"List projects"`,
  never `"Lists projects."`.
- **No emoji, anywhere.** Not in help, not at runtime.
- **All output through `internal/ui`.** `▸` step, `✓` success, `!` warning, `✗` error, and
  hints indented two spaces. Warnings and errors go to stderr.
- **Data on stdout, everything else on stderr**, so a piped command emits only data.
- **`--json` on every command carrying data**, printing one document and nothing else. It
  forces colour off.
- **Exit codes**: `0` success, `1` failure, `2` usage, `130` SIGINT. `root.go` maps them;
  `commandStarted` is what distinguishes a usage error from a failed one.
- Errors are lowercase, name what failed, and end with the fix after an em dash. The glyph
  is added by the printer, never baked into the message.
- `--version` prints exactly `antenne <semver>` — the installer parses that line.

## The client is read-only against settings

It reads configuration and never writes it. There is no `antenne providers add`.

Antenne's settings are one document replaced wholesale on save, with secrets redacted on the
way out and merged back on the way in. A CLI that PUT a partial document would have to
reproduce that merge exactly, and getting it wrong overwrites live credentials with the
redaction marker — which has happened once, from a script, and cost eleven of them.

## Traps worth not rediscovering

- **Antenne serves its dashboard on the same origin as its API**, so a wrong path returns `200`
  and HTML. The client reads bodies as text and reports that as a wrong URL rather than
  letting a JSON syntax error hide it.
- **The API authenticates by cookie only.** No bearer header, no key. `login` reads
  `Set-Cookie` and every later call sends `Cookie: nook_session=…`.
- **An instance with no `ANTENNE_ADMIN_PASSWORD` serves everyone as the admin** and sets no
  cookie. That is a success with an empty token, not a failure.
- **The stream client has no timeout**, unlike every other call. A tail is supposed to stay
  open.
