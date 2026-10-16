# Decision record

One file per architectural decision, in the order they were made. Append-only:
a decision that no longer holds is **superseded** by a new one, not edited or
deleted. The history is the point — it shows what was considered and rejected,
which is exactly what a future reader needs before changing it.

| # | Decision | Status |
| --- | --- | --- |
| [0001](0001-monorepo-multi-module.md) | Monorepo with one Go module per deployable | Accepted |
| [0002](0002-postgres-schema-per-service.md) | One Postgres instance, one schema per service | Accepted |
| [0003](0003-transport-split.md) | REST at the edge, gRPC between services, events for the rest | Accepted |
| [0004](0004-defer-message-broker-choice.md) | Defer the Kafka/NATS choice and ship no broker | Accepted |
| [0005](0005-shared-error-vocabulary.md) | One error vocabulary, translated at each boundary | Accepted |

**Status** is `Proposed`, `Accepted`, `Superseded by NNNN` or `Rejected`. A
rejected decision is still worth keeping: it stops the same idea being
relitigated.

New decision: copy [`0000-template.md`](0000-template.md), take the next number,
add a row above.
