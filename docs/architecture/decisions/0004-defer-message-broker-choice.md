# 0004. Defer the Kafka/NATS choice and ship no broker

- **Status:** Accepted
- **Date:** 2026-10-16
- **Supersedes:** —

## Context

[ADR 0003](0003-transport-split.md) commits to events as a transport without
saying what carries them, and the two candidate answers came from different
places.

The scaffolding was built with **NATS JetStream**: a `pkg/broker` wrapper with
durable consumers, an event envelope and ack/nak handling, plus a NATS container
in the local stack, all working and covered by an integration test.

The project preconception ([`../../preconception.md`](../../preconception.md))
says **Kafka**, naming topics as `sys.security.events` and `sys.inventory.events`
and describing a notification service that consumes them.

Nothing decides between them, because the facts needed to decide are missing:
there is no service breakdown, no event volume, no retention or replay
requirement, and no idea whether events are the audit trail or merely a
notification path.

Meanwhile, a working broker in the repository is not neutral. The first service
to need an event would have used what was there, and a default would have become
the decision.

## Decision

Remove the broker entirely until the choice is made. On 2026-10-16 `pkg/broker`,
the NATS container, its configuration and its integration test were all deleted.

There is now no broker package, no broker container and no broker configuration
anywhere in the repository. `deploy/local/docker-compose.yml` and `.env.example`
each carry a comment saying the absence is deliberate, so it does not read as an
oversight.

A service that needs to publish an event has to answer this question first.
That friction is the mechanism, not a side effect.

## Alternatives considered

**Keep NATS, revisit later.** It was built, tested and working. Rejected because
"revisit later" is how a default becomes permanent: by the time a second service
depends on it, the choice has been made by inertia rather than on merit.

**Switch to Kafka now, as the preconception says.** Rejected because the
preconception is a partial record — only its last exchange survives — and the
reasoning behind the Kafka mention was never captured. Adopting it would be
following a conclusion whose argument is lost, and Kafka is the more expensive
of the two to run and operate.

**Keep a transport-agnostic abstraction** — the `Event` envelope and `Handler`
signature, with no implementation. Tempting, and it would have preserved the
event vocabulary. Rejected because an interface designed before its one
implementation tends to encode assumptions from the broker its author had in
mind; Kafka's partitioned, offset-based consumption and NATS JetStream's
subject-filtered consumers do not abstract cleanly behind a shared surface.
Better to write it once, against a known broker.

## Consequences

Nothing in the repository assumes a broker, so either choice stays fully open at
no migration cost. `pkg/` has no transport coupling at all.

What this costs:

- Nothing event-driven can be built until this is settled, including the
  notification engine the preconception describes — which is the manager's
  entire off-site experience, so this may block sooner than it looks.
- Roughly 250 lines of working, tested NATS integration were thrown away. It was
  never committed, so it is not recoverable from history.
- Anyone reading `pkg/` sees an architecture with a missing third of
  [ADR 0003](0003-transport-split.md), which is why this record exists.

## What would settle it

Kafka earns its weight with sustained throughput, long retention and replay from
an offset — which matter if events are the audit trail, or if a new service must
rebuild state from history. NATS is dramatically lighter to run and operate, and
JetStream covers durability and redelivery for a single restaurant's volume
without a cluster to look after.

The deciding questions: are events the audit trail of record, does anything need
to replay history, and is there an operator for a Kafka cluster? Two yeses point
to Kafka; otherwise NATS.
