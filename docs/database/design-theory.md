# Database design theory

The reasoning behind the choices in
[`schema-design-template.md`](schema-design-template.md). General principles
first, Postgres specifics where they matter, and this project's own constraints
marked as **[project]** so the theory stays reusable.

Target engine throughout: **PostgreSQL 18**.

---

## 1. The order of work

Schema design goes wrong most often by starting in the wrong place. Entities
first feels natural and produces tables nobody can query efficiently.

1. **Boundaries.** What does this service own? Everything else is someone
   else's data.
2. **Aggregates.** What must change together atomically?
3. **Access patterns.** Who reads and writes what, how often, filtered by what?
4. **Entities and relationships.** Only now draw tables.
5. **Types and constraints.** Make illegal states unrepresentable.
6. **Indexes.** One per access pattern, not one per column.
7. **Migrations.** How this gets from nothing to here, and from here to next.

Steps 1–3 are requirements work. Step 4 onwards is design. Reversing them
produces a schema that models the world accurately and serves no query well.

---

## 2. Schema-per-service changes the rules

**[project]** [ADR 0002](../architecture/decisions/0002-postgres-schema-per-service.md)
gives each service its own schema in one shared database.

Three familiar tools stop being available across a boundary:

| Normally | Across a service boundary |
| --- | --- |
| Foreign key enforces the reference | Bare `UUID`, nothing enforces it |
| `JOIN` assembles the answer | Two calls, joined in application code |
| One transaction keeps both sides consistent | Two transactions; consistency has to be designed |

This is the cost of service independence, and it is why **where you draw the
boundary matters more than how you model either side**. A boundary through the
middle of an invariant means that invariant can no longer be enforced by the
database at all.

The corollary: if two tables must *always* be consistent with each other, they
belong to the same service. If "usually, within a second" is acceptable, they
can be separated.

---

## 3. Aggregates and transaction boundaries

An **aggregate** is a cluster of data with a single entry point (the root) and
an invariant spanning its members. The classic example: an order and its lines.
The invariant — *the order total equals the sum of its lines* — cannot be
maintained if the two are written independently.

The practical test is a question, not a diagram:

> **If these two rows disagreed for 200 milliseconds, would anything be wrong?**

- *No* → they can be separate aggregates, and separate services.
- *Yes, money or safety is involved* → same aggregate, same transaction, same
  schema.

Aggregates should be as small as the invariants allow. A large aggregate means a
large transaction, which means lock contention on the hot path.

**[project] Worked example.** An order line references a menu item. Must they be
consistent? No — once an order is placed, later menu changes must *not* affect
it. That tells you two things at once: menu and orders can be separate services,
and the order line must copy what it needed rather than point at it (§8).

---

## 4. Normalisation, and when to stop

Normalisation removes redundancy so that one fact lives in one place. Each form
eliminates a specific class of anomaly.

| Form | Rule | Anomaly it removes |
| --- | --- | --- |
| **1NF** | Column values are atomic; no repeating groups or arrays standing in for rows | Cannot query or constrain what is buried inside a column |
| **2NF** | 1NF, and every non-key column depends on the *whole* primary key | Partial dependency — only possible with a composite key |
| **3NF** | 2NF, and no non-key column depends on another non-key column | Transitive dependency: updating one fact requires updating many rows |
| **BCNF** | Every determinant is a candidate key | The residual edge cases 3NF allows |

**3NF is the default. Aim there, then deviate deliberately.**

Three legitimate reasons to deviate:

1. **Temporal correctness** — copying a value so history cannot be rewritten.
   This looks like redundancy and is not: the snapshot and the current value are
   *different facts* (§8). This is the most common and most important case.
2. **A read pattern you have measured** — a counter or rollup that would
   otherwise need an expensive aggregate on the hot path. Measure first; the
   planner is better than intuition.
3. **A cross-service copy** — data owned elsewhere, cached locally to avoid a
   call. Needs a named staleness tolerance and a refresh story.

"It felt faster" is not on the list. Denormalisation buys read speed with write
complexity and a new way for data to be wrong.

---

## 5. Keys

### Natural versus surrogate

A **natural key** is business data (an email, a table number, a SKU). A
**surrogate key** is meaningless and system-generated.

