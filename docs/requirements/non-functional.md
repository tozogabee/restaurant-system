# Non-functional requirements

How well the system has to behave. Status vocabulary is in
[`README.md`](README.md#where-these-come-from).

**Nearly everything here is `Open`.** The captured source discusses what the
system does, almost never how well — no load figures, no latency targets, no
retention rules. That is normal for a preconception and fine for now, but these
are the requirements that are expensive to retrofit, so the list exists as a
checklist rather than a specification.

## Availability and resilience

| ID | Requirement | Status | Notes |
| --- | --- | --- | --- |
| NFR-001 | The POS must keep taking orders when the network drops | Open | **The big one.** A tablet on restaurant Wi-Fi *will* lose connectivity mid-service. If order entry simply fails, the system is unusable exactly when it is busiest. Local queueing and conflict resolution is a large design decision; see `FR-002`. |
| NFR-002 | Operating hours, and whether maintenance windows exist | Open | A restaurant has a service period; a deploy during it is different from a deploy at 04:00. |
| NFR-003 | Behaviour when a dependency is down | Captured, partly | `/readyz` already removes a service from the load balancer rather than letting it be restarted, and Redis is treated as a soft dependency. Postgres is a single shared failure domain by choice — see [ADR 0002](../architecture/decisions/0002-postgres-schema-per-service.md). |

## Performance

| ID | Requirement | Status | Notes |
| --- | --- | --- | --- |
| NFR-004 | How current "live revenue" must be | Open | Decides whether `FR-050` is a query, a cached aggregate or a running total fed by events. Seconds and minutes are very different builds. |
| NFR-005 | Order entry responsiveness | Open | The highest-frequency interaction; a waiter repeats it all service. |
| NFR-006 | Push notification delivery latency | Open | `FR-036` is waiting on it: a cashier and customer stand at the till while this travels. |
| NFR-007 | Concurrent users, covers per service, peak order rate | Open | No volume figures exist anywhere. This is what decides Kafka vs NATS — see [ADR 0004](../architecture/decisions/0004-defer-message-broker-choice.md). |

## Security

| ID | Requirement | Status | Notes |
| --- | --- | --- | --- |
| NFR-010 | Internal traffic is currently unencrypted | Captured | `grpcx.Dial` uses insecure credentials, assuming a trusted network or a mesh terminating TLS. Fine locally; a deployment decision before production. |
| NFR-011 | Internal errors never leak their cause to a client | Captured | Enforced in code and tested — see [ADR 0005](../architecture/decisions/0005-shared-error-vocabulary.md). |
| NFR-012 | Card data handling and PCI scope | Open | Follows `FR-090`. The cheapest answer is usually to never touch card data at all. |
| NFR-013 | Audit retention: what is kept, for how long, and who may read it | Open | `FR-032` and `FR-035` produce security events; nothing says how long they live. |
| NFR-014 | Identity assurance over chat | Open | See `FR-073`. A platform account id is a much weaker claim than a JWT. |
| NFR-015 | Device trust for a shared floor tablet | Open | Relates to `FR-084`. |
| NFR-016 | The customer self-ordering client is untrusted: its API is scoped to one table's own order and exposes no staff operation | Inferred | The capability is `FR-101` (Captured); this security posture follows from it but nobody stated it. A tablet in a dining room is a public client inside the building — treat its surface as internet-facing. |
| NFR-017 | A table tablet is locked to its ordering application | Inferred | Follows from `FR-100`: unsupervised public access for the length of a meal. Kiosk mode is a device-management requirement, not a code one, but it has to be someone's job. |

## Data

| ID | Requirement | Status | Notes |
| --- | --- | --- | --- |
| NFR-020 | Backup and recovery, and the acceptable loss window | Open | A day's takings is not reconstructible from anything else. |
| NFR-021 | Retention for orders, bills and audit events | Open | Interacts with `NFR-013` and with whatever `FR-091` turns out to require legally. |
| NFR-022 | Money is exact, never floating point | Captured | `Money` in `api/proto` is integer minor units plus an ISO 4217 currency code. |
| NFR-023 | Time zone and currency handling | Open | `Money` carries a currency; nothing says whether more than one is in scope. |

## Operability

| ID | Requirement | Status | Notes |
| --- | --- | --- | --- |
| NFR-030 | Every request is traceable across services | Captured | `x-request-id` is adopted or generated at the edge, propagated through gRPC metadata, and attached to every log line and error response. |
| NFR-031 | Structured logging | Captured | `log/slog`, JSON outside local development, logger carried on the context. |
| NFR-032 | Metrics and distributed tracing | Open | Nothing exists. Request correlation is in place, which is the groundwork, but there is no exporter and no backend. |
| NFR-033 | Deployment target and environments | Open | No target, no pipeline, no environment definitions. |
| NFR-034 | Graceful shutdown | Captured | `app.Runner` stops components in reverse registration order under one deadline. |

## Usability

| ID | Requirement | Status | Notes |
| --- | --- | --- | --- |
| NFR-040 | The manager's mobile view is high-signal and action-oriented, not a shrunken desktop | Captured | The source is explicit: *"You cannot expect someone to parse a 50-row inventory grid on an iPhone 15 Pro while on the move."* |
| NFR-041 | An alert must be resolvable without reading a screen at length | Captured | The driving case; this is why `FR-065` exists. |
| NFR-042 | The POS is usable at speed, one-handed, by someone carrying plates | Inferred | Follows from the waiter's actual working conditions; nobody stated it. |
| NFR-043 | Accessibility targets | Open | — |
| NFR-044 | Languages | Open | — |
