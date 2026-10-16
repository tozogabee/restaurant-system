# 0001. Monorepo with one Go module per deployable

- **Status:** Accepted
- **Date:** 2026-10-16
- **Supersedes:** —

## Context

The system is a set of microservices plus several clients, built by a very small
team. Two things had to be settled before any code: whether the services share a
repository, and whether they share a Go module.

Separate repositories per service mean a contract change touches several pull
requests across several repositories, and keeping them in step is manual work
that scales with the number of services.

A single Go module across a monorepo means every service's binary links every
other service's dependencies. Adding a heavy client library to one service grows
all of them, and `go.mod` becomes a shared contention point.

## Decision

One repository. Multiple Go modules, tied together by a committed `go.work`:

- `pkg/` — libraries shared by every service
- `api/` — protobuf contracts and their generated Go code
- `services/<name>/` — one module per service, added as it is written

`go.work` is committed so every clone resolves the same module set with no
setup. `go.work.sum` is not, being a local artifact.

The Makefile discovers modules by finding `go.mod` files rather than listing
them, so adding a service needs no build-system change.

## Alternatives considered

**One module for the whole repository.** Much simpler — no workspace, no
per-service `go.mod`. Rejected because it couples every service's dependency
graph: one service needing a large SDK imposes it on all of them, and a single
`go.mod` is where every change collides.

**One repository per service.** Normal for larger organisations. Rejected
because at this team size the coordination cost dominates: a protobuf change
would become a multi-repository release dance, and the contracts would drift.

**Go submodules without `go.work`.** Workable, but every developer then needs
`replace` directives or published versions of `pkg/` to build locally. The
workspace exists precisely to avoid that.

## Consequences

A contract change and every consumer of it land in one commit, reviewable
together. Services stay independently buildable and deployable.

The costs are real:

- Multi-module tooling is less well-trodden. Commands that work on one module
  need a loop, which is why everything goes through the Makefile.
- A module with no Go files yet breaks `go build ./...`, so the Makefile skips
  modules with no buildable packages.
- `pkg/` is consumed via the workspace, not a version. A breaking change to it
  breaks every service at once. That is the intended trade — it keeps `pkg/`
  honest — but it means `pkg/` has to stay small and stable.