Prefer surrogate primary keys. Business data changes — a table gets renumbered,
an email gets corrected — and a changing primary key cascades everywhere.

Natural keys still earn their place as `UNIQUE` constraints. That is where they
belong: enforcing a business rule, not identifying a row.

### Choosing the type

| Type | For | Against |
| --- | --- | --- |
| `BIGINT` identity | Compact (8 bytes), fast, index-friendly, readable in logs | Needs a round trip to learn the id; enumerable from outside; awkward to merge datasets |
| `UUID` v4 | Generated anywhere, no coordination, not guessable | 16 bytes; **random, so inserts scatter across the B-tree** — page splits, cache churn, index bloat |
| `UUID` v7 | Time-ordered, so inserts append; keeps v4's independence | 16 bytes; leaks creation time |

**Postgres 18 has native `uuidv7()`.** It is the sensible default for a
distributed system: ids can be generated by any service or client without
coordination, and inserts stay sequential.

```sql
id UUID PRIMARY KEY DEFAULT uuidv7()
```

Note that `gen_random_uuid()` is v4 — random. It has been in core since
Postgres 13, so `pgcrypto` is not needed for UUIDs alone.

If an id is exposed to an untrusted client, sequential integers let that client
enumerate your data. **[project]** The customer tablet (`NFR-016`) makes this
concrete.

---

## 6. Types

Choosing a type is choosing which illegal values the database will refuse.

### Money

**Never floating point.** `0.1 + 0.2 != 0.3` in binary floating point, and a
restaurant adds prices thousands of times a day.

Two correct options:

| Approach | Example | Trade-off |
| --- | --- | --- |
| Integer minor units | `BIGINT` cents + `CHAR(3)` currency | Exact, fast, trivial to sum; the application formats it |
| `NUMERIC(p,s)` | `NUMERIC(12,2)` | Exact, readable in SQL, arbitrary precision; slower arithmetic |

**[project]** Integer minor units, because `api/proto` already defines `Money`
that way. The database should match the wire contract — a conversion at the
boundary is a place for a rounding bug to hide.

Store the currency with every amount, even in a single-currency system. An
amount without a currency is not a quantity of money.

### Time

| Type | Stores | Use for |
| --- | --- | --- |
| `TIMESTAMPTZ` | An instant (UTC internally, converted on output) | **Everything.** Any moment something happened |
| `TIMESTAMP` | A wall-clock reading with no zone | Essentially nothing |
| `DATE` | A calendar day | Business dates, deliberately zoneless |
| `INTERVAL` | A duration | Durations — not two timestamps subtracted in application code |

`TIMESTAMP` without a zone is the classic silent corruption: it survives
development, then breaks at a daylight-saving transition or the first server in
another region.

Distinguish two different clocks, and store both when they differ:

- **Valid time** — when the thing happened in the world (`occurred_at`)
- **Transaction time** — when the database learned about it (`recorded_at`)

An offline client makes these diverge by minutes. A report on the wrong one is
wrong.

**[project] The business day.** A restaurant open until 02:00 has one service
spanning two calendar dates. `GROUP BY date(created_at)` splits it, and every
daily figure is wrong. Store an explicit `business_date DATE`, set when the
order opens, and never derive it from a timestamp.

### Text

`TEXT` and `VARCHAR(n)` are identically fast in Postgres; `n` is a constraint,
not an optimisation. Use `TEXT` with a `CHECK` where you actually have a limit.

`CITEXT` (installed here) is case-insensitive — right for emails and
identifiers, where `Bob@x.com` and `bob@x.com` must not be two rows.

### Enumerations

Three ways to constrain a column to a set of values:

| Approach | Adding a value | Removing one | Join needed |
| --- | --- | --- | --- |
| `TEXT` + `CHECK` | Alter the constraint | Alter the constraint | No |
| Postgres `ENUM` | `ALTER TYPE ... ADD VALUE` | **Impossible** | No |
| Lookup table + FK | Insert a row | Delete a row | Yes |

**Prefer `TEXT` + `CHECK`.** Postgres enums cannot have a value removed, and a
value added inside a transaction cannot be used until that transaction commits —
which makes some migrations awkward in ways you discover at the worst moment.

