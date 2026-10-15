# Decision log

Every durable choice with meaningful alternatives — architecture or process — lives in one place: this directory. Architecture and process decisions share the same format, lifecycle, and verifier. Specs own change contracts; commits own change history.

## Format

The format is **MADR (Markdown Any Decision Records)** with a documented `Class:` extension. Names and structure follow the industry standard so existing MADR tooling keeps working; the extension is ignored by tools that do not know it.

A decision record is a file named `NNNN-title.md` (4-digit zero-padded number, `-`, kebab-case title) directly inside this directory. Numbers are assigned as unique identifiers; gaps are allowed so multiple open PRs can reserve numbers independently. Once assigned, a number is the record's identity and never changes. Do not renumber existing records to close gaps.

### Required sections

Every record contains exactly these `##` sections, in this order:

1. `## Status`
2. `## Context and Problem Statement`
3. `## Decision Drivers`
4. `## Considered Options`
5. `## Decision Outcome`
6. `## Pros and Cons of the Options`
7. `## Links`

`## Links` may be empty ("None.") but must exist. Additional `##` sections may follow `## Links` when a record needs them.

### Status line

The first non-empty line under `## Status` is the status value. Valid values:

- `Proposed` — the decision is being considered; not yet shipped.
- `Accepted` — the decision shipped; the record describes what is.
- `Rejected` — considered and declined; kept while its rationale prevents a tempting, meaningful mistake.
- `Deprecated` — no longer recommended; superseded by a linked record.
- `Superseded by [NNNN](NNNN-title.md)` — replaced by a newer record; the target must exist.

New supersession statuses use the linked form. The verifier also accepts the exact legacy form `Superseded by NNNN` while its numbered target exists; other free-form or mismatched references are invalid.

A record whose status is `Accepted`, `Deprecated`, or `Superseded by …` describes current or frozen reality. It is never rewritten into a different decision: to change a decision, add a new record and mark the old one `Superseded by [NNNN](NNNN-title.md)`. Both records stay, cross-linked.

### Class line (extension)

The line immediately after the status value may be `Class: <value>`. It classifies the decision:

| Class | Covers |
|---|---|
| `architecture` | Structure of the shipped source: modules, boundaries, runtime vocabulary |
| `process` | Tooling, policy, workflow around the code: gates, package manager, conventions |
| `testing` | Test infrastructure and strategy |
| `feature` | A new user- or model-facing capability |
| `bug-fix` | Corrects a defect or closes a gap a postmortem surfaced |
| `simplification` | Removes code, behavior, or surface area without adding capability |

A missing `Class:` line is valid (it is an extension); an invalid value is a violation.

## Lifecycle

A decision starts `Proposed`. Once implemented, its status becomes `Accepted` and the record is kept current with what actually shipped (facts only — names, paths, structure — not the decision itself). A declined proposal is `Rejected`; delete a rejected record only when its rationale no longer prevents a plausible mistake. An obsolete accepted decision becomes `Superseded by [NNNN](NNNN-title.md)` or `Deprecated`; never edit it into its opposite.

## Writing rules

- Create or update a record when a change chooses among meaningful alternatives and future maintainers may reasonably revisit the rationale. Do not create a decision record for routine implementation, mechanical refactors, obvious fixes already defined by regression tests, or work that merely follows an accepted decision; risk-boundary feature behavior belongs in a spec and change history belongs in commits.
- State the decision, what it beats, and what it gives up. `## Considered Options` lists genuine alternatives; `## Pros and Cons of the Options` records why the losers lost. A decision without its alternatives invites re-litigation.
- When a record relies on external evidence or an implementation source, cite it descriptively in `## Links` with a stable URL, DOI, or versioned permalink. Do not leave research or quantitative claims unlinked.
- Cross-reference records with relative Markdown links (`[0001](0001-title.md)`), never bare numbers.
- Document current reality, not change history. Put change stories in commits; the decision record states the live contract.

## Reserving numbers in parallel PRs

- Before creating a MADR, check `dev` and open PRs for existing or reserved numbers. Reserve a distinct unused four-digit number in your PR or linked Issue (for example, `MADR 0084 reserved for #1171`) so other contributors can coordinate. A reservation is a coordination convention, not a committed placeholder record.
- A reserved number may be absent from `dev` until its PR merges; other decisions can merge first. Do not create empty placeholder MADRs or renumber another contributor's record to fill a gap.
- A reservation becomes active when first published in a PR or linked Issue. If concurrent PRs claim the same number, the earliest published claim keeps it; other PRs must choose another unused number and update their filenames and links before merging. If the order is unclear, maintainers resolve the collision before any affected PR merges. Reservations in unmerged branches are not automatically enforced by the verifier.
- If a proposed decision is abandoned, its reservation may be released explicitly in the Issue/PR; leaving a gap is valid.

## Verifier

`scripts/verify-decisions.mjs` enforces: file naming, unique numbering (gaps permitted for reserved or unmerged records), required sections, valid status values, valid `Class:` values when present, and `Superseded by` references whose targets exist. It exits non-zero on any violation. `scripts/run-gates.mjs` selects it from the manifest.

## Index

The numbered files in [this directory](./) are the complete, version-specific decision list.
New decisions do not require editing this README. Generate a sorted list with titles and current
statuses from the repository root:

```bash
node scripts/generate-decisions-index.mjs
```

The generator runs the decision verifier before rendering. Documentation CI publishes the list in
its job summary and the `decision-index` artifact; every record link is pinned to the checked-out
commit. Find runs in [CI / Verify](https://github.com/YourTongji/YourTJ-Hub/actions/workflows/ci.yml)
and select the run for the source commit you need. Artifacts are retained for seven days; the
command always reproduces the list from that checkout. Consumers of the former README list should
use this command or discover `NNNN-*.md` files directly, rather than parse this README.
