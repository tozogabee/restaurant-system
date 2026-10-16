# services

One directory per microservice, each its own Go module:

```
services/<name>/
  cmd/<name>/main.go     wiring: config, logger, clients, app.Runner
  internal/handler/      HTTP (Gin) and gRPC entry points
  internal/service/      business rules
  internal/repository/   SQL against this service's schema only
  internal/domain/       types and invariants, no transport or SQL imports
  migrations/            golang-migrate files, first one creates the schema
```

Nothing here yet. See "Adding a service" in the root README.
