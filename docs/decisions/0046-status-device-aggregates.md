# Server-side joint device aggregates for public status

## Status

Proposed
Class: architecture

## Context and Problem Statement

The status page needs a device → operating system → browser Sankey diagram. Separate dimension
totals cannot establish these relationships. The configured Umami instance offers a joint Breakdown
report, but the public share does not enable that report section.

## Decision Drivers

- Preserve real relationships and reconcile visitor counts within the same reporting window.
- Publish only predefined coarse categories and counts, never raw visitor records or arbitrary labels.
- Keep collection independent of browser traffic and of the forum runtime.
- Avoid broadening the existing public share's access.

## Considered Options

- Join the independent device, OS and browser marginal totals.
- Enable Breakdown on the public Umami share.
- Read joint reports with server-only credentials and persist an allowlisted aggregate.

## Decision Outcome

Use an optional read-authorized Umami account configured in Functions-only environment variables.
Reuse one login token within a scheduled collection without storing it. Query only the website
resolved by the configured share ID and the fixed device/OS/browser fields. Map raw labels to
predefined families, merge identical tuples, and persist only counts and reporting bounds.

Extend the snapshot architecture of [0027](0027-independent-status-netlify.md) with independent
device range, freshness and failure state. Credential rotation or removal invalidates device snapshot
identity without changing basic public traffic snapshots. Reject invalid counts; mark upstream
row limits or incomplete totals explicitly. Do not infer missing relationships or residual categories.
Device reports use a fifteen-second total budget including login and reconciliation, since thirty-day
joint reports need more time than basic counters. Other providers retain their eight-second budget.

## Pros and Cons of the Options

- Marginal totals: no additional access needed, but connections would be fabricated.
- Public report sharing: avoids account credentials, but exposes more raw report dimensions than the
  coarse public status projection needs and changes the operator's existing sharing boundary.
- Server aggregate: provides real joint counts with a narrow public response, at the cost of optional
  credentials, one additional provider adapter, and report API compatibility maintenance. Operators
  should use a dedicated read-only account instead of an administrator account.

## Links

- [Status product semantics](../product/server-status.md)
- [Functions configuration](../operations/status-netlify.md)
- [Umami authentication](https://docs.umami.is/docs/api/authentication)
- [Umami Breakdown report implementation](https://github.com/umami-software/umami/blob/v3.3.0/src/queries/sql/reports/getBreakdown.ts)
