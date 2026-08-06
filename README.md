# nook-cli

Terminal client for a [Nook](https://github.com/FacileStudio/Nook) instance — read the alert
log, follow it live, and exercise delivery targets without opening the dashboard.

Configuration stays in the instance. This client never writes it.

## What it does

- Follows the activity log live, one line per event, filtered by source
- Lists providers and delivery targets with what routes to what
- Names the delivery targets nothing routes to, which is why they look broken
- Sends a test to one delivery target, or one alert through the whole pipeline
- Replays a logged event to any target
- Reports the Nook Pool's boot epoch and connected apps
- Emits `--json` on every command that carries data

## Stack

| Layer | Tech |
|---|---|
| CLI | Go 1.25, cobra, session token in a 0600 file |
| Releases | GoReleaser, Homebrew tap |

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/FacileStudio/nook-cli/main/install.sh | bash
```

Installs to `~/.local/bin`. Pass `--bin-dir <dir>` to change that, `--source` to build from
source, `--no-skill` to skip AI agent skill registration.

```sh
brew install FacileStudio/tap/nook
```

## Usage

```sh
nook login nook.facile.studio    # stores the URL and the session it returns
nook status                      # what the instance is watching and delivering
nook tail                        # follow the event stream until ctrl-c
nook events -n 50 --source imap  # search and filter the log
nook test "Ops room"             # send straight to one target
nook targets --json | jq         # every command that carries data speaks JSON
```

Full command reference: [docs/usage.md](docs/usage.md).

## Configuration

There is one file, `~/.config/nook/config.json`, holding the instance URL and the session
token. `nook login` writes it; nothing else does.

| Variable | What it does |
|---|---|
| `NOOK_URL` | Instance URL, overriding the stored one. `--url` beats both |
| `XDG_CONFIG_HOME` | Moves the config directory |
| `NO_COLOR` | Disables colour, as does `--no-color` and any non-TTY stdout |

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | How the client talks to an instance, and why it stores what it does |
| [Configuration](docs/configuration.md) | The config file, the environment, and precedence |
| [Development](docs/development.md) | Local setup, the quality gate, releasing |
| [Usage](docs/usage.md) | Every command and flag |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
