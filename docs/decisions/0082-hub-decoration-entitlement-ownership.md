# Hub owns decoration entitlements and equipment

## Status
Proposed
Class: architecture

## Context and Problem Statement

Avatar frames and nickname styles require one authoritative entitlement and equipment state across
Web and Flutter. Issue #1149 places these within a planned Mall service; #1156 requires a usable
administrator-granted decoration flow before that service is integrated. Maintaining separate Hub
and Mall inventories would make revocation, refunds and displayed appearance disagree.

## Decision Drivers

- The user explicitly selects Hub ownership and administrator-only grants for the initial flow.
- Preserve the embedded Go/Vue forum deployment and existing numeric identity source.
- Make grants, revocations and equipment recoverable and traceable across retries.
- Keep XP/trust qualification separate from permanent Credit ownership.

## Considered Options

- Hub owns entitlements/equipment; Mall owns products/orders and calls Hub's public boundary.
- Require Mall ownership and deployment before any decoration flow can operate.
- Maintain a temporary Hub inventory and an independent Mall inventory with later synchronization.

## Decision Outcome

Choose Hub ownership. Hub persists administrator-managed decoration definitions (the initial catalog
is empty), source-scoped entitlements,
equipment and transactional audit/idempotency receipts. Web and Flutter consume the effective
appearance computed by Hub. Administrator-only grants require no purchase or qualification system.
Trusted XP/trust/Credit adapters reuse the transaction-bound owner service; a Mall service must use
the owner API, with no direct cross-domain table writes and no second equipment state.

Request receipts prevent replaying a past grant after a later revoke. XP/trust source removal leaves
an independently active Credit source intact. Manual administrator revocation deliberately removes
all sources of the chosen style. Cosmetic appearance never conveys management permission.

## Pros and Cons of the Options

- Hub ownership provides one usable, atomic forum flow and reuses the identity boundary. It requires
  Mall to depend on a Hub entitlement API rather than owning every aspect of a digital product.
- Mall ownership matches the broader PRD draft but makes the basic cosmetic flow depend on an
  unimplemented commerce service and expands the requested implementation into payment/order work.
- Dual inventories unblock isolated prototypes but introduce reconciliation, split authority and
  refund/revocation races; they are unnecessary for this requirement.

## Links

- [Decoration requirement](https://github.com/YourTongji/YourTJ-Hub/issues/1156)
- [XP, Credit and Mall PRD](https://github.com/YourTongji/YourTJ-Hub/issues/1149)
- [Supported decoration behavior](../product/profile-decorations.md)
