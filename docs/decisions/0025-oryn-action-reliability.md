# Preserve Oryn admission ownership and report time across retries

## Status

Accepted
Class: process

## Context and Problem Statement

Queued events can outlive their target source or open state. Re-running an execution can reuse an
already settled plan, spend another model budget and then lose publication ownership. Report
serialization can fail after successful evidence gathering, and a long investigation can leave no
reporting time. Same-name artifacts across attempts make the intended publication input ambiguous.

## Decision Drivers

- Preserve source and command authority before execution and publication.
- Distinguish normal invalidation from retryable operational failure.
- Reserve reporting time without increasing the total task deadline.
- Keep publication credentials separate from the model job.

## Considered Options

1. Pin the published runtime fix and bind publication to its producing artifact ID.
2. Re-run old plans and choose artifacts by a shared name.
3. Disable freshness guards or increase the total timeout.

## Decision Outcome

Choose option 1. The pinned runtime assigns unique lease IDs, checks ownership before model work and
publication, and settles terminal receipts idempotently. Obsolete work produces skipped, superseded
or cancelled outcomes. Real provider, validation and exhausted report failures remain failures.

Reserve twenty percent of each invocation's existing deadline for up to two schema-validated report
attempts in its ephemeral Core session. The report phase exposes only a submission tool. Incomplete
evidence requires human review and cannot propose closure or automatic implementation. Diagnostics
retain bounded structural metadata rather than report text, reasoning or provider bodies.

Each item exposes its upload artifact ID to the publisher and names results by Actions attempt. A
full rerun replaces the plan artifact. Missing IDs fail before download, and outcome artifacts cannot
be published as reports. Fresh workflows replan inactive admissions; publication-only retries still
require current source and ownership. All receipt writers upgrade together, because historical strict
schemas cannot read extended receipts.

## Pros and Cons of the Options

- Option 1 preserves execution boundaries and useful failure evidence, at the cost of coordinated
  runtime upgrades and less investigation time for unusually large tasks.
- Option 2 repeats model cost without resolving ownership and can select obsolete terminal files.
- Option 3 can publish stale conclusions or spend longer without guaranteeing a usable report.

## Links

- [Operations guide](../operations/oryn.md)
- [Oryn implementation and Core regression fixtures](https://github.com/yzxoi/oryn-mini/pull/13)
- [Item workflow](../../.github/workflows/oryn-item.yml)
