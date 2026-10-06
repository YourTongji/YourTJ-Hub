# Persistent anonymous forum persona with a private account binding

## Status

Proposed
Class: feature

## Context and Problem Statement

Ordinary forum conversations need a recognizable anonymous author with its own public history.
Per-post masking cannot provide that history. A second login account would duplicate credentials,
posting limits, moderation and numeric OIDC identity. The selected naming policy uses every exact
THUOCL word, ten candidates per batch and ten batches per account/day, with a one-year name lock.
Readers and ordinary moderators must not receive the main-account binding, while restricted audit
operators can reveal it for a reason. Indefinite retention of binding and audits, including after
closure, is explicitly approved.

## Decision Drivers

- One persona per numeric account with the original authentication and accountability subject.
- Consistent public attribution across synchronous reads, asynchronous messages and exports.
- A database-enforced shared quota, idempotent recovery and calendar name lock across clients.
- Content-scoped governance without revealing its subject; separately granted audited reveal.
- PostgreSQL/SQLite compatibility and the existing single binary deployment.

## Considered Options

- Mask each post or topic without a persistent public identity.
- Create a second users row and independent anonymous login account.
- Keep a public persona plus a restricted one-to-one numeric account binding.
- Remove all bindings on closure or retain private binding/audits for governance continuity.
- Build a curated or length-filtered naming pool, or use the complete embedded THUOCL pool.

## Decision Outcome

Chosen: a public persona with a private one-to-one binding, indefinite restricted binding/audit
retention and the complete versioned THUOCL pool. Product rules belong in
[anonymous identity](../product/anonymous-identity.md); recovery belongs in
[operations](../operations/anonymous-identity.md).

The public UID and avatar seed are independent cryptographic random values. Numeric users remain
the account and quota subjects. Public DTOs carry a persona UID with numeric author ID zero and
never recover a missing author from the private binding. Owner locks serialize draws, confirmation
and persona writes; a server-persisted request key recovers a committed draw. Main profiles,
Following and public participation exclude persona ownership.

The explicit reveal capability does not inherit Admin's wildcard. Reveal and governance audit
in the same private transaction before exposing or changing the binding subject. Authenticated Web
pages suppress analytics, session replay and configured scripts so private identity choices and
reveals cannot enter that channel. Public exports cannot reconstruct private attribution and cannot
be used for persona recovery.

This extends ordinary forum publishing without replacing the legacy Wiki per-post masking in
[0013](0013-anonymous-wiki-comments.md) or course-review anonymity. It does not migrate their history.

## Pros and Cons of the Options

- Per-post masking: small surface, but no stable history, avatar or independent profile.
- Second account: reuses profiles, but duplicates login, quotas and permissions and risks account
  switch mistakes; it conflicts with the single accountability subject.
- Public persona/private binding: preserves accountability with independent presentation, but
  requires an explicit public projection in every outbound channel and restricted database access.
- Removing bindings on closure: minimizes retained association but prevents the approved audit and
  one-slot continuity. Indefinite retention preserves those guarantees but enlarges restricted
  evidence and backup responsibilities.
- Curated names: more predictable display, but violates the chosen unfiltered naming policy.
  The full pool includes long, symbolic and duplicate names; clients wrap candidates and UIDs
  distinguish people with identical names. Disabling authenticated tracking reduces visitor coverage.

## Links

- [Issue 1068: research, stories and acceptance](https://github.com/YourTongji/YourTJ-Hub/issues/1068).
- [THUOCL fixed source](https://github.com/thunlp/THUOCL/tree/a30ce79d895d01ab5132a5c74c29703ff7efb4cc).
- [Lark company-circle member guide](https://www.larksuite.com/hc/zh-CN/articles/360048488478-%E6%88%90%E5%91%98%E4%BD%BF%E7%94%A8%E5%85%AC%E5%8F%B8%E5%9C%88).
- [Discourse anonymous-mode documentation](https://meta.discourse.org/t/enable-and-configure-anonymous-mode/155638).
