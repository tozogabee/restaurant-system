# web

The browser-based clients. Two of them, with very different audiences.

| Client | Audience | Status |
| --- | --- | --- |
| **Admin panel** | Manager / admin, authenticated staff | Not started. React, mobile-first PWA, Tailwind CSS — per [`docs/preconception.md`](../docs/preconception.md) |
| **Customer self-ordering** | Dine-in guests, on a tablet at the table | Not started. Framework undecided |

Nothing is built yet. This directory holds the intent and the constraints both
clients have to respect.

## Admin panel

One codebase, two experiences. On desktop it renders full data tables,
multi-column analytics and scheduling grids; on mobile those collapse into
touch-friendly summary cards with non-essential columns hidden. It installs to a
phone home screen, so there is no App Store or Play Store deployment.

The mobile view is deliberately **not** a shrunken desktop. A manager reading it
is usually on the move, so it is high-signal and action-oriented — see
[`NFR-040`](../docs/requirements/non-functional.md).

## Customer self-ordering

A seated guest browses the menu and places their own order from a tablet at the
table ([`FR-100`, `FR-101`](../docs/requirements/functional.md#customer-self-ordering)).

**This client is untrusted, and that is not a detail.** Every other client in
the system carries a JWT identifying a known employee. This one has no person
behind it: it sits in a dining room all day, handled by the public, with
unsupervised physical access for the length of a meal. Treat its surface as
internet-facing even though the device never leaves the building.

What follows from that:

- It gets its **own narrow API**, scoped to one table's own order. It must not
  share the staff API with a different screen on top — the same surface that
  voids bills cannot be reachable from a guest's tablet.
- The claim "I am at table 12" comes from the device, not a person. How a tablet
  is bound to its table, and who may rebind it, is unresolved
  ([`FR-107`](../docs/requirements/functional.md#customer-self-ordering)).
- The device itself needs locking to the application
  ([`NFR-017`](../docs/requirements/non-functional.md)) — a device-management
  job, not a code one, but someone has to own it.

Two open questions change what this client even is:

- **Does a waiter confirm the order before the kitchen sees it?**
  ([`FR-105`](../docs/requirements/functional.md#customer-self-ordering)) With
  confirmation it is a faster notepad. Without it, a member of the public puts
  work into the kitchen directly, which needs rate limiting and a sanity check
  on quantities.
- **Does the guest pay from the tablet?**
  ([`FR-106`](../docs/requirements/functional.md#customer-self-ordering)) If so,
  payment acquires a public surface.

Worth noting: a table tablet could equally run the Flutter codebase in a
customer mode rather than a web app. It is listed here because that is where the
project owner placed it; if it ends up sharing the Flutter build, this section
moves and the constraints above travel with it unchanged.

## Constraints both clients share

Every client talks **only** to the gateway's REST API — never to an individual
service, and never to Postgres, Redis or a message broker directly.

The error shape to handle is `httpx.ErrorBody` in `pkg/httpx/response.go`, which
every service returns:

```json
{
  "error": {
    "kind": "not_found",
    "code": "menu_item_unavailable",
    "message": "that dish is not available right now",
    "fields": { "item_id": "unknown" },
    "request_id": "0b4f…"
  }
}
```

`kind` is the coarse class to branch on; `code` is a stable identifier when the
service sets one; `fields` carries per-field validation detail; `request_id`
correlates the response with the server logs and is worth surfacing in any error
a user can report.

Money is always integer minor units plus an ISO 4217 currency code — never a
float. `1250` with `"EUR"` is €12.50. Formatting is the client's job; arithmetic
on a float price is a bug.
