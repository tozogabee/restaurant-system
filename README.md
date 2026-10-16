# restaurant-system

A restaurant management system: a Go microservice backend in a monorepo with the
clients that talk to it.

## Aim

Run a restaurant's day-to-day operations end to end — take orders on the floor,
route them to the kitchen, track stock and staff, report revenue — and let a
manager stay in control of all of it without being on site.

That second half is the part that shapes the architecture. A manager who is
driving or in a meeting cannot read a dashboard, so oversight is delivered as
actionable push notifications and a chat assistant rather than as another admin
screen. See [`preconception.md`](preconception.md) for where this comes from.

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
| AI assistant | Python NLP over WhatsApp / Telegram / Slack | ❌ not started |
| Push | Firebase Cloud Messaging, actionable notifications | ❌ not started |
| Auth | JWT, claims driving RBAC | ❌ not started |
| Local stack | Docker Compose | ✅ `deploy/local/` |
| Logging | stdlib `log/slog`, structured, context-carried | ✅ `pkg/logger` |

Only the Go foundation and the local stack are real today. Everything marked ❌
is intent recorded in `preconception.md`, not code.

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
web/                  Frontend (empty — React PWA per the preconception)
deploy/local/         Docker Compose stack: Postgres, Redis
preconception.md      Early project thinking, and what it leaves undecided
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

## Architecture decisions

**Transport.** Three paths, each for a different job:

- **REST** at the gateway, the only thing the clients talk to.
- **gRPC** between services, for anything a caller needs an answer to.
  Contracts live in `api/proto`, generated with `make proto`.
- **Events** for anything that must not be lost and must not block the caller —
  an order paid, a bill voided, a kitchen ticket running late. **Not built
  yet:** the broker is undecided, so nothing here commits to one.

Errors keep their meaning across the two that exist: a service returns an
`errs.Error`, and `httpx.Fail`, `grpcx.Status` and `grpcx.FromStatus` translate
it in both directions. Nothing below the transport layer imports `net/http` or
`grpc`.

**Data.** One Postgres instance with a schema per service, plus one shared
Redis. A service reads and writes only its own schema; if it needs another
service's data, it asks that service. The DSN pins `search_path` to the
service's schema so a stray unqualified query fails rather than reaching across
the boundary.

**Migrations.** Each service owns `services/<name>/migrations`, applied with
`make migrate-up SERVICE=<name>`. Every service's first migration creates its
own schema, so no shared bootstrap file has to be edited to add one.

**Configuration.** Environment only, no config files: a service embeds
`config.Base` and adds the dependencies it uses, and `config.Load` fails at
startup when something required is missing. `.env.example` lists every variable
with its local value.

**Observability.** Every request carries an `x-request-id`, generated at the
edge, propagated through gRPC metadata and onto published events, and attached
to every log line. One client call is traceable through every service that
touched it.

## Questionable points

Things that are genuinely undecided, not just unbuilt. The full list, with
context, is in [`preconception.md`](preconception.md#open-questions).

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
