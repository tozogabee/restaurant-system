# 0003. REST at the edge, gRPC between services, events for the rest

- **Status:** Accepted
- **Date:** 2026-10-16
- **Supersedes:** —

## Context

Three kinds of traffic exist in this system, and they have different
requirements.

Clients are a React PWA, a Flutter app and a chat assistant. They are written in
three languages by whoever picks them up, they are debugged in a browser's
network tab, and they change shape more often than the backend does.

Service-to-service calls are internal, high-frequency, and benefit from a typed
contract that breaks a build rather than a production request.

Some things must happen reliably but must not block the caller: a bill is voided
and a manager has to be notified, a kitchen ticket runs late and someone has to
hear about it. Making the caller wait for all of that couples unrelated work to
its latency.

## Decision

Use all three, each for what it is good at.

| Transport | Between | For |
| --- | --- | --- |
| REST / JSON | client → gateway | the only public surface |
| gRPC | service ↔ service | anything the caller needs an answer to |
| Events | service → service | anything that must not be lost and must not block |

Protobuf definitions in `api/proto` are the source of truth for the internal
API; generated code is not committed. The gateway is the only service exposed
publicly, and clients never call a service directly.

The event broker is deliberately not chosen — see
[ADR 0004](0004-defer-message-broker-choice.md).

## Alternatives considered

**REST everywhere, including between services.** Simplest: no code generation,
debuggable with `curl`, one mental model. Rejected because the internal contract
is where typing pays most — a field rename should fail at compile time in every
caller, not at runtime in one. Hand-written JSON structs on both sides of a
boundary drift.

**gRPC everywhere, including to clients.** Consistent, and grpc-web exists.
Rejected because it taxes exactly the layer that should be cheapest to work on:
a browser client would need a proxy, and the people building the React and
Flutter apps would need the protobuf toolchain to make a single call.

**Events for everything, event sourcing.** Strong audit trail, which a system
handling money and voids genuinely wants. Rejected as the primary transport
because most of what this system does is synchronous question-and-answer —
"what is on the menu", "is this table free" — and forcing those through a log
makes simple reads hard. Revisit if auditability turns out to be a legal
requirement rather than a nicety.

## Consequences

Each transport is used where it is strongest, and the public surface stays a
plain REST API that any client can consume.

What this costs:

- Three transports to understand, operate and keep consistent. The mitigation is
  [ADR 0005](0005-shared-error-vocabulary.md): one error vocabulary translated
  at each boundary, so meaning survives a hop.
- A code generation step (`make proto`) between editing a contract and using it,
  which has to be in the onboarding path or a fresh clone does not build.
- The gateway is a single point of failure and a place where logic accretes. It
  should stay a router with auth, not become a service.
- Request correlation has to cross all three. Hence `x-request-id`, propagated
  through gRPC metadata and onto published events.
