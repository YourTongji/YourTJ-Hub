# Preserve Oryn failure evidence and budget long model requests

## Status

Accepted
Class: process

## Context and Problem Statement

Long max-reasoning reviews can hit a fixed request wall limit before exhausting the task budget.
Generic execution failure receipts hide the Core/Provider error needed to distinguish model errors,
cancellation and report-format failures.

## Decision Drivers

- Let healthy reasoning streams use the bounded task budget.
- Preserve useful failure evidence without retaining model histories or credentials.
- Keep task cancellation, source freshness and publication checks effective.

## Considered Options

1. Upgrade the pinned Oryn runtime to task-bounded calls and redacted failure artifacts.
2. Keep the short fixed request timeout and generic failure receipt.
3. Retain complete model session logs for troubleshooting.

## Decision Outcome

Choose option 1. Requests default to the remaining task budget, retaining first-byte and idle bounds.
An explicit repository variable may impose a shorter request wall limit. Operators remove the legacy
300-second override to adopt the default. The worker preserves selected Core/Provider errors and
budget/progress metadata before cleanup; CLI and workflow failure settlement retain that diagnostic.
Format failures include the attempt and output size. Raw model content and provider bodies are excluded.

## Pros and Cons of the Options

- Option 1 supports long reasoning and useful diagnostics while preserving total cost/time bounds.
- Option 2 is simpler but repeatedly interrupts useful work and conceals the reason.
- Option 3 captures more evidence but needlessly retains sensitive and reusable session content.

## Links

- [Operations guide](../operations/oryn.md)
- [Oryn implementation and fault fixtures](https://github.com/yzxoi/oryn-mini/pull/12)
