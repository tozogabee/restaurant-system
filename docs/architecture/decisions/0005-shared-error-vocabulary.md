# 0005. One error vocabulary, translated at each boundary

- **Status:** Accepted
- **Date:** 2026-10-16
- **Supersedes:** —

## Context

[ADR 0003](0003-transport-split.md) uses three transports, each with its own way
of saying something went wrong: HTTP status codes, gRPC status codes, and a dead
letter on a broker. Without a shared vocabulary each service invents its own
mapping, and the same failure arrives as 404 from one service and 500 from
another.

Two further problems follow from a system of services calling each other:

A failure's meaning has to survive a hop. If service A asks B for a menu item
that does not exist and B's `NotFound` reaches A as an opaque internal error,
then A reports a 500 to the client for what is really a 404.

Internal detail must not leak. A `pq: relation "orders" does not exist` reaching
a client discloses schema and is useless to them, but the same string in the
service log is exactly what the on-call engineer needs.

## Decision

Define the vocabulary once, in `pkg/errs`, as an `Error` carrying a `Kind` —
`invalid_input`, `unauthorized`, `forbidden`, `not_found`, `conflict`,
`unavailable`, `internal` — plus an optional stable `Code` for clients to branch
on, a client-safe `Message`, per-field validation `Fields`, and a wrapped cause.

Domain, service and repository code returns that. Only the transport layer
translates:

- `httpx.Fail` → HTTP status and a single JSON error body
- `grpcx.Status` → gRPC status code
- `grpcx.FromStatus` → back into an `errs.Error` when a client call fails

The `Message` is for the client; the wrapped cause is for the log. For
`internal`, the transport substitutes a generic message and drops `Fields`
entirely, so a cause cannot cross a boundary. Everything else passes its message
through.

An untyped error is treated as `internal`, so forgetting to classify something
fails safe rather than disclosing it.

Nothing below the transport layer imports `net/http` or `grpc`.

## Alternatives considered

**Return HTTP status codes from the service layer.** Common and direct.
Rejected because it makes business logic unusable over gRPC: the layer that
knows a reservation is invalid would have to know about HTTP to say so.

**Sentinel errors with `errors.Is`** — `ErrNotFound`, `ErrConflict` and so on.
Idiomatic Go and lighter. Rejected because the extras are needed in practice:
per-field validation detail, a stable code for clients, and a message distinct
from the cause. A sentinel carries none of those without a wrapper type anyway.

**gRPC status codes as the single vocabulary**, mapped to HTTP at the edge.
Genuinely tempting, since it removes one translation. Rejected because it leaks
the internal transport into the domain — `codes.FailedPrecondition` is not a
concept the business logic has — and gRPC's codes do not map cleanly onto HTTP
in both directions.

## Consequences

A failure keeps its meaning end to end: a `NotFound` in a repository is a 404 to
an HTTP client, `codes.NotFound` to a gRPC caller, and `KindNotFound` again if
that caller is another service. The round trip is covered by tests, as is the
guarantee that an internal cause never serialises.

Every service returns the same JSON error shape, so one client-side error
handler covers all of them.

What this costs:

- Two mappings to maintain, and they must stay inverse. Both are small table
  functions in `pkg/errs` and `pkg/grpcx`, tested as a round trip for exactly
  this reason.
- `errs.Kind` is deliberately coarse. A failure needing finer distinction uses
  `Code`, and the temptation will be to add kinds instead — the set should stay
  small enough to map onto any transport.
- Every service depends on `pkg/errs`, making it the least changeable package in
  the repository.
