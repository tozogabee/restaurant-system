# Architecture

Two kinds of document, with different lifetimes.

| Document | Lifetime | Purpose |
| --- | --- | --- |
| [`overview.md`](overview.md) | Kept current | What the system looks like *now*. Edited whenever that changes. |
| [`decisions/`](decisions/) | Append-only | Why it looks that way. One file per decision, never rewritten. |

The split matters. The overview answers "how does this work?" for someone
joining today. The decision record answers "why is it like this, and what did we
rule out?" for someone six months from now who is about to change it. A current
overview alone loses the reasoning; a pile of decisions alone makes a reader
reconstruct the present from history.

## What belongs here

Anything that constrains more than one service: transport choices, data
ownership, auth, deployment topology, cross-cutting conventions.

What does not belong here:

- **What the system must do** — that is
  [`../requirements/`](../requirements/). A requirement names no technology; an
  ADR does.
- **How one service works internally** — that goes in that service's own
  `README.md` under `services/<name>/`.
- **API contracts** — `api/proto` is the source of truth. Do not restate a
  message shape here; it will drift.
- **How to run things** — the root [`README.md`](../../README.md) owns setup and
  the local stack.

## Adding a decision

Copy [`decisions/0000-template.md`](decisions/0000-template.md) to the next
number, write it, and add it to the index in
[`decisions/README.md`](decisions/README.md). Keep it short — a decision that
needs ten pages is usually several decisions.

Record a decision when it is **expensive to reverse** or when someone will
reasonably ask "why on earth is it done this way?". Library picks and naming
conventions do not need one.

## Diagrams

Prefer text that can be diffed. Mermaid renders on GitHub and lives in the
Markdown file that explains it, so it stays next to its context rather than
becoming a stale PNG nobody can edit.
