# restaurant-system

A restaurant management system: a Go microservice backend in a monorepo with the
clients that talk to it.

## Aim

Run a restaurant's day-to-day operations end to end — take orders, whether from
a waiter on the floor or a guest at their own table, route them to the kitchen,
track stock and staff, report revenue — and let a manager stay in control of all
of it without being on site.

That second half is the part that shapes the architecture. A manager who is
driving or in a meeting cannot read a dashboard, so oversight is delivered as
actionable push notifications and a chat assistant rather than as another admin
screen. See [`docs/preconception.md`](docs/preconception.md) for where this comes from.

## Status

**Foundations only.** No business services exist yet. What is here is the shared
infrastructure every service will be built on, plus a local stack to run
against. The domain model and service boundaries are still open — see
[Questionable points](#questionable-points).

## Technology stack

| Layer | Choice | In the repo? |
| --- | --- | --- |
| Services | Go 1.26, Gin for HTTP | ✅ `pkg/httpx` |
| Service-to-service | gRPC, protobuf contracts via buf | ✅ `pkg/grpcx`, `api/` |
| Events | **Undecided — Kafka or NATS** | ❌ removed until decided |
| Database | Postgres 18, pgx, schema per service | ✅ `pkg/postgres` |
| Cache / sessions | Redis 8 | ✅ `pkg/redisx` |
| Migrations | golang-migrate, per service | ✅ `make migrate-*` |
| Admin panel | React PWA, mobile-first, Tailwind CSS | ❌ `web/` is empty |
| POS app | Flutter, one app, role-switched views | ❌ not started |
| Customer ordering | Tablet at the table, framework undecided | ❌ not started |
| AI assistant | Python NLP over WhatsApp / Telegram / Slack | ❌ not started |
| Push | Firebase Cloud Messaging, actionable notifications | ❌ not started |
| Auth | JWT, claims driving RBAC | ❌ not started |
| Local stack | Docker Compose | ✅ `deploy/local/` |
| Logging | stdlib `log/slog`, structured, context-carried | ✅ `pkg/logger` |

Only the Go foundation and the local stack are real today. Everything marked ❌
is intent recorded in [`docs/preconception.md`](docs/preconception.md), not code.

## Layout

```
api/                  Service contracts (protobuf), the source of truth for RPCs
  proto/              Hand-written .proto files
  gen/                Generated Go code (not committed; `make proto`)
pkg/                  Shared Go libraries, one module
  app/                Component supervisor: run and shut down cleanly
  config/             Environment-driven configuration
  errs/               The error vocabulary every layer speaks
  grpcx/              gRPC server, client, interceptors, status mapping
  health/             /healthz and /readyz
  httpx/              Gin engine, middleware, error responses
  logger/             slog setup and context propagation
  postgres/           pgx pool, transactions, driver-error mapping
  redisx/             Redis client and per-service key namespaces
  reqid/              Request-ID propagation across HTTP and gRPC
services/             One directory per microservice (empty — next step)
web/                  Browser clients: admin panel, customer ordering (empty)
deploy/local/         Docker Compose stack: Postgres, Redis
docs/
  requirements/       What the system must do, for whom, and how well
  architecture/       Design overview and the decision record (ADRs)
  preconception.md    Early project thinking, and what it leaves undecided
```

Go modules are tied together by `go.work`, which is committed. `pkg/` and
`api/` are separate modules, and each service will be its own module too, so a
service can be built and deployed without pulling in its siblings.

## Getting started

### Prerequisites

| Tool | Version | Why |
| --- | --- | --- |
| Go | 1.26.4 or newer | `go.work` and both modules pin 1.26.4 |
| Docker | with Compose v2 | the local stack, and `buf` / `golang-migrate` run as containers |
| make | any | every entry point goes through the Makefile |

Nothing else has to be installed. `buf` and `golang-migrate` are invoked from
containers, so there is no protoc or migrate binary to manage; if you prefer
local binaries, set `BUF=buf` or `MIGRATE=migrate`.

```bash
go version      # expect go1.26.4+
docker info     # expect a running daemon, not a connection error
```

### First-time setup

```bash
git clone git@github.com:tozogabee/restaurant-system.git
cd restaurant-system

make proto               # generate api/gen from api/proto — not committed,
                         # so a fresh clone has to do this once
make up                  # start Postgres and Redis, wait until healthy
make check               # fmt, vet and unit tests across every module
make test-integration    # prove the stack is actually reachable
```

If all four succeed, the toolchain is sound. `make test-integration` is the one
that matters: it opens real connections to Postgres and Redis rather than
trusting that the containers are up.

### Starting a work session

```bash
make up      # idempotent; safe to run when the stack is already up
make ps      # confirm both services report (healthy)
```

Then work as normal. Before pushing:

```bash
make check   # what CI would run
```

`make up` only starts what is in `docker-compose.yml` today. If a service was
removed from the compose file while its container was still running, that
container is an orphan and `make down` will not touch it — use
`docker compose -f deploy/local/docker-compose.yml down --remove-orphans`.

### Ending a work session

```bash
make down    # stop the containers, keep the data
make reset   # stop them and delete the volumes — next `make up` starts empty
```

`make down` is the normal one. Reach for `make reset` when migrations have left
the database in a state you no longer want, since every service's migrations
are replayable from scratch.

### Everyday commands

| Command | What it does |
| --- | --- |
| `make help` | list every target with its description |
| `make logs` | tail the local stack's logs |
| `make test` | unit tests only, no stack needed |
| `make cover` | tests with a coverage summary per module |
| `make tidy` | `go mod tidy` in every module, then `go work sync` |
| `make proto` | regenerate `api/gen` after editing a `.proto` |
| `make proto-lint` | lint the proto definitions |
| `make migrate-up SERVICE=<name>` | apply one service's migrations |
| `make modules` | list the Go modules the Makefile discovered |

Targets that loop over modules skip any module with no buildable packages, so a
freshly created service directory will not break the build before it has code.

### Local service credentials

| Service  | Port | Notes                             |
| -------- | ---- | --------------------------------- |
| Postgres | 5432 | user/password/db all `restaurant` |
| Redis    | 6379 | append-only, survives a restart   |

No message broker runs locally — see [Questionable points](#questionable-points).
`.env.example` lists every variable a service will read, with its local value.

### When something is wrong

| Symptom | Cause |
| --- | --- |
| `make proto` fails to start a container | Docker daemon is not running; `buf` runs as a container |
| Postgres restarts in a loop after an image bump | Postgres 18+ wants the whole `/var/lib/postgresql` mounted, not `.../data`; `make reset` after fixing the mount |
| `make up` reports a port already allocated | another Postgres or Redis is bound to 5432/6379 — stop it, or change the published port |
| `no required module provides package .../services/<name>` | the new service module is not in the workspace: `go work use ./services/<name>` |
| A container you removed from compose is still running | it is an orphan — `docker compose ... down --remove-orphans` |

## Frontend

Four clients, none built yet. They are listed together because the differences
between them are the interesting part.

| Client | Who uses it | Where | Tech |
| --- | --- | --- | --- |
| **Admin panel** | Manager / admin | Desktop and phone browser | React PWA, mobile-first, Tailwind |
| **POS app** | Waiter, manager | Floor tablet, manager's phone | Flutter, one app, view chosen by JWT claims |
| **Customer self-ordering** | Dine-in guests | Tablet at the table | Undecided |
| **Chat assistant** | Manager | WhatsApp / Telegram / Slack | Python NLP service, no UI of its own |

Details and constraints: [`web/README.md`](web/README.md) for the browser
clients, [`docs/requirements/actors.md`](docs/requirements/actors.md) for who
each one serves.

**One split matters more than the rest.** Three of these are used by
authenticated employees. The customer tablet is not: it sits in the dining room
all day, handled by the public, with no person behind the request. So it does
not get the staff API with a guest role bolted on — it needs its own narrow API
scoped to a single table's order, and its surface should be treated as
internet-facing even though the device never leaves the building. See
[`NFR-016`](docs/requirements/non-functional.md).

Everything client-side goes through the gateway's REST API, returns the single
`httpx.ErrorBody` error shape, and handles money as integer minor units with an
ISO 4217 code — never a float.

## Requirements

What the system has to do lives in
[`docs/requirements/`](docs/requirements/) — [actors](docs/requirements/actors.md),
[functional](docs/requirements/functional.md) (`FR-NNN`) and
[non-functional](docs/requirements/non-functional.md) (`NFR-NNN`).

Every requirement carries a status, because most of them are not settled:
**Captured** is evidenced in a source, **Inferred** follows from something
captured but nobody said it, **Open** is known to be needed and unspecified.
Treat `Inferred` as a question, not a spec.

## Architecture

The design and the reasoning behind it live in
[`docs/architecture/`](docs/architecture/):

- [**Overview**](docs/architecture/overview.md) — how the system fits together
  now: transports, data ownership, errors, cross-cutting concerns. Kept current.
- [**Decision record**](docs/architecture/decisions/) — why it is that way, one
  append-only file per decision, including what was rejected.

| # | Decision |
| --- | --- |
| [0001](docs/architecture/decisions/0001-monorepo-multi-module.md) | Monorepo with one Go module per deployable |
| [0002](docs/architecture/decisions/0002-postgres-schema-per-service.md) | One Postgres instance, one schema per service |
| [0003](docs/architecture/decisions/0003-transport-split.md) | REST at the edge, gRPC between services, events for the rest |
| [0004](docs/architecture/decisions/0004-defer-message-broker-choice.md) | Defer the Kafka/NATS choice and ship no broker |
| [0005](docs/architecture/decisions/0005-shared-error-vocabulary.md) | One error vocabulary, translated at each boundary |

The short version: clients speak REST to a gateway and nothing else; services
speak gRPC to each other with protobuf contracts in `api/proto`; each service
owns one Postgres schema and asks its neighbours rather than reading their
tables; one error vocabulary in `pkg/errs` is translated at each transport
boundary so a failure keeps its meaning across a hop; and every request carries
an `x-request-id` through all of it.

## Questionable points

Things that are genuinely undecided, not just unbuilt. The full list, with
context, is in [`docs/preconception.md`](docs/preconception.md#open-questions).

**Blocking the first service**

- **Kafka or NATS? Deliberately unanswered.** The preconception says Kafka;
  a NATS JetStream wrapper was built and then **removed**, so that nothing in
  the repository quietly assumes either one. Kafka earns its weight with
  throughput, long retention and replay; NATS is far lighter to run and
  operate. Restaurant event volume makes neither obviously right. Until this is
  settled there is no broker package, no broker container, and no broker
  configuration — any service needing events will have to raise the question
  first, which is the point.
- **No service breakdown exists.** Floor plan, orders, kitchen, inventory,
  staff, billing and audit all appear in the preconception, but the boundaries
  between them were never drawn. Nothing can be built first without this.
- **No permission matrix.** RBAC from JWT claims is described as central, and
  three roles appear (manager, waiter, cashier), but no role's permissions are
  written down.

**Design questions with real consequences**

- **Does a customer's order reach the kitchen unreviewed?** Guests order from a
  tablet at the table, and whether a waiter confirms first
  ([`FR-105`](docs/requirements/functional.md#customer-self-ordering)) decides
  what is being built: with confirmation the tablet is a faster notepad; without
  it, a member of the public puts work into the kitchen directly, which needs
  rate limiting and a sanity check on quantities. The same client also makes the
  system's first public surface, so the API it uses has to be scoped to one
  table rather than being the staff API with a guest role.
- **Void approval semantics.** A cashier's over-threshold void waits on a
  manager tapping Approve on a lock screen. Is the void blocking meanwhile?
  Auto-denied after a timeout? Left pending indefinitely? Each answer implies a
  different state machine.
- **Can the chat assistant write?** "86 the sweet potatoes" changes menu state
  from a WhatsApp message. Establishing *who is asking* over a chat transport is
  much weaker than a JWT, so write commands need either a hard identity binding
  or a read-only first version.
- **Offline POS.** A tablet on restaurant Wi-Fi will lose connectivity. Order
  entry probably cannot simply fail, which pushes toward local queueing and
  conflict resolution — a large design decision, currently unaddressed.

**Absent from the preconception entirely**

- **Payments and fiscal receipts.** A restaurant system usually has legal
  obligations here. Nothing is specified.
- **Reservations**, despite tables and a floor plan being central.
- **Multi-tenancy** — one restaurant or several? This changes every schema.
- **iOS push.** FCM covers Android and web; iOS needs APNs behind it.
- **A non-Go service.** The Python assistant means `go.work` and the Makefile no
  longer describe the whole repo.
- **Deployment, environments, CI.** No target, no pipeline.

## Adding a service

1. `mkdir -p services/<name>/{cmd/<name>,internal,migrations}`
2. `cd services/<name> && go mod init github.com/tozogabee/restaurant-system/services/<name>`
3. `go work use ./services/<name>` from the repository root
4. Define its RPCs in `api/proto/restaurant/<name>/v1/` and run `make proto`
5. Wire `cmd/<name>/main.go` with `config.Load`, `logger.New`, the clients it
   needs, and an `app.Runner` holding its HTTP and gRPC servers

Inside a service, keep the layers one-way:
`internal/handler` → `internal/service` → `internal/repository` → `internal/domain`.
Handlers speak HTTP or gRPC, services hold the business rules, repositories
speak SQL, and domain types know about none of it.
