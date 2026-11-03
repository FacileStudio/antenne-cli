# antenne-cli — Usage

Every command and flag, with the behaviour that is not obvious from the help text.

## Global flags

| Flag | What it does |
|---|---|
| `--url <url>` | Instance URL for this invocation, beating `ANTENNE_URL` and the stored value |
| `--json` | Print one JSON document to stdout and nothing else. Forces colour off |
| `--no-color` | Disable colour. Colour is already off when stdout is not a TTY or `NO_COLOR` is set |
| `--version` | `antenne <semver>`, one line |
| `-h`, `--help` | Help for the command |

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | The work failed |
| `2` | Usage error — a bad flag, a missing argument, an unknown command |
| `130` | Stopped with ctrl-c, which is how `antenne tail` normally ends |

Data goes to stdout; steps, warnings and errors go to stderr. A piped command emits only
data, so `antenne targets --json | jq` and `antenne tail --json | while read -r line` both work.

## `antenne login [url]`

Prints an approval link, opens it when this machine has a browser, and waits until you approve
the sign-in in any browser. Nothing listens on a port and nothing is typed or pasted, so it works
over SSH. It stores the instance URL and the session it returns.

```sh
antenne login                                # https://antenne.facile.studio, approval link
antenne login antenne.example.org            # another instance
antenne login --listener                     # the loopback flow: a local browser redirects here
antenne login --no-listener                  # no browser here: paste the code from another machine
antenne login --password "$ANTENNE_PASSWORD" # the instance password, for a script
```

The URL is the argument, then `ANTENNE_URL`, then the stored one, then
`https://antenne.facile.studio`. A bare hostname assumes `https://`. An instance that has not
shipped device login falls back to the loopback flow with a notice, or to the code paste on a
machine with no browser. An account that never signed in to the instance in a browser is refused:
sign in there once, then run `login` again. `--email` and `--password` sign in with the instance
password on an instance that has one: the prompt prefills the address the instance reports.

A config file written by an older release holds a cookie-era token. It simply stops being
accepted: any command answers *"not authenticated — run `antenne login`"*, and one login
replaces it.

## `antenne logout`

Revokes the session on the instance and clears it locally, keeping the URL. If the instance
cannot be reached, the local token is cleared anyway and the command warns: the copy on this
machine is the one that matters.

## `antenne status`

What the instance is running.

```
✓ https://antenne.facile.studio is healthy
  providers     16 (16 enabled)
  targets       4 (4 enabled)
  events        184
  delivered     160
  failed        30
  pool          3 connected
  listening on  9090
  settings      saved
```

Health is probed before anything else, because it needs no session — which is what tells
"unreachable" apart from "not logged in".

A delivery target with no provider and no tag attached is called out as a warning. Antenne's
routing is opt-in with no fallthrough, so such a target receives nothing, and this is the
only place outside the dashboard that says so.

## `antenne providers`

Every configured source, with the address it watches or answers on and the tags a delivery
target can route on. `●` enabled, `○` disabled.

## `antenne targets`

Every delivery target and what routes to it.

```
   NAME                    TYPE        ROUTED BY
●  Facile Perception       perception  19 providers
●  Facile Logs Matrix      matrix      5 providers, tags: infra
●  GW Discord              discord     nothing attached
```

`nothing attached` means the routing rules can never select it.

## `antenne events`

The activity log, newest first.

| Flag | Default | What it does |
|---|---|---|
| `-n`, `--limit` | `20` | How many to show |
| `-s`, `--search` | — | Full-text across the event, provider, message and delivery errors |
| `--source` | — | `webhook`, `website`, `imap`, `rss`, `system`, `test` |
| `--provider` | — | Scope to one provider id |
| `--target` | — | Scope to one delivery target id |

`--provider` and `--target` each scope the whole query, so they cannot be combined.

The counters in `--json` describe the whole filtered set rather than the page, which is what
keeps them stable while paging.

## `antenne tail`

Follows the log until interrupted.

```sh
antenne tail                    # everything
antenne tail --source webhook   # one source
antenne tail --json | jq -r '.event'
```

The transport is server-sent events. The instance's feed is unfiltered, so `--source` is
applied client-side. A dropped connection reconnects after two seconds — a deploy restarts
the instance routinely, and a tail should survive it — but a rejected session does not
retry.

With `--json`, each event is one document on its own line rather than a growing array: a
stream has no end at which an array could be closed.

## `antenne test [target]`

With a target, sends straight to it, bypassing the routing rules. This is what answers "does
this channel work at all".

```sh
antenne test "Ops room"                                    # by name
antenne test 6388bd80-680e-4733-a60a-76e682a33592          # by id
```

The name is matched exactly first, then as a substring, both case-insensitively. An
ambiguous name is refused rather than guessed — sending a test to the wrong channel is noise
in somebody's chat room.

With no target, sends one alert through the whole pipeline instead, routing rules included.
That stays silent unless a target has opted into Antenne's own events by selecting the system
provider, which is the point: it tests the routing, not the channel.

## `antenne replay <event> <target>`

Re-sends a logged event's stored envelope to one target, bypassing the routing rules. Only
events that carry an envelope can be replayed. Event ids come from `antenne events --json`.

```sh
antenne replay 1786022937782-000003 "Ops room"
```

## `antenne pool`

The Antenne bus boot epoch and every connected app with the channels it subscribed to.

The epoch changes on every restart. An app that reconnects and sees a new one knows to
resync from its own cursor rather than trust its offsets across the gap.

## `antenne keys`

Manage API keys for connected applications.

### `antenne keys list`

List registered API keys.

```sh
antenne keys list
antenne keys list --app myapp
antenne keys list --json
```

| Flag | Default | What it does |
|---|---|---|
| `--app <name>` | — | Filter keys by application name |

### `antenne keys create`

Create a secret backend key or a public browser key.

```sh
antenne keys create --app myapp
antenne keys create --app myapp --public --origins "https://myapp.local" --quota 5000
```

| Flag | Default | What it does |
|---|---|---|
| `--app <name>` | *(required)* | Application name |
| `--public` | `false` | Create a public browser key instead of a secret backend key |
| `--origins <urls>` | — | Comma-separated allowed origins for public keys |
| `--quota <N>` | `0` | Daily event quota limit for public keys (0 for unlimited) |

The created token is printed on stdout and is never displayed again.

### `antenne keys revoke <id>`

Revoke an API key by its numeric ID.

```sh
antenne keys revoke 42
antenne keys revoke 42 --yes
```

| Flag | Default | What it does |
|---|---|---|
| `-y`, `--yes` | `false` | Confirm revocation without prompting |

