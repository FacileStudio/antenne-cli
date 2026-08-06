# nook-cli — Configuration

One file and three environment variables. There is nothing else to set.

## The file

`~/.config/nook/config.json`, or `$XDG_CONFIG_HOME/nook/config.json` when that is set.

```json
{
  "url": "https://nook.facile.studio",
  "token": "…"
}
```

`nook login` writes it; no other command does. The file is `0600` and its directory `0700`,
because the token is a bearer credential — anything that can read it can act as the admin
of that instance until the session expires, thirty days after it was issued.

`nook logout` clears the token and keeps the URL, so logging back in does not mean typing
the host again.

## Environment

| Variable | What it does |
|---|---|
| `NOOK_URL` | Instance URL, overriding the stored one |
| `XDG_CONFIG_HOME` | Moves the config directory |
| `NO_COLOR` | Disables colour, whatever the terminal reports |
| `FACILE_BIN_DIR` | Where `install.sh` puts the binary. Default `~/.local/bin` |

## Precedence

Most specific wins:

```
--url  >  NOOK_URL  >  the stored url  >  http://localhost:9090
```

The fallback is localhost on purpose. This is a self-hosted tool, and a client that silently
pointed at somebody else's instance would be a surprise nobody wants.

## Several instances

There is one stored instance. For a second, either pass `--url` per call or point
`XDG_CONFIG_HOME` at another directory:

```sh
nook status --url https://nook.internal.example

XDG_CONFIG_HOME=~/.config/nook-staging nook login nook-staging.example
XDG_CONFIG_HOME=~/.config/nook-staging nook tail
```

`--url` alone reuses the stored session, which is only useful against an instance that
requires no password. For a second authenticated instance, use the second config directory.

## Colour

Colour is emitted only when stdout is a TTY, `TERM` is not `dumb`, and `NO_COLOR` is unset —
handled by `fatih/color`, so it behaves the same as every other Facile CLI. `--json` forces
it off as well: a caller piping into `jq` must not receive escape codes.
