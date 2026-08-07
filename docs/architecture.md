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

Antenne's API authenticates by cookie and reads nothing else — no bearer header, no API key. So
`antenne login` posts the password, reads `Set-Cookie: antenne_session=…` off the response, and
stores the value; every later call sends it back as a `Cookie` header.

The token is a bearer credential in a file, so the file is `0600` and its directory `0700`.
There is no keychain integration: on Go that means either cgo or a dependency, and the file
is what every other Go tool on the machine already uses.

An instance running with no `ANTENNE_ADMIN_PASSWORD` serves every caller as the admin and sets
no cookie. That is a success with an empty token, not a failure — `login` checks
`/api/session/public` first so it never prompts for a password that does not exist.

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
