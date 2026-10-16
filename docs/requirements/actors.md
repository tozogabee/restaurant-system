# Actors

Who uses the system. Source: [`../preconception.md`](../preconception.md#roles)
unless noted.

Three staff roles come from the preconception, plus one more — the dine-in
customer — added by the project owner on 2026-10-16. None of the staff roles has
a full permission list: RBAC is described as strict and driven by JWT claims, but
the matrix was never written (see [`functional.md`](functional.md) `FR-082`).

The customer is categorically different from the other three, and the difference
drives most of what follows: **every staff actor is authenticated and trusted;
the customer is neither.**

## Waiter

**Device:** tablet, shared, on the restaurant floor.
**Client:** Flutter POS app, routed straight to the floor plan at login.

Needs to see which tables are occupied and take orders against them, fast and
repeatedly, during a service. This is the highest-frequency interaction in the
whole system and the one most exposed to bad network conditions.

*Status: Captured.* The device being shared is **Inferred** — a floor tablet
usually is, which matters because it decides whether sessions are per-shift or
per-person.

## Cashier

**Device:** not stated.
**Client:** presumably the POS app. Not stated.

Appears exactly once in the source, as the actor who *"attempts to void a
bill"* — phrasing that implies they can void up to a threshold and need
approval above it.

*Status: Captured, barely.* Whether cashier is a distinct role or a permission a
waiter can hold is **Open**, and it changes the permission model.

## Manager / admin

**Devices:** desktop, personal phone, and no device at all (chat, notifications).
**Clients:** React admin PWA, the POS app in manager mode, push notifications,
and a chat assistant.

The subject of the entire captured exchange, and the actor that shapes the
architecture. What makes the manager unusual is that **they are frequently not
on site** — driving, or in a meeting — and the system has to stay usable in both
of those states:

| Situation | What works | What does not |
| --- | --- | --- |
| At a desk | Full admin PWA: data tables, analytics, scheduling grids | — |
| On the floor | POS app in manager mode: live revenue, staff on the clock, alerts | — |
| Driving | Actionable push notifications, resolved from the lock screen | Anything requiring reading or typing |
| In a meeting | Chat assistant, discreet text questions and commands | Opening a console |

Reads live revenue, staffing and alerts; approves or denies escalations; changes
menu availability.

*Status: Captured, in detail.* This is the best-evidenced actor in the source.

## Customer (dine-in)

**Device:** a tablet at the table, in the restaurant, handled by the public.
**Client:** a self-ordering interface — see [`../../web/README.md`](../../web/README.md).

A seated guest browses the menu and places their own order from a tablet at the
table, without a waiter taking it.

*Status: Captured.* Stated by the project owner, 2026-10-16.

This is the first actor who is **not an employee**, and that changes the threat
model rather than extending it. Everywhere else, a request arrives with a JWT
issued to a known person with a known role. Here:

- Nobody signs in. The claim "I am at table 12" comes from the device, not from a
  person, and the device is sitting in a dining room all day.
- The guest has unsupervised physical access to the client, for as long as their
  meal lasts.
- Anything the tablet can reach, a determined guest can try to reach. It is an
  untrusted client on the inside of the building.

So the self-ordering surface has to be treated as public, not internal: a
narrow, dedicated API scoped to one table's own order, never the staff API with
a different screen on top. See `FR-100`–`FR-107` in
[`functional.md`](functional.md) and `NFR-016` in
[`non-functional.md`](non-functional.md).

**What this actor does not settle:** whether the guest also pays from the tablet,
whether a waiter confirms the order before it reaches the kitchen, and how the
tablet is bound to its table in the first place. Those are open questions, not
details.

## Not in the source

Roles a restaurant system usually has, which no source mentions. Listed so their
absence is a known gap rather than an assumption.

The dine-in customer was on this list until 2026-10-16 and has moved above.
Note what that did: the first version is **not** staff-only, so the system has a
public surface, which was the reason the row flagged it as worth confirming.

| Possible actor | Why it matters |
| --- | --- |
| **Kitchen staff** | The Kitchen Service flags delays, so *something* tracks ticket progress — but whether a human marks a ticket ready, or it is inferred, is unspecified. This decides whether a kitchen display client exists at all. |
| **Owner / multi-site manager** | Unknown whether this is one restaurant or several. Multi-tenancy changes every schema, so it cannot stay open for long. |
| **System administrator** | Nobody is named as responsible for users, roles, devices or configuration. |