Use a lookup table when the set is genuinely data: editable by users, or
carrying attributes of its own.

### JSONB

`JSONB` is right for data that is genuinely schemaless, write-once, or not
yours: an event payload, a third-party API response, a user-defined attribute
bag.

It is wrong for your core domain. Inside `JSONB` you lose column constraints,
foreign keys, cheap statistics for the planner, and a schema anyone can read.
A `JSONB` column holding an order's lines is a table that gave up its
advantages.

---

## 7. Constraints: database or application?

Application code can be bypassed — by a second service, a migration, a
maintenance script, a future bug. The database cannot.

Put it in the database when a violation would be **data corruption** rather
than a rejected request:

| Constraint | Belongs in the database because |
| --- | --- |
| `NOT NULL` | A missing required value is corruption, not a validation failure |
| `UNIQUE` | Only the database can check this without a race |
| `CHECK` | Cheap, declarative, and documents the rule where it applies |
| `FOREIGN KEY` | Only within a schema — see §2 |

Keep in the application: anything needing other services, external calls, or
rules that change often. Complex multi-table invariants in triggers become
invisible behaviour that nobody expects.

A `UNIQUE` constraint maintained only in application code is a race waiting for
concurrency. Two requests both check, both find nothing, both insert.

---

## 8. History: the part most schemas get wrong

Transactional records must not change meaning when master data changes.

```sql
-- WRONG. Reprice the dish and last quarter's revenue silently changes.
CREATE TABLE order_lines (
    order_id     UUID NOT NULL,
    menu_item_id UUID NOT NULL REFERENCES menu_items(id),
    quantity     INT  NOT NULL
);

-- RIGHT. The line records what was actually charged.
CREATE TABLE order_lines (
    order_id          UUID    NOT NULL,
    menu_item_id      UUID    NOT NULL,   -- reference for reporting only
    name_at_order     TEXT    NOT NULL,   -- what the customer saw
    unit_amount_minor BIGINT  NOT NULL,   -- what they were charged
    currency          CHAR(3) NOT NULL,
    quantity          INT     NOT NULL CHECK (quantity > 0)
);
```

The rule generalises: **anything a customer saw or agreed to gets copied onto
the transaction.** Price, name, tax rate, discount terms. The foreign key, if
the tables are even in the same schema, points at *which* item — never at *what
it costs*.

Two consequences follow:

**Referenced master data can never be hard-deleted.** Orders point at menu
items; deleting an item breaks history or cascades disastrously. Deletion
becomes a state. Which means:

**Lifecycle flags and domain states are different columns.** "Archived" (an
operator retired it) and "unavailable" (the kitchen ran out) are not the same
fact and do not belong in one `status` column. Conflating them is how an
archived dish reappears when stock is restocked.

**Financial records are append-only.** A correction is a new row referencing the
original, never an `UPDATE` or `DELETE`. This is what makes an audit trail
trustworthy — and whatever a jurisdiction requires fiscally will assume it.

---

## 9. Concurrency

### Postgres isolation levels

MVCC means readers never block writers and dirty reads never happen at any
level.

| Level | Prevents | Still allows |
| --- | --- | --- |
| **Read Committed** (default) | Dirty reads | Non-repeatable reads, phantoms, lost updates in read-modify-write code |
| **Repeatable Read** | The above, plus phantoms (snapshot isolation) | Write skew |
| **Serializable** | Everything; behaves as if transactions ran one at a time | Nothing — but raises serialization failures the application must retry |

Repeatable Read and Serializable both fail transactions rather than blocking, so
**any code using them needs a retry loop.** Without one, raising the isolation
level trades a rare data bug for a frequent user-visible error.

### The lost update

Two clients read a row, both modify it, the second write erases the first.
Read Committed does not prevent it.

```sql
-- Optimistic: cheap, and correct when conflicts are rare.
UPDATE orders SET total_minor = $1, version = version + 1
WHERE id = $2 AND version = $3;
-- Zero rows affected means someone else got there first. Re-read and retry.
```

```sql
-- Pessimistic: correct when conflicts are common, at the cost of holding a lock.
SELECT * FROM orders WHERE id = $1 FOR UPDATE;
```

