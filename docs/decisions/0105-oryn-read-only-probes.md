# Run manual Oryn probes independently of receipt writers

## Status

Accepted
Class: process

## Context and Problem Statement

Oryn's shared workflow concurrency group protects receipt writers across planning, execution and
publication. A manual preflight or non-publishing review also joins that group, although it cannot
admit a reservation or run the publisher. A long automatic batch therefore blocks read-only runtime
verification and provider diagnosis behind work that the probe cannot modify.

## Decision Drivers

- Preserve serialization for workflows that can write receipts or publish results.
- Let an explicitly requested read-only probe verify a new pinned runtime without interrupting work.
- Keep the existing authority, source checks and publication gates.

## Considered Options

1. Give manual preflights and non-publishing dispatches a unique per-run concurrency group.
2. Keep all runs in the shared group.
3. Cancel a running automatic batch to make room for a probe.

## Decision Outcome

Choose option 1. Only `workflow_dispatch` with `preflight=true` or `publish!=true` uses the independent
probe group. A preflight does not execute model jobs. A non-publishing run skips admission and its
publisher remains disabled, including repair mode. Normal publishing dispatches and automatic events
keep `oryn-sweep`, queued concurrency and cancellation disabled. Release-note smoke retains its
existing independent group.

This changes scheduling only. It grants no new credentials or mutation authority. Explicitly
requested probes can use runner/model capacity concurrently with a batch; their existing task and
job budgets still apply.

## Pros and Cons of the Options

- Option 1 avoids queue contention for read-only verification while preserving the writer boundary;
  it permits explicit probes to consume resources concurrently.
- Option 2 minimizes concurrency distinctions, but makes a read-only check wait for an entire batch.
- Option 3 frees the queue, but interrupts unrelated authorized work and discards useful progress.

## Links

- [Oryn workflow](../../.github/workflows/oryn.yml)
- [Regression tests](../../scripts/test-oryn-validation.mjs)
- [Receipt ownership and retry decision](0025-oryn-action-reliability.md)
- [Queued probe reproduction](https://github.com/YourTongji/YourTJ-Hub/actions/runs/39822419828)
