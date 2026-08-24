# antenne-cli — Architecture

How the client talks to an instance, what it stores, and the decisions that are not obvious
from the code.

## Shape

```
main.go            three lines: hands off to cmd
cmd/               one file per command, plus root.go and format.go
internal/
  client/          the HTTP surface and the SSE reader
  config/          the instance URL and the session token
  ui/              CLI-STANDARD §4 output vocabulary
```

Dependencies are cobra, `fatih/color` and `golang.org/x/term`. Everything else is the
standard library. A client for one API does not need an HTTP framework.

## Authentication

Antenne's API runs on the suite's `porte` kit. `antenne login` posts an address and a password
to `/api/auth/login` and stores the token the response *body* carries; every later call sends
it as `Authorization: Bearer …`.

The instance sets a session cookie on that same response, and the CLI ignores it on purpose.
porte reads the cookie before the Authorization header, and refuses a cookie-authenticated
*mutating* request that carries no `X-Facile-CSRF` header — so a cookie client that forgot the
header sees every read succeed and every write 403, which reads as "the tool works, saving is
broken". Bearer is exempt from that rule by construction, because nothing attaches a bearer
header on the caller's behalf.

The token is a bearer credential in a file, so the file is `0600` and its directory `0700`.
There is no keychain integration: on Go that means either cgo or a dependency, and the file
is what every other Go tool on the machine already uses.

`login` reads `/api/auth/config` before prompting for anything. It reports whether a password
login exists at all, whether single sign-on is mandatory, and which address the bootstrap
account signs in with — which is what the prompt prefills. The address is the instance's to
decide and is never recomputed here; a client that guessed would be wrong the first time that
rule changed. An instance that only federates gets told it cannot be signed into from here
yet, rather than being asked for a password it would refuse.

## Why the response body is read as text

Antenne serves its dashboard from the same origin as its API, and the SPA catch-all answers an
unknown path with `200` and HTML. A streaming JSON decoder reports that as a syntax error at
byte 1, which sends the reader hunting for a bug in the API instead of a typo in the URL.

So the body is read whole and parsed defensively, and a decode failure is reported as
*"answered with something that is not JSON — check the URL points at an Antenne instance"*. It
is the same class of trap as a health check that only asserts `status < 400`.

## The event stream

`antenne tail` reads server-sent events, not a WebSocket: the traffic is one-way, and the
instance already exposes `/api/events/stream` for its own dashboard.

`internal/client/sse.go` implements only the part of the grammar Antenne emits — `data:` lines
terminated by a blank line, comments skipped as keep-alives. A frame that will not decode is
skipped rather than ending the stream: one malformed entry should not stop a tail that has
been running for an hour.

The stream client deliberately has no timeout, unlike every other call. The whole point is
that it stays open.

Reconnection waits two seconds and is unconditional except on a rejected session. An
instance restarts on every deploy, and a tail that dies each time is a tail nobody leaves
running; but retrying a `401` forever would just hammer the login rate limiter.

## Resolving a target by name

`antenne test` and `antenne replay` take a target id or a name. The match is exact id, then exact
name, then substring — all case-insensitive.

An ambiguous name is an error, not a guess. Sending a test alert to the wrong channel is
noise in somebody's chat room, and `antenne targets` is one command away.

## What this client does not do

It reads configuration and never writes it. There is no `antenne providers add`, no
`antenne settings set`.

That is a deliberate boundary, not a missing feature. Antenne's settings are one document
replaced wholesale on save, with secrets redacted on the way out and merged back on the way
in. A CLI that PUT a partial document would have to reproduce that merge exactly, and
getting it wrong overwrites live credentials with the redaction marker — which has happened
once already, from a script, and cost eleven of them. The dashboard owns that write path.