Optimistic locking is the right default: no lock held across application
round-trips, and the failure is explicit.

Note that `UPDATE t SET n = n + 1` is *not* a lost update — Postgres re-reads
the row under lock. The danger is specifically read-then-write in application
code.

**[project]** `FR-101` lets a guest add to their table's order while a waiter
edits the same order. That is a textbook lost update with a real customer at the
end of it, so orders need a `version` column from the first migration.

### Idempotency

Any client that may retry needs its writes to be idempotent. The cheapest
mechanism is a client-generated key with a unique constraint:

```sql
idempotency_key TEXT NOT NULL,
CONSTRAINT orders_idempotency_key_unique UNIQUE (idempotency_key)
```

The second attempt violates the constraint; the handler treats that as success
and returns the existing row. The database does the hard part.

**[project]** `NFR-001` guarantees this is needed: a POS tablet that loses
connectivity mid-request will resend.

---

## 10. Integrity without foreign keys

Across a service boundary there is no foreign key. Pick a strategy per
reference and write it down:

| Strategy | How | When |
| --- | --- | --- |
| **Validate on write** | RPC to the owning service before inserting | The reference must be real at write time |
| **Snapshot** | Copy what you need, keep the id for reference only | History must not change — usually the right answer (§8) |
| **Accept eventual** | Insert; reconcile later | A dangling reference is tolerable |
| **Subscribe** | Keep a local projection updated by events | Frequent reads, staleness tolerable |

The failure mode nobody plans for: the owning service is down. "Validate on
write" means your service is down too. Decide deliberately whether that
coupling is acceptable.

---

## 11. Events and the outbox

Writing to the database and publishing an event are two systems. Do them
separately and either can succeed alone — a state change nobody hears about, or
an event for a change that rolled back.

The **outbox pattern** removes the dual write:

1. In one transaction: change the state *and* insert a row into an `outbox`
   table.
2. A separate publisher reads unsent outbox rows, publishes them, marks them
   sent.

The event is now as durable as the state change, because it *is* a state change.

```sql
CREATE TABLE outbox (
    id            UUID        PRIMARY KEY DEFAULT uuidv7(),
    event_type    TEXT        NOT NULL,
    payload       JSONB       NOT NULL,   -- genuinely schemaless: correct use of JSONB
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at  TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished ON outbox (id) WHERE published_at IS NULL;
```

That partial index is the pattern's point: the publisher's query touches only
unsent rows, so the index stays small however large the table grows.

Delivery is **at-least-once** — a crash between publishing and marking sent
replays the event. Consumers must be idempotent; that is why the envelope needs
a stable event id.

**[project]** The broker is undecided
([ADR 0004](../architecture/decisions/0004-defer-message-broker-choice.md)), but
the outbox table is broker-agnostic. Only the publisher that drains it is not,
so this can be designed now.

---

## 12. Indexes

An index is a write-time cost paid for a read-time benefit. Every one makes
`INSERT`, `UPDATE` and `DELETE` slower and consumes cache.

**Every index should trace to a known query.** Indexing "just in case" is pure
cost.

### What to know

**The leading-column rule.** An index on `(a, b, c)` serves filters on `a`, on
`a, b`, and on `a, b, c`. It does *not* usefully serve a filter on `b` alone.
Order the columns by selectivity and by what your queries filter on first.

**Equality before range.** For `WHERE status = $1 AND created_at > $2`, index
`(status, created_at)`. The reverse forces a scan of everything after the date.

**Partial indexes** cover a subset, staying small:

```sql
CREATE INDEX orders_open ON orders (table_id) WHERE closed_at IS NULL;
```

Open orders are a handful at any moment, out of years of history. The index
stays tiny and the hot-path query stays fast.

**Covering indexes** let a query be answered from the index alone:

```sql
CREATE INDEX ON order_lines (order_id) INCLUDE (quantity, unit_amount_minor);
```

### Types beyond B-tree

| Type | For |
| --- | --- |
| B-tree | Default: equality, ranges, ordering |
| GIN | `JSONB` containment, arrays, full-text search |
| BRIN | Very large, naturally ordered tables — tiny index, cheap range scans |
| Hash | Equality only; rarely worth choosing over B-tree |

