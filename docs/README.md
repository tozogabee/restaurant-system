# docs

Project documentation. Code-level explanation stays in Go doc comments next to
the code; what lives here is the reasoning that does not fit in a comment.

| Path | What it holds |
| --- | --- |
| [`requirements/`](requirements/) | What the system has to do, for whom, and how well — with the gaps marked |
| [`architecture/`](architecture/) | How the system is built and why — the design overview and the decision record |
| [`preconception.md`](preconception.md) | The project's earliest thinking, and what it left undecided |

`preconception.md` is a historical record, deliberately not kept up to date. It
says what was envisaged before anything was built. When a question it leaves
open gets answered, the answer belongs in
[`requirements/`](requirements/) or
[`architecture/decisions/`](architecture/decisions/), not in an edit to the
preconception.

The split between the other two: `requirements/` says *what* and *how well*,
`architecture/` says *how* and *why that way*. A requirement that names a
technology is in the wrong file.
