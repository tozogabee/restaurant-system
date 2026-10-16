# Preconception

Early thinking about what the restaurant system is, before any of it is built.

**This document is a record, not a decision.** Everything here is what the
preconception says; where it conflicts with code already in the repository,
that conflict is called out rather than quietly resolved. See
[Open questions](#open-questions) for what still has to be decided.

## Provenance and completeness

Captured on 2026-10-16 from a Gemini conversation
([share link](https://share.gemini.google/dsexK39FAe0R)). The link is not
machine-readable — the page renders client-side, so the shared text could not be
extracted — and only its **final exchange** was recovered, pasted by hand.

That exchange answers one narrow question: how a manager uses the system while
driving or in a meeting. It is rich about clients, roles and notifications, and
it names technologies in passing that were clearly settled earlier in the
conversation. Everything before it — the domain model, the service breakdown,
the core ordering flow — is **missing**. Sections below marked *(not in the
captured source)* are gaps in this record, not gaps in the thinking.

| Section | Source confidence |
| --- | --- |
| Clients and manager-on-the-move UX | Stated directly, in detail |
| Technology choices | Named in passing; the reasoning is not captured |
| Services, events, roles | Named as examples only; no complete list |
| Domain model | Inferred from passing mentions |
| Non-functional requirements | Not in the captured source |

---

## System shape

Four clients over a Go microservice backend.

### Clients

**React admin panel** — mobile-first Progressive Web App, Tailwind CSS for the
responsive work. One codebase, two experiences:

- *Desktop:* full data tables, multi-column analytics, complex scheduling grids.
- *Mobile:* media queries collapse tables into touch-friendly summary cards and
  hide non-essential columns. Installable to the home screen, so it behaves like
  a native app with no App Store or Play Store deployment.

The stated reasoning: *"You cannot expect someone to parse a 50-row inventory
grid on an iPhone 15 Pro while on the move."* The mobile view is deliberately
high-signal and action-oriented rather than a shrunken dashboard.

**Flutter POS app** — one app, not two, with the view chosen by role at login:

- Waiter on a tablet → straight to the floor plan and order entry.
- Manager on their personal phone → `ManagerDashboardView`, showing live
  revenue, staff currently on the clock, and critical operational alerts.

JWT claims decide the route. The explicit intent is to avoid building a third
application for managers.

**Push notifications** — the driving case. A manager behind the wheel needs
asynchronous alerts, not a dashboard. Notifications are *actionable*: the
manager taps **Approve** or **Deny** on the lock screen and the response goes
back to the API gateway over REST, without the app ever opening.

**Messaging-based AI assistant** — for when opening a console would be too
disruptive, e.g. mid-meeting. The manager discreetly messages a bot over
WhatsApp Business API, Telegram or Slack:

> "What is the live revenue right now?"
>
> "86 the sweet potatoes."

A Python NLP engine parses the intent and makes the corresponding gRPC calls to
the backend.

### Backend

| Element | What the source says |
| --- | --- |
| Services | Go microservices |
| Edge | An API gateway; notification actions come back to it over REST |
| Internal calls | gRPC |
| Events | Kafka topics |
| AI | A separate Python service for NLP / chatbot |
| Notifications | A lightweight Go service subscribing to critical Kafka topics, pushing via Firebase Cloud Messaging (FCM) |
| Auth | JWT, with claims driving role-based access control |

Services named in passing: **Kitchen Service** (flags severe delays), and
whatever owns inventory and security auditing. The full service list is *not in
the captured source*.

Kafka topics named as examples: `sys.security.events`, `sys.inventory.events`.
The `sys.<domain>.events` naming convention is visible; the full topic list is
not.

---

## Roles

Three appear, none with a full permission list:

- **Manager / admin** — the subject of the whole captured exchange. Reads live
  revenue, staffing and alerts; approves or denies escalations; can change menu
  availability.
- **Waiter** — floor plan and order entry on a tablet.
- **Cashier** — mentioned only as the actor who *"attempts to void a bill"*,
  which implies they cannot void above a threshold without approval.

RBAC is central and explicitly described as *strict*, enforced from JWT claims.
The source's closing offer — *"Want to map out the RBAC implementation?"* — was
never taken up, so **the permission matrix does not exist yet**.

---

## Workflows

Two are described concretely.

**Void approval.** A cashier tries to void a bill over a configured threshold →
the backend emits a security event → the notification service picks it up →
the manager gets an actionable push → Approve or Deny → the decision posts back
to the API gateway → the void proceeds or is refused.

This is the clearest requirement in the source and it constrains a lot: voids
need a threshold, an approval state, an audit trail, and a blocking wait on a
decision that may arrive from a lock screen minutes later.

**Kitchen delay escalation.** The Kitchen Service detects a severe delay →
emits an event → the manager is alerted.

**Menu availability over chat.** "86 the sweet potatoes" → NLP resolves the
intent and the item → a gRPC call marks it unavailable. ("86" is kitchen slang
for an item being out of stock; the system needs that as a first-class state,
not a deletion.)

---

## Domain concepts

Surfaced only in passing, listed here because they tell us which services will
be needed:

- Floor plan and tables
- Orders and order entry
- Bills, and voiding them
- Inventory and stock levels; item availability ("86")
- Kitchen tickets and preparation delays
- Staff: who is on the clock, and shift scheduling
- Revenue, live and reported
- Security and audit events

A real data model — entities, fields, relationships — is *not in the captured
source*.

---

## Conflicts with the current repository

The foundations committed to this repository were built before this document
existed, on choices made in a separate conversation. Two differences matter:

**Event broker: Kafka (preconception) vs NATS — unresolved, and now unbuilt.**
A NATS JetStream wrapper (`pkg/broker`) and a NATS container were built
alongside the rest of the foundations, before this document existed. Both were
**removed on 2026-10-16** at the user's instruction, precisely because the
choice is not made: the preconception says Kafka, the scaffolding said NATS, and
leaving one of them in place would have let services start depending on it by
accident.

So there is currently no broker package, no broker container and no broker
configuration anywhere in the repository. Kafka earns its weight with high
throughput, long retention and replay; NATS is far lighter to run and operate.
Neither is obviously right for a restaurant's event volume. The first service
that genuinely needs events has to settle this first.

**Frontend: previously open, now answered.** React for the admin PWA and
Flutter for the POS. `web/` is still an empty placeholder and its README still
says "framework not chosen yet".

**Python service.** The AI assistant is Python, so the monorepo is no longer
Go-only. Nothing in the Makefile or `go.work` accounts for a non-Go service yet.

Consistent with what is built, needing no change: Go services, gRPC internally,
REST at the gateway, and the `x-request-id` propagation that already spans both.

---

## Open questions

Anything a build would otherwise have to guess.

**Scope and priority**

- Which service is built first, and what is the thinnest useful slice?
- Is this one restaurant or several (does multi-tenancy exist)?

**Services**

- What is the full service list, and where are the boundaries?
- Who owns the void-approval state machine — an orders service, or a dedicated
  approvals service?

**Events**

- Kafka or NATS — see above. Nothing is built either way; this blocks any
  event-driven service, including the notification engine the preconception
  describes.
- What is the full topic list, and does the `sys.<domain>.events` convention
  cover domain events too, or only system ones?

**Auth and RBAC**

- The permission matrix per role. The source offered to map this and stopped.
- Where do JWTs come from — an own auth service, or an external identity
  provider?
- What happens when a manager's approval never arrives? Is a pending void
  blocking, auto-denied after a timeout, or left open?

**The AI assistant**

- Which messaging platform first?
- Is it read-only at first, or does it execute write commands like "86" from day
  one? A bot that can change menu state needs the same RBAC checks as the app,
  and a chat transport makes identity harder to establish.

**Notifications**

- Which events are "critical" enough to push?
- FCM covers Android and web; iOS needs APNs behind it. Does the manager's phone
  matter here?

**Not in the captured source at all**

- Payments and fiscal/receipt requirements — a restaurant system usually has
  legal obligations here
- Reservations, despite tables and a floor plan being central
- Offline behaviour: a POS tablet on restaurant Wi-Fi will lose connectivity,
  and order entry probably cannot simply fail
- Deployment target, environments, CI
- Non-functional expectations: concurrent users, data retention, backup

---

## Next step

Recover the rest of the original conversation if it still exists — the missing
part almost certainly covers the service breakdown and domain model, which is
exactly what this record lacks. Failing that, the fastest way forward is to pick
the first vertical slice; the Kafka/NATS question only has to be settled once a
service actually needs to publish an event, and nothing in the repository
pre-empts it.
