# nook-cli — Usage

Every command and flag, with the behaviour that is not obvious from the help text.

## Global flags

| Flag | What it does |
|---|---|
| `--url <url>` | Instance URL for this invocation, beating `NOOK_URL` and the stored value |
| `--json` | Print one JSON document to stdout and nothing else. Forces colour off |
| `--no-color` | Disable colour. Colour is already off when stdout is not a TTY or `NO_COLOR` is set |
| `--version` | `nook <semver>`, one line |
| `-h`, `--help` | Help for the command |

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | The work failed |
| `2` | Usage error — a bad flag, a missing argument, an unknown command |
| `130` | Stopped with ctrl-c, which is how `nook tail` normally ends |

Data goes to stdout; steps, warnings and errors go to stderr. A piped command emits only
data, so `nook targets --json | jq` and `nook tail --json | while read -r line` both work.

## `nook login [url]`

Stores the instance URL and the session it returns.

```sh
nook login nook.facile.studio          # prompts, without echoing
nook login --password "$NOOK_PASSWORD" # for a script
nook login                             # re-authenticate against the stored URL
```

The URL takes a bare hostname and assumes `https://`. The password is read from the terminal
without echo, or from stdin when piped.

An instance running with no `NOOK_ADMIN_PASSWORD` serves every caller as the admin. Against
one of those, `login` says so and stores only the URL — there is no session to hold.

## `nook logout`

Revokes the session on the instance and clears it locally, keeping the URL. If the instance
cannot be reached, the local token is cleared anyway and the command warns: the copy on this
machine is the one that matters.

## `nook status`

What the instance is running.

```
✓ https://nook.facile.studio is healthy
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

A delivery target with no provider and no tag attached is called out as a warning. Nook's
routing is opt-in with no fallthrough, so such a target receives nothing, and this is the
only place outside the dashboard that says so.

## `nook providers`

Every configured source, with the address it watches or answers on and the tags a delivery
target can route on. `●` enabled, `○` disabled.

## `nook targets`

Every delivery target and what routes to it.

```
   NAME                    TYPE        ROUTED BY
●  Facile Perception       perception  19 providers
●  Facile Logs Matrix      matrix      5 providers, tags: infra
●  GW Discord              discord     nothing attached
```

`nothing attached` means the routing rules can never select it.

## `nook events`

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

## `nook tail`

Follows the log until interrupted.

```sh
nook tail                    # everything
nook tail --source webhook   # one source
nook tail --json | jq -r '.event'
```

The transport is server-sent events. The instance's feed is unfiltered, so `--source` is
applied client-side. A dropped connection reconnects after two seconds — a deploy restarts
the instance routinely, and a tail should survive it — but a rejected session does not
retry.

With `--json`, each event is one document on its own line rather than a growing array: a
stream has no end at which an array could be closed.

## `nook test [target]`

With a target, sends straight to it, bypassing the routing rules. This is what answers "does
this channel work at all".

```sh
nook test "Ops room"                                    # by name
nook test 6388bd80-680e-4733-a60a-76e682a33592          # by id
```

The name is matched exactly first, then as a substring, both case-insensitively. An
ambiguous name is refused rather than guessed — sending a test to the wrong channel is noise
in somebody's chat room.

With no target, sends one alert through the whole pipeline instead, routing rules included.
That stays silent unless a target has opted into Nook's own events by selecting the system
provider, which is the point: it tests the routing, not the channel.

## `nook replay <event> <target>`

Re-sends a logged event's stored envelope to one target, bypassing the routing rules. Only
events that carry an envelope can be replayed. Event ids come from `nook events --json`.

```sh
nook replay 1786022937782-000003 "Ops room"
```

## `nook pool`

The Nook Pool's boot epoch and every connected app with the channels it subscribed to.

The epoch changes on every restart. An app that reconnects and sees a new one knows to
resync from its own cursor rather than trust its offsets across the gap.
