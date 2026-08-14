---
name: antenne
description: >
  Facile alert bus terminal client. Use when the user asks to read the activity
  or alert log, follow it live, list or test delivery targets, or mentions
  Antenne.
---

# antenne — Facile alert bus

Binary: `antenne`
Config: `<config_dir>/antenne/config.json` (instance URL + session token)

Antenne aggregates alerts from providers and routes them to delivery targets
(Matrix, SMTP, etc.). This CLI reads the resulting activity log, follows it
live, and exercises delivery — without the dashboard.

## When to apply

Use when the user mentions alerts, the activity/event log, delivery targets,
providers, or wants to know what the bus is doing or why a target looks broken.
Triggers: "alert", "alerts", "activity log", "event", "delivery target",
"target", "provider", "antenne", "test alert"

## Commands

```
antenne login [url]               Store the instance URL and its session
antenne status                    What the instance watches and delivers
antenne tail                      Follow the event stream live until ctrl-c
antenne events -n 50 [--source X] Search and filter the log
antenne providers                 List configured sources
antenne targets                   List delivery targets and their routes
antenne test "<target|alert>"     Send straight to one delivery target
antenne replay <id> [--target X]  Re-send a logged event
antenne bus                       Bus bootstrap epoch and connected apps
```

## Rules
- A session is required; run `antenne login <url>` first, or set `ANTENNE_URL`.
- `--json` on every command carrying data (targets, providers, events, status,
  bus) — pipe to `jq` when composing.
- `tail` is a live SSE stream; ctrl-c stops it with exit 130.
- The instance's feed is unfiltered, so filtering (`--source`) is applied
  client-side.
- `test` exercises the whole pipeline and waits on the third party — it is a
  live delivery, not a dry run; use it to verify, not to preview.
- `targets` also names delivery targets nothing routes to — that is usually why
  they look broken.
- Exit: `0` success, `1` failure, `2` usage, `130` SIGINT.
