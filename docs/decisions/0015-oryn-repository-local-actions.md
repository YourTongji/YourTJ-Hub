# Run Oryn maintenance in repository-local Actions

## Status

Accepted
Class: process

## Context and Problem Statement

YourTJ-Hub needs App-authored maintenance reports with source evidence and native Issue/PR command
intake. The existing Oryn runtime is hosted in a separate private repository. Copying its implementation
would create a second maintenance burden; running centrally would require event forwarding or polling.

## Decision Drivers

- Keep execution, logs, credentials and operator policy with the YourTongji maintainers.
- Use native repository events without a webhook server or persisted model memory.
- Keep App write authority separate from model execution.
- Do not accept Bun runtime tests as validation of Go/Vue/Flutter/PostgreSQL application repairs.

## Considered Options

1. Repository-local Actions with a commit-pinned Oryn checkout and App installation tokens.
2. Central Oryn Actions with cross-repository polling or event forwarding.
3. Copy the complete Oryn implementation into this monorepo.

## Decision Outcome

Choose option 1. The local workflow checks out trusted policy from the default branch for native events,
loads an immutable private Oryn revision using a source-only read token, and separates planning,
read-only model execution and publication into jobs with independently narrowed target tokens.
The App is installed only on the target and runtime repositories. Native events replace webhooks.
Manual preflight exercises read access and planning without inference or publication.

Use GLM 5.3 Flash with explicit max reasoning, native image input and a configured one-million-token
context window. Enable all repository policy capabilities under existing per-item authority, freshness,
independent review and required CI gates. Install a trusted validation entry outside the candidate
checkout; select Go/Vue/Flutter/contract/PostgreSQL checks from changes relative to the exact job base.
A disposable PostgreSQL service and temporary Flutter cache support isolated repair validation.
Enable native event publication and six-hour sweeps; bounded batches and cooldown receipts cover backlog.
A successful live App preflight and model run establish access; local checks alone do not prove deployment.

## Pros and Cons of the Options

- Option 1 gives YourTongji native events and direct credential ownership; it requires a local entry,
  version maintenance and App installation on the private source repository as well as the target.
- Option 2 centralizes updates and model secrets, but moves operational ownership away from this repo
  and needs polling or another event transport.
- Option 3 removes cross-repository fetch credentials, but duplicates the runtime and security fixes.

## Links

- [Operations guide](../operations/oryn.md)
- [Pinned Oryn runtime](https://github.com/yzxoi/oryn-mini/tree/e36e678fae9a4d3e173750d9ce583c9140235b10)
- [GitHub App token action](https://github.com/actions/create-github-app-token/tree/bcd2ba49218906704ab6c1aa796996da409d3eb1)
