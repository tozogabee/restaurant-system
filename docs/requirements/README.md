# Requirements

What the system has to do, and for whom. Kept separate from
[`../architecture/`](../architecture/), which is *how* it will be done.

| Document | Holds |
| --- | --- |
| [`actors.md`](actors.md) | Who uses the system, and what each of them needs from it |
| [`functional.md`](functional.md) | What it must do — `FR-NNN` |
| [`non-functional.md`](non-functional.md) | How well it must do it — `NFR-NNN` |

## Where these come from

Almost everything here is extracted from
[`../preconception.md`](../preconception.md), which is a **partial** record —
only the final exchange of the original conversation survived. So this is not a
specification anyone signed off; it is the requirements that can be evidenced,
with the gaps left visible rather than filled in.

Every requirement therefore carries a status, and that status is the most
important column in the table:

| Status | Meaning |
| --- | --- |
| **Captured** | Stated in a source document. Safe to build against. |
| **Inferred** | Follows from something captured, but nobody said it. **Confirm before building.** |
| **Open** | Known to be needed, not specified. Blocks the work that depends on it. |

An `Inferred` requirement is a guess with a paper trail. Promote it to
`Captured` only when a human confirms it — not when it has merely been there a
while.

## Conventions

- IDs are permanent. A requirement that is dropped gets status `Withdrawn` and
  keeps its number; numbers are never reused, so a reference in a commit
  message or an ADR never goes stale.
- One requirement per row, testable as written. "The system is fast" is not a
  requirement; `NFR-003` is.
- No solutions here. "Voids above a threshold need manager approval" is a
  requirement; "the approval travels over FCM" is architecture, and belongs in
  an ADR.
- Keep the source column honest. If a row's only basis is an inference, say so.

## Relationship to the decision record

Requirements say what is needed; ADRs say what was chosen and what was given up.
When a requirement forces an architectural choice, link them both ways — the
requirement cites the ADR, the ADR's Context cites the requirement.

When an `Open` requirement gets answered, the answer lands here. If answering it
also commits the system to a design, that part goes in
[`../architecture/decisions/`](../architecture/decisions/).