Verify with `EXPLAIN (ANALYZE, BUFFERS)` against realistic data volumes. On a
hundred rows the planner picks a sequential scan whatever you build.

---

## 13. Migrations

### Forward-only in practice

A `down` migration is a development convenience. In production you roll
*forward* — a down migration that drops a column destroys data the rollback was
supposed to protect. Write them for local iteration; never plan to use one
against live data.

### Expand and contract

Any non-additive change to a deployed service needs the old and new shapes to
coexist, because code and schema never deploy at the same instant:

| Phase | Schema | Code |
| --- | --- | --- |
| **Expand** | Add the new column, nullable | Still writes the old one |
| **Backfill** | Populate it, in batches | Writes both |
| **Migrate** | — | Reads new, writes both |
| **Contract** | Drop the old column, add `NOT NULL` | Uses only the new |

Four deploys to rename a column. The alternative is downtime, or a window where
one version of the code cannot read the database.

**Adding a `NOT NULL` column with no default to a populated table fails.** Add
it nullable, backfill, then add the constraint.

### Discipline

- A migration that has been applied anywhere shared is **immutable**. Fix it
  with a new one.
- Keep DDL and data migrations in separate files: different failure modes,
  different runtimes.
- Test on a populated copy. An empty-database migration proves nothing about one
  holding a million rows.
- Long-held locks are an outage. `ALTER TABLE` often needs `ACCESS EXCLUSIVE`;
  know which variants do not.

---

## 14. Anti-patterns

| Anti-pattern | Why it hurts |
| --- | --- |
| Float for money | Rounding errors that compound and cannot be reconciled |
| `TIMESTAMP` instead of `TIMESTAMPTZ` | Silent corruption at a DST boundary or a region change |
| Referencing a live price from a transaction | Repricing rewrites history |
| Hard-deleting referenced master data | Broken history or a cascade that destroys more than intended |
| One `status` column for several unrelated concerns | Impossible to reason about; invalid combinations become representable |
| **EAV** (entity-attribute-value) | A database inside a database: no types, no constraints, unqueryable |
| Polymorphic foreign keys (`owner_type` + `owner_id`) | No referential integrity possible |
| Indexing every column | Write amplification with no reader |
| `UNIQUE` enforced only in application code | A race condition, not a constraint |
| Denormalising before measuring | Complexity and a new inconsistency, for an unproven gain |
| Nullable columns meaning two different things | `NULL` as "unknown" and "not applicable" at once |
| Foreign keys across service boundaries | Couples deployments; forbidden here by ADR 0002 |
| `SELECT *` in a migration or view | Breaks when a column is added |

---

## 15. Postgres 18 notes

Verified against the running container:

| Feature | Note |
| --- | --- |
| `uuidv7()` | Native and time-ordered. The default key choice (§5) |
| `gen_random_uuid()` | v4, random. In core since 13 — no `pgcrypto` needed for this alone |
| `CITEXT` | Installed. Case-insensitive text for emails and identifiers |
| `pgcrypto` | Installed. `crypt()`, `digest()`, HMACs |
| `GENERATED ... STORED` | Computed columns, maintained by the database |
| `INCLUDE` on indexes | Covering indexes (§12) |
| Partial indexes | The outbox and open-order patterns above |
| `ON CONFLICT` | Upserts, and the idempotency pattern (§9) |
| `REVOKE CREATE ON SCHEMA public` | Already done in the local bootstrap, so a service cannot create tables outside its schema |

**[project]** Data directory note: the Postgres 18 image expects the whole
`/var/lib/postgresql` mounted, not `.../data` — see
`deploy/local/docker-compose.yml`.

---

## Further reading

- *Designing Data-Intensive Applications*, Kleppmann — chapters 2, 5, 7, 11
- *SQL Antipatterns*, Karwin — EAV, polymorphic associations, and the rest
- *Implementing Domain-Driven Design*, Vernon — aggregates and boundaries
- [PostgreSQL documentation: data types](https://www.postgresql.org/docs/18/datatype.html),
  [indexes](https://www.postgresql.org/docs/18/indexes.html),
  [transaction isolation](https://www.postgresql.org/docs/18/transaction-iso.html)
