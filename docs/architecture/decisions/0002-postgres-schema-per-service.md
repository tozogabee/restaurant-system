# 0002. One Postgres instance, one schema per service

- **Status:** Accepted
- **Date:** 2026-10-16
- **Supersedes:** —

## Context

Microservices owning their own data is the point of the pattern: if two services
read the same table, a schema change to it breaks both, and they are one service
wearing two hats.

Enforcing that strictly — a database per service, or an instance per service —
costs real operational effort, which at this team size comes out of the budget
for building the actual restaurant system.

## Decision

One Postgres instance holding one database, with **one schema per service**. A
service reads and writes only its own schema; if it needs another service's
data, it asks that service over gRPC.

The boundary is enforced mechanically rather than by discipline: the connection
string pins `search_path` to the owning service's schema, so an unqualified
query that reaches for another service's table fails instead of quietly
succeeding. The local bootstrap also revokes `CREATE` on `public`, so a service
cannot accidentally create tables outside its schema.

Redis is likewise one shared instance, with keys namespaced per service by
`redisx.NewNamespace`.

Migrations live with the service that owns them, in `services/<name>/migrations`,
each tracked in its own `schema_migrations_<name>` table. A service's first
migration creates its own schema.

## Alternatives considered

**A database per service, same instance.** Stricter isolation, and closer to
textbook practice. Rejected for now because it rules out ever reading across
boundaries even in an emergency, while still sharing the instance's failure
domain and resources — so it buys isolation on paper without buying independence
in operations. The upgrade path from schemas is straightforward if it is wanted
later.

**An instance per service.** Genuine isolation, including failure domains.
Rejected as disproportionate: a heavier local stack and real per-instance
operational cost, for a system that has no load figures yet.

**A shared schema with table prefixes.** Rejected outright. It relies entirely
on convention, gives no mechanism to enforce ownership, and is where "one
service wearing two hats" comes from.

## Consequences

Each service owns its tables and can migrate them without coordinating. The
local stack stays one Postgres container, so `make up` is fast.

What this costs:

- No foreign keys across service boundaries, and no cross-service joins. A
  query that wants both services' data becomes two calls and a join in code.
- No distributed transaction. A write that must span services needs a saga or
  an outbox, and the consistency has to be designed per case.
- One shared failure domain: the instance going down takes every service with
  it. Acceptable for a single restaurant, and the thing to revisit first if that
  assumption changes.
- Per-service connection pools against one instance share a connection limit,
  which needs watching as services multiply.
