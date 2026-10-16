# Functional requirements

What the system must do. Status vocabulary is defined in
[`README.md`](README.md#where-these-come-from) — **Captured** is evidenced,
**Inferred** needs confirming, **Open** is unspecified.

Source `P` means [`../preconception.md`](../preconception.md).

## Floor and orders

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-001 | A waiter authenticating on a POS tablet is taken directly to the floor plan | Captured | P |
| FR-002 | A waiter can enter an order against a table | Captured | P |
| FR-003 | The floor plan shows which tables are occupied | Inferred | implied by FR-001 |
| FR-004 | An order reaches the kitchen | Inferred | the Kitchen Service tracks tickets, so orders must arrive there |
| FR-005 | Order lifecycle states, and who may move an order between them | Open | — |

## Kitchen

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-010 | The system tracks preparation progress of kitchen tickets | Captured | P |
| FR-011 | A severe preparation delay raises an alert to the manager | Captured | P |
| FR-012 | What counts as a "severe" delay, and whether it is configurable | Open | — |
| FR-013 | Whether a human marks a ticket ready, or readiness is inferred | Open | decides whether a kitchen display client exists |

## Menu and inventory

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-020 | A menu item has an availability state separate from existence — "86" marks it unavailable without deleting it | Captured | P |
| FR-021 | An authorised user can mark an item unavailable, including from the chat assistant | Captured | P |
| FR-022 | The system tracks inventory stock levels | Captured | P |
| FR-023 | Inventory changes emit events | Captured | P (`sys.inventory.events`) |
| FR-024 | Whether availability is derived from stock or set manually, and what happens when they disagree | Open | — |
| FR-025 | Menu structure: categories, modifiers, variants, pricing rules | Open | only `Money` (minor units + ISO 4217) exists, in `api/proto` |

## Billing and voids

The best-evidenced workflow in the source, and the one with the most
consequences.

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-030 | A cashier can void a bill | Captured | P |
| FR-031 | A void above a configured amount requires manager approval | Captured | P |
| FR-032 | A void request over the threshold emits a security event | Captured | P (`sys.security.events`) |
| FR-033 | The manager receives an actionable notification carrying Approve and Deny | Captured | P |
| FR-034 | The manager's decision returns to the API gateway and the void either proceeds or is refused | Captured | P |
| FR-035 | Every void, approved or denied, is auditable afterwards | Inferred | a security event with an approval decision is an audit trail or it is pointless |
| FR-036 | What the pending void does while it waits — blocks the till, releases, or holds | Open | **blocks implementation**; see note |
| FR-037 | What happens if no decision ever arrives: timeout, auto-deny, or indefinite | Open | **blocks implementation** |
| FR-038 | Whether any other manager may approve, or only a specific one | Open | — |
| FR-039 | Who sets the threshold, and at what scope | Open | — |

> **FR-036 / FR-037 are the sharp edge.** A cashier is standing at a till with a
> customer waiting while a manager, possibly driving, has not yet looked at their
> phone. Each answer implies a different state machine, and the UX of the POS
> changes completely depending on it. Nothing downstream of billing can be built
> without settling these two.

## Staff

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-040 | The system knows which staff are currently on the clock | Captured | P |
| FR-041 | The system holds shift schedules | Inferred | the desktop admin renders "complex scheduling grids" |
| FR-042 | Clock-in and clock-out: which client, and whether it is enforced | Open | — |

## Revenue and reporting

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-050 | Live revenue is readable at any moment, from the POS manager view and the chat assistant | Captured | P |
| FR-051 | The desktop admin presents multi-column analytics | Captured | P |
| FR-052 | How current "live" has to be, and over what period revenue is measured | Open | see `NFR-004` |

## Manager access while off site

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-060 | The admin panel installs to a phone home screen with no app store deployment | Captured | P |
| FR-061 | On desktop it renders full data tables, analytics and scheduling grids | Captured | P |
| FR-062 | On mobile, tables collapse into touch-friendly summary cards and non-essential columns are hidden | Captured | P |
| FR-063 | One POS application serves both waiter and manager; JWT claims choose the view at login | Captured | P |
| FR-064 | The manager POS view shows live revenue, staff on the clock, and critical operational alerts | Captured | P |
| FR-065 | An actionable notification can be resolved from the lock screen without opening the app | Captured | P |
| FR-066 | Which events are critical enough to push | Open | — |

## Chat assistant

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-070 | The manager can query system state in natural language over a messaging platform | Captured | P ("What is the live revenue right now?") |
| FR-071 | The manager can issue commands in natural language | Captured | P ("86 the sweet potatoes") |
| FR-072 | Intent is resolved to a backend call | Captured | P |
| FR-073 | How the sender is authenticated, and bound to a user with a role | Open | **security-critical**; see note |
| FR-074 | Which messaging platform is supported first | Open | — |
| FR-075 | What the assistant does when intent is ambiguous or the item is not found | Open | — |

> **FR-073 matters more than it looks.** Everywhere else, authorisation comes
> from a JWT. A chat message carries a platform account id, which is a far weaker
> claim about who is asking — and `FR-071` lets that caller change menu state. A
> read-only first version avoids the problem entirely until identity binding is
> designed.

## Authentication and authorisation

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-080 | Users authenticate and receive a JWT whose claims carry their role | Captured | P |
| FR-081 | Access control is enforced from those claims, strictly | Captured | P |
| FR-082 | The permission matrix: every role against every action | Open | **blocks every service**; the source offered to map this and stopped |
| FR-083 | Where identity lives: an own auth service or an external provider | Open | — |
| FR-084 | Session lifetime, and whether a shared floor tablet is per-shift or per-person | Open | relates to `actors.md` |

## Customer self-ordering

Source `O` means the project owner, stated 2026-10-16.

| ID | Requirement | Status | Source |
| --- | --- | --- | --- |
| FR-100 | A seated customer can browse the menu on a tablet at their table | Captured | O |
| FR-101 | A customer can place an order for their table from that tablet, without a waiter | Captured | O |
| FR-102 | A customer-placed order is attributed to a specific table | Inferred | an order has to land somewhere; `FR-002` does this via the waiter's floor plan |
| FR-103 | A customer-placed order reaches the kitchen by the same path as a waiter-placed one | Inferred | otherwise the kitchen has two sources of truth |
| FR-104 | A customer is not offered items that are unavailable | Inferred | `FR-020` makes "86" a real state; letting a guest order an 86'd dish turns a kitchen problem into a customer-facing one |
| FR-105 | Whether a waiter confirms a customer's order before the kitchen sees it | Open | **shapes the whole flow**; see note |
| FR-106 | Whether the customer pays from the tablet, or the bill still goes through a cashier | Open | decides whether `FR-090` has a public surface |
| FR-107 | How a tablet is bound to its table, and who may rebind it | Open | `FR-102` depends on it |
| FR-108 | What a customer may see of their table's existing order, and whether they can amend or cancel it | Open | — |
| FR-109 | How the system handles a guest ordering on a tablet at a table that has not been seated | Open | — |

> **FR-105 is the one to answer first.** With waiter confirmation, the tablet is
> a faster notepad and almost nothing about the kitchen flow changes. Without it,
> a member of the public can put work into the kitchen directly — which needs
> rate limiting, a sanity check on quantities, and an answer for a guest who
> orders forty desserts for a joke. The two are different products, not two
> settings.

> **The customer is not a staff member with fewer permissions.** Every other
> actor arrives with a JWT identifying a person. A table tablet has no person
> behind it and sits unsupervised in a dining room. Extending the staff API with
> a guest role would put a public client on the same surface that voids bills;
> the self-ordering client needs its own narrow API scoped to one table's order.
> See [`actors.md`](actors.md#customer-dine-in) and `NFR-016`.

## Absent from the source entirely

Not gaps in the record — things the captured material never raises at all.
Listed because a restaurant system is normally expected to have them.

| ID | Area | Why it cannot stay unanswered |
| --- | --- | --- |
| FR-090 | **Payments** | Bills can be voided, so bills are paid somehow. No payment flow, provider, or card handling is specified — and `FR-106` may put this in front of the public. |
| FR-091 | **Fiscal and receipt obligations** | Many jurisdictions mandate receipt format, fiscal printers or tax reporting. This is a legal constraint, not a feature, and it is cheaper to design in than to retrofit. |
| FR-092 | **Reservations** | Tables and a floor plan are central, yet booking never appears. |
| FR-093 | **Multi-tenancy** | One restaurant or several? This changes every schema. |
| FR-094 | **Split bills, tips, discounts** | Standard restaurant billing, entirely unmentioned. |
| FR-095 | **Table transfers and merges** | Follows from having a floor plan. |
